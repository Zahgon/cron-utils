// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package javatime

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// Zone is a time-zone: either a fixed offset from UTC (written Z, +hh:mm or +hh:mm:ss) or a region whose rules come
// from the IANA time-zone database (Europe/Paris), including the fixed regions UTC, GMT and UT and their offset forms
// such as UTC+01:00.
type Zone struct {
	id       string
	isOffset bool
	offset   int
	loc      *time.Location
	rules    *zoneRules
}

// zoneRules caches a region's offset history for the spans of the time line used so far: offsets[i] is in force
// from transitions[i-1] (or from lo) up to transitions[i] (or hi). The history is read from the time-zone database
// one year at a time, sampling each day and locating each change to the second.
type zoneRules struct {
	mu          sync.Mutex
	loc         *time.Location
	lo, hi      int64
	transitions []int64
	offsets     []int
}

const (
	daySeconds   = 86400
	chunkSeconds = 366 * daySeconds
)

// The instants whose offsets are cached: years 1000 to 3000. Earlier and later instants are looked up directly.
const (
	minCached = -30_610_224_000
	maxCached = 32_503_680_000
)

var rulesCache sync.Map

func rulesFor(id string, loc *time.Location) *zoneRules {
	r, _ := rulesCache.LoadOrStore(id, &zoneRules{loc: loc})
	return r.(*zoneRules)
}

func (r *zoneRules) offsetOf(t int64) int {
	_, off := time.Unix(t, 0).In(r.loc).Zone()
	return off
}

// changeIn returns the first second in (a, b] whose offset differs from the offset at a.
func (r *zoneRules) changeIn(a, b int64) int64 {
	before := r.offsetOf(a)
	for b-a > 1 {
		mid := a + (b-a)/2
		if r.offsetOf(mid) == before {
			a = mid
		} else {
			b = mid
		}
	}
	return b
}

// scan returns the transitions in (a, b] and the offsets from a onwards.
func (r *zoneRules) scan(a, b int64) ([]int64, []int) {
	offsets := []int{r.offsetOf(a)}
	var transitions []int64
	prevT, prev := a, offsets[0]
	for t := a + daySeconds; ; t += daySeconds {
		t = min(t, b)
		if o := r.offsetOf(t); o != prev {
			at := r.changeIn(prevT, t)
			transitions = append(transitions, at)
			offsets = append(offsets, r.offsetOf(at))
			prev = offsets[len(offsets)-1]
			prevT = at
			t = at
			continue
		}
		prevT = t
		if t >= b {
			return transitions, offsets
		}
	}
}

// ensure extends the cached history to cover the instants [from, to].
func (r *zoneRules) ensure(from, to int64) {
	if r.offsets == nil {
		r.lo = from - from%daySeconds - daySeconds
		r.hi = r.lo
		r.offsets = []int{r.offsetOf(r.lo)}
	}
	for to >= r.hi {
		transitions, offsets := r.scan(r.hi, r.hi+chunkSeconds)
		if offsets[0] != r.offsets[len(r.offsets)-1] {
			transitions = append([]int64{r.hi}, transitions...)
		} else {
			offsets = offsets[1:]
		}
		r.transitions = append(r.transitions, transitions...)
		r.offsets = append(r.offsets, offsets...)
		r.hi += chunkSeconds
	}
	for from < r.lo {
		transitions, offsets := r.scan(r.lo-chunkSeconds, r.lo)
		if offsets[len(offsets)-1] != r.offsets[0] {
			transitions = append(transitions, r.lo)
		} else {
			offsets = offsets[:len(offsets)-1]
		}
		r.transitions = append(transitions, r.transitions...)
		r.offsets = append(offsets, r.offsets...)
		r.lo -= chunkSeconds
	}
}

func (r *zoneRules) offsetAt(epochSecond int64) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensure(epochSecond, epochSecond)
	i := sort.Search(len(r.transitions), func(i int) bool { return r.transitions[i] > epochSecond })
	return r.offsets[i]
}

func (r *zoneRules) periods(from, to int64) []period {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensure(from, to)
	i := sort.Search(len(r.transitions), func(i int) bool { return r.transitions[i] > from })
	var out []period
	for ; ; i++ {
		p := period{start: math.MinInt64, end: math.MaxInt64, offset: r.offsets[i]}
		if i > 0 {
			p.start = r.transitions[i-1]
		}
		if i < len(r.transitions) {
			p.end = r.transitions[i]
		}
		out = append(out, p)
		if p.end == math.MaxInt64 || p.end > to {
			return out
		}
	}
}

// UTC is the zero offset, Z.
var UTC = Zone{id: "Z", isOffset: true}

// notRegions are names the Go time-zone database knows that are not region IDs of the IANA database proper.
var notRegions = map[string]bool{"Local": true, "Factory": true, "EST": true, "HST": true, "MST": true, "ROC": true}

