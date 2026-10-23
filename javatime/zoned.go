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
	"strings"
	"time"
)

// ZonedDateTime is a local date-time with the offset in force and the zone it belongs to.
//
// Date-based changes (setting a field, truncating, adding days, months or years) work on the local date-time and
// then resolve it against the zone: a local time that falls into a gap moves forward by the length of the gap, and a
// local time that falls into an overlap keeps the current offset when it is one of the two, otherwise it takes the
// earlier offset. Time-based changes (adding hours, minutes, seconds or nanoseconds) work on the instant.
type ZonedDateTime struct {
	dt     LocalDateTime
	offset int
	zone   Zone
}

// Of returns the zoned date-time of a local date-time in zone, taking the earlier offset in an overlap.
func Of(dt LocalDateTime, zone Zone) ZonedDateTime {
	return ofLocal(dt, zone, 0, false)
}

// OfDateTime returns the zoned date-time of a date and a time in zone, taking the earlier offset in an overlap.
func OfDateTime(date LocalDate, t LocalTime, zone Zone) ZonedDateTime {
	return Of(LocalDateTime{date, t}, zone)
}

// OfFields returns the zoned date-time of the given fields in zone, or an error when a field is out of range or the
// date does not exist.
func OfFields(year, month, day, hour, minute, second, nano int, zone Zone) (ZonedDateTime, error) {
	date, err := DateOf(year, month, day)
	if err != nil {
		return ZonedDateTime{}, err
	}
	t, err := TimeOfNano(hour, minute, second, nano)
	if err != nil {
		return ZonedDateTime{}, err
	}
	return OfDateTime(date, t, zone), nil
}

// OfInstant returns the zoned date-time of an instant in zone.
func OfInstant(epochSecond int64, nano int, zone Zone) ZonedDateTime {
	offset := zone.offsetAt(epochSecond)
	return ZonedDateTime{dateTimeOfEpochSecond(epochSecond, nano, offset), offset, zone}
}

// FromTime returns the zoned date-time of t in zone.
func FromTime(t time.Time, zone Zone) ZonedDateTime {
	return OfInstant(t.Unix(), t.Nanosecond(), zone)
}

func ofLocal(dt LocalDateTime, zone Zone, preferred int, hasPreferred bool) ZonedDateTime {
	r := zone.resolve(dt)
	switch len(r.valid) {
	case 0:
		dt = dt.plusSeconds(int64(r.gapAfter - r.gapBefore))
		return ZonedDateTime{dt, r.gapAfter, zone}
	case 1:
		return ZonedDateTime{dt, r.valid[0], zone}
	}
	if hasPreferred {
		for _, o := range r.valid {
			if o == preferred {
				return ZonedDateTime{dt, o, zone}
			}
		}
	}
	return ZonedDateTime{dt, r.valid[0], zone}
}

func (z ZonedDateTime) resolveLocal(dt LocalDateTime) ZonedDateTime {
	return ofLocal(dt, z.zone, z.offset, true)
}

// LocalDateTime returns the local date-time.
func (z ZonedDateTime) LocalDateTime() LocalDateTime { return z.dt }

// Date returns the local date.
func (z ZonedDateTime) Date() LocalDate { return z.dt.date }

// Zone returns the zone.
func (z ZonedDateTime) Zone() Zone { return z.zone }

// Offset returns the offset from UTC in seconds.
func (z ZonedDateTime) Offset() int { return z.offset }

// Year returns the year.
func (z ZonedDateTime) Year() int { return z.dt.date.year }

// Month returns the month, 1 to 12.
func (z ZonedDateTime) Month() int { return z.dt.date.month }

// Day returns the day of the month.
func (z ZonedDateTime) Day() int { return z.dt.date.day }

// DayOfWeek returns the ISO day of the week, 1 for Monday to 7 for Sunday.
func (z ZonedDateTime) DayOfWeek() int { return z.dt.date.DayOfWeek() }

// Hour returns the hour of the day.
func (z ZonedDateTime) Hour() int { return z.dt.time.hour }

// Minute returns the minute of the hour.
func (z ZonedDateTime) Minute() int { return z.dt.time.minute }

// Second returns the second of the minute.
func (z ZonedDateTime) Second() int { return z.dt.time.second }

// Nano returns the nanosecond of the second.
func (z ZonedDateTime) Nano() int { return z.dt.time.nano }

// Get returns the value of the given time unit's field: the hour of the day, the minute of the hour or the second of
// the minute.
func (z ZonedDateTime) Get(unit Unit) int {
	switch unit {
	case Hours:
		return z.Hour()
	case Minutes:
		return z.Minute()
	case Seconds:
		return z.Second()
	case Days:
		return z.Day()
	}
	return z.Nano()
}

// EpochSecond returns the instant in seconds from 1970-01-01T00:00Z.
func (z ZonedDateTime) EpochSecond() int64 {
	return z.dt.toEpochSecond(z.offset)
}

// Time returns the instant as a time.Time.
func (z ZonedDateTime) Time() time.Time {
	return time.Unix(z.EpochSecond(), int64(z.Nano()))
}

// plusInstant moves the instant by seconds and nanoseconds.
func (z ZonedDateTime) plusInstant(seconds, nanos int64) ZonedDateTime {
	sec := z.EpochSecond() + seconds + floorDiv(nanos, nanosPerSecond)
	nano := int64(z.Nano()) + floorMod(nanos, nanosPerSecond)
	sec += nano / nanosPerSecond
	nano %= nanosPerSecond
	return OfInstant(sec, int(nano), z.zone)
}

// PlusSeconds moves the instant by n seconds.
func (z ZonedDateTime) PlusSeconds(n int64) ZonedDateTime { return z.plusInstant(n, 0) }

