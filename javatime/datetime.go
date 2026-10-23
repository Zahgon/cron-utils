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
	"strings"
)

// Unit is a unit a date-time can be truncated to or moved by.
type Unit int

// The units of a date-time.
const (
	Nanos Unit = iota
	Seconds
	Minutes
	Hours
	Days
)

// LocalTime is a time of day without a time-zone.
type LocalTime struct {
	hour, minute, second, nano int
}

// Midnight is the start of the day.
var Midnight = LocalTime{}

// MaxSeconds is the last second of the day, 23:59:59.
var MaxSeconds = LocalTime{23, 59, 59, 0}

// TimeOf returns the time hour:minute:second, or an error when a field is out of range.
func TimeOf(hour, minute, second int) (LocalTime, error) {
	return TimeOfNano(hour, minute, second, 0)
}

// TimeOfNano returns the time hour:minute:second.nano, or an error when a field is out of range.
func TimeOfNano(hour, minute, second, nano int) (LocalTime, error) {
	if err := fieldHour.check(int64(hour)); err != nil {
		return LocalTime{}, err
	}
	if err := fieldMinute.check(int64(minute)); err != nil {
		return LocalTime{}, err
	}
	if err := fieldSecond.check(int64(second)); err != nil {
		return LocalTime{}, err
	}
	if err := fieldNano.check(int64(nano)); err != nil {
		return LocalTime{}, err
	}
	return LocalTime{hour, minute, second, nano}, nil
}

func timeOfNanoOfDay(nanoOfDay int64) LocalTime {
	h := nanoOfDay / (secondsPerHour * nanosPerSecond)
	nanoOfDay -= h * secondsPerHour * nanosPerSecond
	m := nanoOfDay / (secondsPerMinute * nanosPerSecond)
	nanoOfDay -= m * secondsPerMinute * nanosPerSecond
	s := nanoOfDay / nanosPerSecond
	nanoOfDay -= s * nanosPerSecond
	return LocalTime{int(h), int(m), int(s), int(nanoOfDay)}
}

// Hour returns the hour of the day.
func (t LocalTime) Hour() int { return t.hour }

// Minute returns the minute of the hour.
func (t LocalTime) Minute() int { return t.minute }

// Second returns the second of the minute.
func (t LocalTime) Second() int { return t.second }

// Nano returns the nanosecond of the second.
func (t LocalTime) Nano() int { return t.nano }

func (t LocalTime) secondOfDay() int64 {
	return int64(t.hour)*secondsPerHour + int64(t.minute)*secondsPerMinute + int64(t.second)
}

func (t LocalTime) truncatedTo(unit Unit) LocalTime {
	switch unit {
	case Days:
		return Midnight
	case Hours:
		return LocalTime{t.hour, 0, 0, 0}
	case Minutes:
		return LocalTime{t.hour, t.minute, 0, 0}
	case Seconds:
		return LocalTime{t.hour, t.minute, t.second, 0}
	}
	return t
}

func (t LocalTime) compare(o LocalTime) int {
	switch {
	case t.hour != o.hour:
		return sign(t.hour - o.hour)
	case t.minute != o.minute:
		return sign(t.minute - o.minute)
	case t.second != o.second:
		return sign(t.second - o.second)
	default:
		return sign(t.nano - o.nano)
	}
}

// String returns the time as HH:mm, HH:mm:ss or HH:mm:ss with a fraction of three, six or nine digits: the seconds
// appear only when they or the nanoseconds are not zero.
func (t LocalTime) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%02d:%02d", t.hour, t.minute)
	if t.second > 0 || t.nano > 0 {
		fmt.Fprintf(&b, ":%02d", t.second)
		switch {
		case t.nano == 0:
		case t.nano%1_000_000 == 0:
			fmt.Fprintf(&b, ".%03d", t.nano/1_000_000)
		case t.nano%1000 == 0:
			fmt.Fprintf(&b, ".%06d", t.nano/1000)
		default:
			fmt.Fprintf(&b, ".%09d", t.nano)
		}
	}
	return b.String()
}

// LocalDateTime is a date and a time without a time-zone.
type LocalDateTime struct {
	date LocalDate
	time LocalTime
}

// DateTimeOf combines a date and a time.
func DateTimeOf(date LocalDate, time LocalTime) LocalDateTime {
	return LocalDateTime{date, time}
}

// Date returns the date part.
func (dt LocalDateTime) Date() LocalDate { return dt.date }

// Time returns the time part.
func (dt LocalDateTime) Time() LocalTime { return dt.time }

// epochSecondUTC returns the seconds from 1970-01-01T00:00 to dt, reading dt as UTC.
func (dt LocalDateTime) epochSecondUTC() int64 {
	return dt.date.EpochDay()*secondsPerDay + dt.time.secondOfDay()
}

// toEpochSecond returns the instant of dt at the given offset in seconds.
func (dt LocalDateTime) toEpochSecond(offset int) int64 {
	return dt.epochSecondUTC() - int64(offset)
}

func dateTimeOfEpochSecond(epochSecond int64, nano int, offset int) LocalDateTime {
	local := epochSecond + int64(offset)
	day := floorDiv(local, secondsPerDay)
	secs := floorMod(local, secondsPerDay)
	return LocalDateTime{DateOfEpochDay(day), timeOfNanoOfDay(secs*nanosPerSecond + int64(nano))}
}

// plusSeconds adds seconds on the local time line.
func (dt LocalDateTime) plusSeconds(seconds int64) LocalDateTime {
	if seconds == 0 {
		return dt
	}
	return dateTimeOfEpochSecond(dt.epochSecondUTC()+seconds, dt.time.nano, 0)
}

func (dt LocalDateTime) truncatedTo(unit Unit) LocalDateTime {
	return LocalDateTime{dt.date, dt.time.truncatedTo(unit)}
}

// Compare returns -1, 0 or 1 as dt is before, equal to or after o.
func (dt LocalDateTime) Compare(o LocalDateTime) int {
	if c := dt.date.Compare(o.date); c != 0 {
		return c
	}
	return dt.time.compare(o.time)
}

// String returns the date-time as yyyy-MM-ddTHH:mm[:ss[.fraction]].
func (dt LocalDateTime) String() string {
	return dt.date.String() + "T" + dt.time.String()
}