var regionID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9~/._+-]+$`)

// ZoneOf returns the zone with the given ID: an offset (Z, +h, +hh, +hh:mm, +hhmm, +hh:mm:ss, +hhmmss), a
// prefixed offset (UTC, GMT or UT, optionally followed by an offset) or a region of the time-zone database.
func ZoneOf(id string) (Zone, error) {
	switch {
	case len(id) <= 1 || id[0] == '+' || id[0] == '-':
		return OffsetOf(id)
	case len(id) >= 3 && (id[:3] == "UTC" || id[:3] == "GMT"):
		return zoneWithPrefix(id, 3)
	case len(id) >= 2 && id[:2] == "UT":
		return zoneWithPrefix(id, 2)
	}
	return regionOf(id)
}

func zoneWithPrefix(id string, prefixLength int) (Zone, error) {
	prefix := id[:prefixLength]
	if len(id) == prefixLength {
		return Zone{id: prefix}, nil
	}
	if id[prefixLength] != '+' && id[prefixLength] != '-' {
		return regionOf(id)
	}
	offset, err := OffsetOf(id[prefixLength:])
	if err != nil {
		return Zone{}, cronerr.Wrap(cronerr.DateTime, err, "Invalid ID for offset-based ZoneId: %s", id)
	}
	if offset.offset != 0 {
		prefix += offset.id
	}
	return Zone{id: prefix, offset: offset.offset}, nil
}

func regionOf(id string) (Zone, error) {
	if !regionID.MatchString(id) {
		return Zone{}, cronerr.New(cronerr.DateTime, "Invalid ID for region-based ZoneId, invalid format: %s", id)
	}
	if notRegions[id] {
		return Zone{}, cronerr.New(cronerr.DateTime, "Unknown time-zone ID: %s", id)
	}
	loc, err := time.LoadLocation(id)
	if err != nil {
		return Zone{}, cronerr.Wrap(cronerr.DateTime, err, "Unknown time-zone ID: %s", id)
	}
	return Zone{id: id, loc: loc, rules: rulesFor(id, loc)}, nil
}

// OffsetOf parses a zone offset: Z, +h, +hh, +hh:mm, +hhmm, +hh:mm:ss or +hhmmss (a minus sign for offsets west
// of UTC), at most 18 hours.
func OffsetOf(id string) (Zone, error) {
	if id == "Z" {
		return UTC, nil
	}
	var hours, minutes, seconds int
	var err error
	invalid := func() (Zone, error) {
		return Zone{}, cronerr.New(cronerr.DateTime, "Invalid ID for ZoneOffset, invalid format: %s", id)
	}
	text := id
	switch len(text) {
	case 2:
		text = text[:1] + "0" + text[1:]
		fallthrough
	case 3:
		hours, err = parseOffsetNumber(id, text, 1, false)
	case 5:
		if hours, err = parseOffsetNumber(id, text, 1, false); err == nil {
			minutes, err = parseOffsetNumber(id, text, 3, false)
		}
	case 6:
		if hours, err = parseOffsetNumber(id, text, 1, false); err == nil {
			minutes, err = parseOffsetNumber(id, text, 4, true)
		}
	case 7:
		if hours, err = parseOffsetNumber(id, text, 1, false); err == nil {
			if minutes, err = parseOffsetNumber(id, text, 3, false); err == nil {
				seconds, err = parseOffsetNumber(id, text, 5, false)
			}
		}
	case 9:
		if hours, err = parseOffsetNumber(id, text, 1, false); err == nil {
			if minutes, err = parseOffsetNumber(id, text, 4, true); err == nil {
				seconds, err = parseOffsetNumber(id, text, 7, true)
			}
		}
	default:
		return invalid()
	}
	if err != nil {
		return Zone{}, err
	}
	switch text[0] {
	case '-':
		return offsetOfHoursMinutesSeconds(-hours, -minutes, -seconds)
	case '+':
		return offsetOfHoursMinutesSeconds(hours, minutes, seconds)
	}
	return Zone{}, cronerr.New(cronerr.DateTime, "Invalid ID for ZoneOffset, plus/minus not found when expected: %s", id)
}

func parseOffsetNumber(id, text string, pos int, precededByColon bool) (int, error) {
	if precededByColon && text[pos-1] != ':' {
		return 0, cronerr.New(cronerr.DateTime, "Invalid ID for ZoneOffset, colon not found when expected: %s", id)
	}
	ch1, ch2 := text[pos], text[pos+1]
	if ch1 < '0' || ch1 > '9' || ch2 < '0' || ch2 > '9' {
		return 0, cronerr.New(cronerr.DateTime, "Invalid ID for ZoneOffset, non numeric characters found: %s", id)
	}
	return int(ch1-'0')*10 + int(ch2-'0'), nil
}

func offsetOfHoursMinutesSeconds(hours, minutes, seconds int) (Zone, error) {
	if hours < -18 || hours > 18 {
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset hours not in valid range: value %d is not in the range -18 to 18", hours)
	}
	switch {
	case hours > 0 && (minutes < 0 || seconds < 0):
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset minutes and seconds must be positive because hours is positive")
	case hours < 0 && (minutes > 0 || seconds > 0):
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset minutes and seconds must be negative because hours is negative")
	case (minutes > 0 && seconds < 0) || (minutes < 0 && seconds > 0):
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset minutes and seconds must have the same sign")
	}
	if minutes < -59 || minutes > 59 {
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset minutes not in valid range: value %d is not in the range -59 to 59", minutes)
	}
	if seconds < -59 || seconds > 59 {
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset seconds not in valid range: value %d is not in the range -59 to 59", seconds)
	}
	if (hours == 18 || hours == -18) && (minutes != 0 || seconds != 0) {
		return Zone{}, cronerr.New(cronerr.DateTime, "Zone offset not in valid range: -18:00 to +18:00")
	}
	return offsetZone(hours*secondsPerHour + minutes*secondsPerMinute + seconds), nil
}

// offsetZone returns the fixed-offset zone of the given total seconds.
func offsetZone(total int) Zone {
	return Zone{id: offsetID(total), isOffset: true, offset: total}
}

// offsetID formats an offset as Z, +hh:mm or +hh:mm:ss.
func offsetID(total int) string {
	if total == 0 {
		return "Z"
	}
	abs := total
	sign := "+"
	if total < 0 {
		abs, sign = -total, "-"
	}
	h, m, s := abs/secondsPerHour, abs/secondsPerMinute%60, abs%60
	if s != 0 {
		return fmt.Sprintf("%s%02d:%02d:%02d", sign, h, m, s)
	}
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}

// ID returns the zone's ID as it is written in a zoned date-time.
func (z Zone) ID() string { return z.id }

// IsOffset reports whether the zone is a plain offset rather than a region.
func (z Zone) IsOffset() bool { return z.isOffset }

// Equal reports whether two zones are the same zone: the same offset, or the same region ID.
func (z Zone) Equal(o Zone) bool {
	return z.isOffset == o.isOffset && z.id == o.id
}

// String returns the zone's ID.
func (z Zone) String() string { return z.id }

// offsetAt returns the offset in seconds in force at the given instant.
func (z Zone) offsetAt(epochSecond int64) int {
	if z.loc == nil {
		return z.offset
	}
	if z.rules != nil && epochSecond > minCached && epochSecond < maxCached {
		return z.rules.offsetAt(epochSecond)
	}
	_, off := time.Unix(epochSecond, 0).In(z.loc).Zone()
	return off
}

// period is a span of the time line [start, end) with one offset.
type period struct {
	start, end int64
	offset     int
}

// periods lists the offset periods that intersect the instants [from, to].
func (z Zone) periods(from, to int64) []period {
	if z.rules != nil && from > minCached && to < maxCached {
		return z.rules.periods(from, to)
	}
	var out []period
	t := from
	for {
		at := time.Unix(t, 0).In(z.loc)
		_, off := at.Zone()
		startTime, endTime := at.ZoneBounds()
		p := period{start: math.MinInt64, end: math.MaxInt64, offset: off}
		if !startTime.IsZero() {
			p.start = startTime.Unix()
		}
		if !endTime.IsZero() {
			p.end = endTime.Unix()
		}
		if n := len(out); n > 0 && out[n-1].offset == p.offset {
			out[n-1].end = p.end
		} else {
			out = append(out, p)
		}
		if p.end == math.MaxInt64 || p.end > to {
			return out
		}
		t = max(p.end, t+1)
	}
}

// resolution describes how a local date-time maps onto the zone's time line.
type resolution struct {
	valid []int
	// For a local date-time inside a gap: the offsets before and after the transition.
	gapBefore, gapAfter int
}

// resolve finds the valid offsets of a local date-time: one normally, two in an overlap (the earlier offset first)
// and none in a gap.
func (z Zone) resolve(dt LocalDateTime) resolution {
	if z.loc == nil {
		return resolution{valid: []int{z.offset}}
	}
	local := dt.epochSecondUTC()
	const window = 18 * secondsPerHour
	ps := z.periods(local-window, local+window)
	var r resolution
	for _, p := range ps {
		if inst := local - int64(p.offset); inst >= p.start && inst < p.end {
			r.valid = append(r.valid, p.offset)
		}
	}
	if len(r.valid) > 0 {
		return r
	}
	for i := 0; i+1 < len(ps); i++ {
		boundary := ps[i].end
		if local >= boundary+int64(ps[i].offset) && local < boundary+int64(ps[i+1].offset) {
			r.gapBefore, r.gapAfter = ps[i].offset, ps[i+1].offset
			return r
		}
	}
	r.valid = []int{z.offsetAt(local)}
	return r
}