// MinusSeconds moves the instant back by n seconds.
func (z ZonedDateTime) MinusSeconds(n int64) ZonedDateTime { return z.plusInstant(-n, 0) }

// PlusMinutes moves the instant by n minutes.
func (z ZonedDateTime) PlusMinutes(n int64) ZonedDateTime {
	return z.plusInstant(n*secondsPerMinute, 0)
}

// PlusHours moves the instant by n hours.
func (z ZonedDateTime) PlusHours(n int64) ZonedDateTime { return z.plusInstant(n*secondsPerHour, 0) }

// PlusNanos moves the instant by n nanoseconds.
func (z ZonedDateTime) PlusNanos(n int64) ZonedDateTime { return z.plusInstant(0, n) }

// MinusNanos moves the instant back by n nanoseconds.
func (z ZonedDateTime) MinusNanos(n int64) ZonedDateTime { return z.plusInstant(0, -n) }

// Plus moves the date-time by n units: days on the local time line, smaller units on the instant time line.
func (z ZonedDateTime) Plus(n int64, unit Unit) ZonedDateTime {
	switch unit {
	case Days:
		return z.PlusDays(n)
	case Hours:
		return z.PlusHours(n)
	case Minutes:
		return z.PlusMinutes(n)
	case Seconds:
		return z.PlusSeconds(n)
	}
	return z.PlusNanos(n)
}

// PlusDays moves the local date by n days.
func (z ZonedDateTime) PlusDays(n int64) ZonedDateTime {
	return z.resolveLocal(LocalDateTime{z.dt.date.PlusDays(n), z.dt.time})
}

// PlusMonths moves the local date by n months, clamping the day to the month's length.
func (z ZonedDateTime) PlusMonths(n int64) ZonedDateTime {
	return z.resolveLocal(LocalDateTime{z.dt.date.PlusMonths(n), z.dt.time})
}

// MinusMonths moves the local date back by n months, clamping the day to the month's length.
func (z ZonedDateTime) MinusMonths(n int64) ZonedDateTime { return z.PlusMonths(-n) }

// PlusYears moves the local date by n years.
func (z ZonedDateTime) PlusYears(n int64) ZonedDateTime {
	return z.resolveLocal(LocalDateTime{z.dt.date.PlusYears(n), z.dt.time})
}

// MinusYears moves the local date back by n years.
func (z ZonedDateTime) MinusYears(n int64) ZonedDateTime { return z.PlusYears(-n) }

// WithMonth sets the month, clamping the day to the month's length.
func (z ZonedDateTime) WithMonth(month int) (ZonedDateTime, error) {
	date, err := z.dt.date.WithMonth(month)
	if err != nil {
		return ZonedDateTime{}, err
	}
	return z.resolveLocal(LocalDateTime{date, z.dt.time}), nil
}

// WithDayOfMonth sets the day of the month, or reports an error when that day does not exist.
func (z ZonedDateTime) WithDayOfMonth(day int) (ZonedDateTime, error) {
	date, err := z.dt.date.WithDayOfMonth(day)
	if err != nil {
		return ZonedDateTime{}, err
	}
	return z.resolveLocal(LocalDateTime{date, z.dt.time}), nil
}

// WithLastDayOfMonth sets the day to the last day of the month.
func (z ZonedDateTime) WithLastDayOfMonth() ZonedDateTime {
	return z.resolveLocal(LocalDateTime{z.dt.date.WithLastDayOfMonth(), z.dt.time})
}

// WithFirstDayOfMonth sets the day to the first day of the month.
func (z ZonedDateTime) WithFirstDayOfMonth() ZonedDateTime {
	return z.resolveLocal(LocalDateTime{LocalDate{z.dt.date.year, z.dt.date.month, 1}, z.dt.time})
}

// WithTime sets the local time.
func (z ZonedDateTime) WithTime(t LocalTime) ZonedDateTime {
	return z.resolveLocal(LocalDateTime{z.dt.date, t})
}

// TruncatedTo drops the fields smaller than unit from the local time.
func (z ZonedDateTime) TruncatedTo(unit Unit) ZonedDateTime {
	return z.resolveLocal(z.dt.truncatedTo(unit))
}

// Equal reports whether both have the same local date-time, offset and zone.
func (z ZonedDateTime) Equal(o ZonedDateTime) bool {
	return z.dt == o.dt && z.offset == o.offset && z.zone.Equal(o.zone)
}

// Compare orders by instant, then by local date-time, then by zone ID.
func (z ZonedDateTime) Compare(o ZonedDateTime) int {
	if a, b := z.EpochSecond(), o.EpochSecond(); a != b {
		if a < b {
			return -1
		}
		return 1
	}
	if c := sign(z.Nano() - o.Nano()); c != 0 {
		return c
	}
	if c := z.dt.Compare(o.dt); c != 0 {
		return c
	}
	return strings.Compare(z.zone.id, o.zone.id)
}

// IsBefore reports whether z is an earlier instant than o.
func (z ZonedDateTime) IsBefore(o ZonedDateTime) bool {
	a, b := z.EpochSecond(), o.EpochSecond()
	return a < b || (a == b && z.Nano() < o.Nano())
}

// CompareOffsets compares two offsets the way zone offsets are ordered: a larger offset (further east) sorts first,
// so the result is positive when a is west of b.
func CompareOffsets(a, b int) int {
	return b - a
}

// String returns the ISO-8601 form, e.g. 2024-03-10T03:00-04:00[America/New_York]; a zone that is a plain offset is
// not repeated in brackets.
func (z ZonedDateTime) String() string {
	s := z.dt.String() + offsetID(z.offset)
	if !z.zone.isOffset {
		s += "[" + z.zone.id + "]"
	}
	return s
}
