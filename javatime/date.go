// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package javatime models dates and zoned date-times on the ISO-8601 proleptic calendar with the semantics cron
// schedules are computed with: local date-times resolved against a time-zone's rules (a local time in a gap moves
// forward by the length of the gap, a local time in an overlap keeps its offset when it can and otherwise takes the
// earlier one), date-based arithmetic that clamps to the end of the month, time-based arithmetic on the instant time
// line, and the ISO-8601 text forms of dates, offsets, zoned date-times and durations.
package javatime

import (
	"fmt"
	"strconv"

	"github.com/jmrozanec/cron-utils/cronerr"
)

const (
	minYear          = -999_999_999
	maxYear          = 999_999_999
	daysPerCycle     = 146097
	days0000To1970   = daysPerCycle*5 - (30*365 + 7)
	secondsPerMinute = 60
	secondsPerHour   = 3600
	secondsPerDay    = 86400
	nanosPerSecond   = 1_000_000_000
)

var monthNames = [...]string{"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE", "JULY", "AUGUST",
	"SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER"}

// field describes a date-time field with its valid range, for range-check error messages.
type field struct {
	name                                             string
	minSmallest, minLargest, maxSmallest, maxLargest int64
}

var (
	fieldYear        = field{"Year", minYear, minYear, maxYear, maxYear}
	fieldMonth       = field{"MonthOfYear", 1, 1, 12, 12}
	fieldDayOfMonth  = field{"DayOfMonth", 1, 1, 28, 31}
	fieldDayOfYear   = field{"DayOfYear", 1, 1, 365, 366}
	fieldHour        = field{"HourOfDay", 0, 0, 23, 23}
	fieldMinute      = field{"MinuteOfHour", 0, 0, 59, 59}
	fieldSecond      = field{"SecondOfMinute", 0, 0, 59, 59}
	fieldNano        = field{"NanoOfSecond", 0, 0, 999_999_999, 999_999_999}
	fieldDayOfWeek   = field{"DayOfWeek", 1, 1, 7, 7}
	fieldMonthOfYear = fieldMonth
)

func (f field) check(value int64) error {
	if value >= f.minSmallest && value <= f.maxLargest {
		return nil
	}
	rng := strconv.FormatInt(f.minSmallest, 10)
	if f.minSmallest != f.minLargest {
		rng += "/" + strconv.FormatInt(f.minLargest, 10)
	}
	rng += " - " + strconv.FormatInt(f.maxSmallest, 10)
	if f.maxSmallest != f.maxLargest {
		rng += "/" + strconv.FormatInt(f.maxLargest, 10)
	}
	return cronerr.New(cronerr.DateTime, "Invalid value for %s (valid values %s): %d", f.name, rng, value)
}

// CheckDayOfWeek reports an error when value is not an ISO day-of-week number (1 for Monday to 7 for Sunday).
func CheckDayOfWeek(value int) error {
	return fieldDayOfWeek.check(int64(value))
}

// CheckMonth reports an error when value is not a month number (1 to 12).
func CheckMonth(value int) error {
	return fieldMonthOfYear.check(int64(value))
}

// IsLeapYear reports whether year is a leap year of the proleptic Gregorian calendar.
func IsLeapYear(year int) bool {
	return year&3 == 0 && (year%100 != 0 || year%400 == 0)
}

