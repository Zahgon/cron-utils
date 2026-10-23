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
	"strconv"
	"strings"
	"time"
)

// Duration is an amount of time in seconds plus a non-negative nanosecond adjustment.
type Duration struct {
	seconds int64
	nanos   int
}

// Between returns the time from start to end on the instant time line; negative when end is earlier.
func Between(start, end ZonedDateTime) Duration {
	secs := end.EpochSecond() - start.EpochSecond()
	nanos := int64(end.Nano() - start.Nano())
	secs += floorDiv(nanos, nanosPerSecond)
	return Duration{secs, int(floorMod(nanos, nanosPerSecond))}
}

// Seconds returns the whole seconds, rounded towards negative infinity.
func (d Duration) Seconds() int64 { return d.seconds }

// Nanos returns the nanosecond adjustment, 0 to 999,999,999.
func (d Duration) Nanos() int { return d.nanos }

// ToHours returns the number of whole hours.
func (d Duration) ToHours() int64 { return d.seconds / secondsPerHour }

// ToDays returns the number of whole days.
func (d Duration) ToDays() int64 { return d.seconds / secondsPerDay }

// Std returns the duration as a time.Duration.
func (d Duration) Std() time.Duration {
	return time.Duration(d.seconds)*time.Second + time.Duration(d.nanos)
}

// String returns the ISO-8601 form PTnHnMn.nS, e.g. PT8H6M12.345S; PT0S for zero.
func (d Duration) String() string {
	if d.seconds == 0 && d.nanos == 0 {
		return "PT0S"
	}
	total := d.seconds
	if d.seconds < 0 && d.nanos > 0 {
		total++
	}
	hours := total / secondsPerHour
	minutes := (total % secondsPerHour) / secondsPerMinute
	secs := total % secondsPerMinute
	var b strings.Builder
	b.WriteString("PT")
	if hours != 0 {
		b.WriteString(strconv.FormatInt(hours, 10) + "H")
	}
	if minutes != 0 {
		b.WriteString(strconv.FormatInt(minutes, 10) + "M")
	}
	if secs == 0 && d.nanos == 0 && b.Len() > 2 {
		return b.String()
	}
	if d.seconds < 0 && d.nanos > 0 && secs == 0 {
		b.WriteString("-0")
	} else {
		b.WriteString(strconv.FormatInt(secs, 10))
	}
	if d.nanos > 0 {
		var frac int64
		if d.seconds < 0 {
			frac = 2*nanosPerSecond - int64(d.nanos)
		} else {
			frac = int64(d.nanos) + nanosPerSecond
		}
		digits := strings.TrimRight(strconv.FormatInt(frac, 10), "0")
		b.WriteString("." + digits[1:])
	}
	b.WriteString("S")
	return b.String()
}