// MonthLength returns the number of days of month in year.
func MonthLength(year, month int) int {
	switch month {
	case 2:
		if IsLeapYear(year) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

// LocalDate is a date without a time-zone.
type LocalDate struct {
	year, month, day int
}

// DateOf returns the date year-month-day, or an error when a field is out of range or the day does not exist.
func DateOf(year, month, day int) (LocalDate, error) {
	if err := fieldYear.check(int64(year)); err != nil {
		return LocalDate{}, err
	}
	if err := fieldMonth.check(int64(month)); err != nil {
		return LocalDate{}, err
	}
	if err := fieldDayOfMonth.check(int64(day)); err != nil {
		return LocalDate{}, err
	}
	if day > 28 && day > MonthLength(year, month) {
		if day == 29 {
			return LocalDate{}, cronerr.New(cronerr.DateTime, "Invalid date 'February 29' as '%d' is not a leap year", year)
		}
		return LocalDate{}, cronerr.New(cronerr.DateTime, "Invalid date '%s %d'", monthNames[month-1], day)
	}
	return LocalDate{year, month, day}, nil
}

// MonthStart returns the first day of a month; a month number outside 1 to 12 counts on from January of year (13
// is January of the next year).
func MonthStart(year, month int) LocalDate {
	return LocalDate{year, 1, 1}.PlusMonths(int64(month - 1))
}

// resolvePreviousValid returns the date, moving a day past the end of the month back to the month's last day.
func resolvePreviousValid(year, month, day int) LocalDate {
	if l := MonthLength(year, month); day > l {
		day = l
	}
	return LocalDate{year, month, day}
}

// OfYearDay returns the dayOfYear-th day of year.
func OfYearDay(year, dayOfYear int) (LocalDate, error) {
	if err := fieldYear.check(int64(year)); err != nil {
		return LocalDate{}, err
	}
	if err := fieldDayOfYear.check(int64(dayOfYear)); err != nil {
		return LocalDate{}, err
	}
	leap := IsLeapYear(year)
	if dayOfYear == 366 && !leap {
		return LocalDate{}, cronerr.New(cronerr.DateTime, "Invalid date 'DayOfYear 366' as '%d' is not a leap year", year)
	}
	month := 1
	for dayOfYear > MonthLength(year, month) {
		dayOfYear -= MonthLength(year, month)
		month++
	}
	return LocalDate{year, month, dayOfYear}, nil
}

// DateOfEpochDay returns the date that is epochDay days after 1970-01-01.
func DateOfEpochDay(epochDay int64) LocalDate {
	zeroDay := epochDay + days0000To1970
	zeroDay -= 60
	var adjust int64
	if zeroDay < 0 {
		adjustCycles := (zeroDay+1)/daysPerCycle - 1
		adjust = adjustCycles * 400
		zeroDay += -adjustCycles * daysPerCycle
	}
	yearEst := (400*zeroDay + 591) / daysPerCycle
	doyEst := zeroDay - (365*yearEst + yearEst/4 - yearEst/100 + yearEst/400)
	if doyEst < 0 {
		yearEst--
		doyEst = zeroDay - (365*yearEst + yearEst/4 - yearEst/100 + yearEst/400)
	}
	yearEst += adjust
	marchDoy0 := doyEst
	marchMonth0 := (marchDoy0*5 + 2) / 153
	month := (marchMonth0+2)%12 + 1
	dom := marchDoy0 - (marchMonth0*306+5)/10 + 1
	yearEst += marchMonth0 / 10
	return LocalDate{int(yearEst), int(month), int(dom)}
}

// Year returns the year.
func (d LocalDate) Year() int { return d.year }

// Month returns the month, 1 to 12.
func (d LocalDate) Month() int { return d.month }

// Day returns the day of the month.
func (d LocalDate) Day() int { return d.day }

// IsLeapYear reports whether the date's year is a leap year.
func (d LocalDate) IsLeapYear() bool { return IsLeapYear(d.year) }

// LengthOfMonth returns the number of days of the date's month.
func (d LocalDate) LengthOfMonth() int { return MonthLength(d.year, d.month) }

// LengthOfYear returns the number of days of the date's year.
func (d LocalDate) LengthOfYear() int {
	if d.IsLeapYear() {
		return 366
	}
	return 365
}

// EpochDay returns the number of days from 1970-01-01 to the date.
func (d LocalDate) EpochDay() int64 {
	y := int64(d.year)
	m := int64(d.month)
	total := 365 * y
	if y >= 0 {
		total += (y+3)/4 - (y+99)/100 + (y+399)/400
	} else {
		total -= y/-4 - y/-100 + y/-400
	}
	total += (367*m - 362) / 12
	total += int64(d.day) - 1
	if m > 2 {
		total--
		if !d.IsLeapYear() {
			total--
		}
	}
	return total - days0000To1970
}

// DayOfWeek returns the ISO day of the week, 1 for Monday to 7 for Sunday.
func (d LocalDate) DayOfWeek() int {
	return int(floorMod(d.EpochDay()+3, 7)) + 1
}

// DayOfYear returns the day of the year, 1 to 366.
func (d LocalDate) DayOfYear() int {
	doy := d.day
	for m := 1; m < d.month; m++ {
		doy += MonthLength(d.year, m)
	}
	return doy
}

// PlusDays returns the date n days later.
func (d LocalDate) PlusDays(n int64) LocalDate {
	if n == 0 {
		return d
	}
	return DateOfEpochDay(d.EpochDay() + n)
}

// PlusMonths returns the date n months later, moving a day past the end of the month back to its last day.
func (d LocalDate) PlusMonths(n int64) LocalDate {
	if n == 0 {
		return d
	}
	count := int64(d.year)*12 + int64(d.month-1) + n
	return resolvePreviousValid(int(floorDiv(count, 12)), int(floorMod(count, 12))+1, d.day)
}

// PlusYears returns the date n years later; February 29 becomes February 28 in a common year.
func (d LocalDate) PlusYears(n int64) LocalDate {
	if n == 0 {
		return d
	}
	return resolvePreviousValid(d.year+int(n), d.month, d.day)
}

// WithMonth returns the date in another month of the same year, clamping the day to the month's length.
func (d LocalDate) WithMonth(month int) (LocalDate, error) {
	if err := fieldMonth.check(int64(month)); err != nil {
		return LocalDate{}, err
	}
	return resolvePreviousValid(d.year, month, d.day), nil
}

// WithDayOfMonth returns the date on another day of the same month, or an error when that day does not exist.
func (d LocalDate) WithDayOfMonth(day int) (LocalDate, error) {
	if d.day == day {
		return d, nil
	}
	return DateOf(d.year, d.month, day)
}

// WithLastDayOfMonth returns the last day of the date's month.
func (d LocalDate) WithLastDayOfMonth() LocalDate {
	return LocalDate{d.year, d.month, d.LengthOfMonth()}
}

// Compare returns -1, 0 or 1 as d is before, equal to or after o.
func (d LocalDate) Compare(o LocalDate) int {
	switch {
	case d.year != o.year:
		return sign(d.year - o.year)
	case d.month != o.month:
		return sign(d.month - o.month)
	default:
		return sign(d.day - o.day)
	}
}

// String returns the date as yyyy-MM-dd; years beyond 9999 carry a plus sign and years before 0 a minus sign.
func (d LocalDate) String() string {
	var year string
	abs := d.year
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs < 1000 && d.year < 0:
		year = fmt.Sprintf("-%04d", abs)
	case abs < 1000:
		year = fmt.Sprintf("%04d", abs)
	case d.year > 9999:
		year = "+" + strconv.Itoa(d.year)
	default:
		year = strconv.Itoa(d.year)
	}
	return fmt.Sprintf("%s-%02d-%02d", year, d.month, d.day)
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 {
	return a - floorDiv(a, b)*b
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
