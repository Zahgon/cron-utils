// Copyright 2015 jmrozanec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Ported to Go from com/cronutils/model/time/SingleExecutionTime.java.

package executiontime

import (
	"errors"
	"slices"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

const maxIterations = 100_000

// singleExecutionTime computes the executions of a single cron by moving a candidate date-time field by field, from
// the year down to the second, until every field matches.
type singleExecutionTime struct {
	definition  *model.CronDefinition
	years       generator
	daysOfWeek  *model.CronField
	daysOfMonth *model.CronField
	daysOfYear  *model.CronField
	months      *timeNode
	hours       *timeNode
	minutes     *timeNode
	seconds     *timeNode
}

func newSingleExecutionTime(definition *model.CronDefinition, year, daysOfWeek, daysOfMonth, daysOfYear *model.CronField,
	months, hours, minutes, seconds *timeNode) (*singleExecutionTime, error) {
	s := &singleExecutionTime{definition: definition, daysOfWeek: daysOfWeek, daysOfMonth: daysOfMonth,
		daysOfYear: daysOfYear, months: months, hours: hours, minutes: minutes, seconds: seconds}
	var err error
	if yd := definition.FieldDefinition(model.Year); yd != nil {
		if !yd.Optional && year == nil {
			return nil, cronerr.New(cronerr.NullValue, "")
		}
		if year == nil {
			year = model.NewCronField(model.Year, model.AlwaysExpression(), yd.Constraints)
		}
		s.years, err = forCronField(year)
	} else {
		s.years, err = forCronField(model.NewCronField(model.Year, model.AlwaysExpression(),
			model.NewFieldConstraintsBuilder().Build()))
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// result is a candidate date-time and whether it matches the cron.
type result struct {
	time  javatime.ZonedDateTime
	match bool
}

func noMatch(t javatime.ZonedDateTime) (result, error) { return result{time: t}, nil }

// NextExecution returns the first execution after date.
func (s *singleExecutionTime) NextExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	next, err := s.nextExecution(date)
	if errors.Is(err, errNoSuchValue) {
		return javatime.ZonedDateTime{}, false, nil
	}
	if err != nil {
		return javatime.ZonedDateTime{}, false, err
	}
	return next, true, nil
}

func (s *singleExecutionTime) nextExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, error) {
	nextMatch, err := s.nextClosestMatch(date)
	if err != nil {
		return nextMatch, err
	}
	if nextMatch.Equal(date) {
		nextMatch, err = s.nextClosestMatch(date.PlusSeconds(1))
		if err != nil {
			return nextMatch, err
		}
		// After the clocks go back, an hourly or less frequent cron would run twice in the repeated hour: skip it.
		if javatime.CompareOffsets(nextMatch.Offset(), date.Offset()) > 0 {
			following, ok, err := s.NextExecution(nextMatch)
			if err != nil {
				return nextMatch, err
			}
			if ok && javatime.Between(nextMatch, following).ToHours() > 1 {
				return s.nextClosestMatch(date.PlusSeconds(1).PlusHours(1))
			}
		}
		return nextMatch, nil
	}
	if date.Minute() != nextMatch.Minute() {
		return s.nextClosestMatch(date.TruncatedTo(javatime.Minutes).PlusMinutes(1))
	}
	return nextMatch, nil
}

func (s *singleExecutionTime) nextClosestMatch(date javatime.ZonedDateTime) (javatime.ZonedDateTime, error) {
	r := result{time: date}
	for i := 0; i < maxIterations; i++ {
		var err error
		if r, err = s.potentialNextClosestMatch(r.time); err != nil {
			return javatime.ZonedDateTime{}, err
		}
		if r.match {
			return r.time, nil
		}
		if r.time.Year()-date.Year() > 100 {
			return javatime.ZonedDateTime{}, errNoSuchValue
		}
	}
	return javatime.ZonedDateTime{}, errNoSuchValue
}

func (s *singleExecutionTime) potentialNextClosestMatch(date javatime.ZonedDateTime) (result, error) {
	years, err := candidates(s.years, date.Year(), date.Year())
	if err != nil {
		return result{}, err
	}
	years = slices.DeleteFunc(years, func(y int) bool { return y < date.Year() })
	lowestMonth, lowestHour, lowestMinute, lowestSecond := s.months.first(), s.hours.first(), s.minutes.first(),
		s.seconds.first()
	if len(years) == 0 {
		return s.nextPotentialYear(date, lowestMonth, lowestHour, lowestMinute, lowestSecond)
	}
	if !s.months.contains(date.Month()) {
		return s.nextPotentialMonth(date, lowestHour, lowestMinute, lowestSecond)
	}
	days, err := s.generateDays(date)
	if err != nil {
		return result{}, err
	}
	if days == nil {
		return noMatch(toBeginOfNextMonth(date))
	}
	if !days.contains(date.Day()) {
		return s.nextPotentialDayOfMonth(date, lowestHour, lowestMinute, lowestSecond, days)
	}
	if !s.hours.contains(date.Hour()) {
		return nextPotentialValue(date, s.hours, javatime.Hours)
	}
	if !s.minutes.contains(date.Minute()) {
		return nextPotentialValue(date, s.minutes, javatime.Minutes)
	}
	// A date-time with nanoseconds is already past its second: round up to the next whole second.
	rounded := date
	if date.Nano() != 0 {
		rounded = date.PlusNanos(int64(1_000_000_000 - date.Nano()))
	}
	if !s.seconds.contains(rounded.Second()) {
		return nextPotentialValue(rounded, s.seconds, javatime.Seconds)
	}
	return result{rounded, true}, nil
}

func (s *singleExecutionTime) nextPotentialYear(date javatime.ZonedDateTime, lowestMonth, lowestHour, lowestMinute,
	lowestSecond int) (result, error) {
	newYear, err := s.years.next(date.Year())
	if err != nil {
		return result{}, err
	}
	first, err := javatime.DateOf(newYear, lowestMonth, 1)
	if err != nil {
		return result{}, err
	}
	days, err := s.generateDays(javatime.OfDateTime(first, javatime.Midnight, date.Zone()))
	if err != nil {
		return result{}, err
	}
	if days == nil {
		return noMatch(toBeginOfNextMonth(date))
	}
	day, err := javatime.DateOf(newYear, lowestMonth, days.first())
	if err != nil {
		return result{}, err
	}
	t, err := javatime.TimeOf(lowestHour, lowestMinute, lowestSecond)
	if err != nil {
		return result{}, err
	}
	return noMatch(javatime.OfDateTime(day, t, date.Zone()))
}

func (s *singleExecutionTime) nextPotentialMonth(date javatime.ZonedDateTime, lowestHour, lowestMinute,
	lowestSecond int) (result, error) {
	nearest := s.months.nextValue(date.Month(), 0)
	if nearest.shifts > 0 {
		d, err := date.TruncatedTo(javatime.Days).WithMonth(1)
		if err != nil {
			return result{}, err
		}
		if d, err = d.WithDayOfMonth(1); err != nil {
			return result{}, err
		}
		return noMatch(d.PlusYears(int64(nearest.shifts)))
	}
	first, err := javatime.OfFields(date.Year(), nearest.value, 1, 0, 0, 0, 0, date.Zone())
	if err != nil {
		return result{}, err
	}
	days, err := s.generateDays(first)
	if err != nil {
		return result{}, err
	}
	if days == nil {
		return noMatch(toBeginOfNextMonth(date))
	}
	d, err := date.TruncatedTo(javatime.Seconds).WithMonth(nearest.value)
	if err != nil {
		return result{}, err
	}
	if d, err = d.WithDayOfMonth(days.first()); err != nil {
		return result{}, err
	}
	t, err := javatime.TimeOf(lowestHour, lowestMinute, lowestSecond)
	if err != nil {
		return result{}, err
	}
	return noMatch(d.WithTime(t))
}

func (s *singleExecutionTime) nextPotentialDayOfMonth(date javatime.ZonedDateTime, lowestHour, lowestMinute,
	lowestSecond int, days *timeNode) (result, error) {
	nearest := days.nextValue(date.Day(), 0)
	if nearest.shifts > 0 {
		d, err := date.TruncatedTo(javatime.Days).WithDayOfMonth(1)
		if err != nil {
			return result{}, err
		}
		return noMatch(d.PlusMonths(int64(nearest.shifts)))
	}
	d, err := date.TruncatedTo(javatime.Seconds).WithDayOfMonth(nearest.value)
	if err != nil {
		return result{}, err
	}
	t, err := javatime.TimeOf(lowestHour, lowestMinute, lowestSecond)
	if err != nil {
		return result{}, err
	}
	return noMatch(d.WithTime(t))
}

// unitRange is the number of steps of a time field's range: 23 hours, 59 minutes, 59 seconds.
func unitRange(unit javatime.Unit) int {
	if unit == javatime.Hours {
		return 23
	}
	return 59
}

func nextPotentialValue(date javatime.ZonedDateTime, node *timeNode, unit javatime.Unit) (result, error) {
	d := date
	for i := 0; i < 2*unitRange(unit); i++ {
		d = d.Plus(1, unit)
		if node.contains(d.Get(unit)) {
			return noMatch(d.TruncatedTo(unit))
		}
	}
	return result{}, errNoSuchValue
}

func toBeginOfNextMonth(d javatime.ZonedDateTime) javatime.ZonedDateTime {
	return d.TruncatedTo(javatime.Days).PlusMonths(1).WithFirstDayOfMonth()
}

// LastExecution returns the last execution before date.
func (s *singleExecutionTime) LastExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	previous, err := s.previousClosestMatch(date)
	if err == nil && previous.Equal(date) {
		previous, err = s.previousClosestMatch(date.MinusSeconds(1))
	}
	if errors.Is(err, errNoSuchValue) {
		return javatime.ZonedDateTime{}, false, nil
	}
	if err != nil {
		return javatime.ZonedDateTime{}, false, err
	}
	return previous, true, nil
}

func (s *singleExecutionTime) previousClosestMatch(date javatime.ZonedDateTime) (javatime.ZonedDateTime, error) {
	r := result{time: date}
	for i := 0; i < maxIterations; i++ {
		var err error
		if r, err = s.potentialPreviousClosestMatch(r.time); err != nil {
			return javatime.ZonedDateTime{}, err
		}
		if r.match {
			return r.time, nil
		}
	}
	return javatime.ZonedDateTime{}, errNoSuchValue
}

func (s *singleExecutionTime) potentialPreviousClosestMatch(date javatime.ZonedDateTime) (result, error) {
	var years []int
	if yd := s.definition.FieldDefinition(model.Year); yd != nil {
		all, err := candidates(s.years, yd.Constraints.StartRange(), date.Year())
		if err != nil {
			return result{}, err
		}
		for _, y := range all {
			if y <= date.Year() {
				years = append(years, y)
			}
		}
	} else {
		years = []int{date.Year()}
	}
	days, err := s.generateDays(date)
	if err != nil {
		return result{}, err
	}
	highestMonth, highestHour, highestMinute, highestSecond := s.months.last(), s.hours.last(), s.minutes.last(),
		s.seconds.last()
	highestDay := 1
	if days != nil {
		highestDay = days.last()
	}
	if !slices.Contains(years, date.Year()) {
		validPrevYear, found := 0, false
		for _, y := range years {
			if y < date.Year() && (!found || y > validPrevYear) {
				validPrevYear, found = y, true
			}
		}
		if found {
			lastDateOfYear, err := javatime.OfFields(validPrevYear, highestMonth, 1, highestHour, highestMinute,
				highestSecond, 0, date.Zone())
			if err != nil {
				return result{}, err
			}
			yearEndDays, err := s.generateDays(lastDateOfYear)
			if err != nil {
				return result{}, err
			}
			if yearEndDays != nil {
				day := min(yearEndDays.last(), lastDateOfYear.Date().LengthOfMonth())
				r, err := lastDateOfYear.WithDayOfMonth(day)
				if err != nil {
					return result{}, err
				}
				if !s.definition.ContainsFieldDefinition(model.Second) {
					r = r.TruncatedTo(javatime.Minutes)
				}
				return noMatch(r)
			}
		}
		return s.previousPotentialYear(date, days, highestMonth, highestDay, highestHour, highestMinute, highestSecond)
	}
	if !s.months.contains(date.Month()) {
		return s.previousPotentialMonth(date)
	}
	if days == nil || !slices.ContainsFunc(days.values, func(d int) bool { return d <= date.Day() }) {
		return s.toEndOfPreviousMonth(date)
	}
	if !days.contains(date.Day()) {
		return s.previousPotentialDayOfMonth(date, days, highestHour, highestMinute, highestSecond)
	}
	if !s.hours.contains(date.Hour()) {
		return previousPotentialValue(date, s.hours, javatime.Hours)
	}
	if !s.minutes.contains(date.Minute()) {
		return previousPotentialValue(date, s.minutes, javatime.Minutes)
	}
	if !s.seconds.contains(date.Second()) {
		return previousPotentialValue(date, s.seconds, javatime.Seconds)
	}
	if !s.definition.ContainsFieldDefinition(model.Second) {
		return result{date.TruncatedTo(javatime.Minutes), true}, nil
	}
	return result{date.TruncatedTo(javatime.Seconds), true}, nil
}

func (s *singleExecutionTime) previousPotentialYear(date javatime.ZonedDateTime, days *timeNode, highestMonth,
	highestDay, highestHour, highestMinute, highestSecond int) (result, error) {
	previousYear, err := s.years.previous(date.Year())
	if err != nil {
		return result{}, err
	}
	if highestDay > 28 && highestDay > javatime.MonthLength(previousYear, highestMonth) {
		if days == nil {
			return result{}, cronerr.New(cronerr.NullValue, "")
		}
		nearest := days.previousValue(highestDay, 1)
		if nearest.shifts > 0 {
			first, err := javatime.DateOf(previousYear, highestMonth, 1)
			if err != nil {
				return result{}, err
			}
			d := javatime.OfDateTime(first, javatime.MaxSeconds, date.Zone()).MinusMonths(int64(nearest.shifts))
			return noMatch(d.WithLastDayOfMonth())
		}
		highestDay = nearest.value
	}
	day, err := javatime.DateOf(previousYear, highestMonth, highestDay)
	if err != nil {
		return result{}, err
	}
	t, err := javatime.TimeOf(highestHour, highestMinute, highestSecond)
	if err != nil {
		return result{}, err
	}
	return noMatch(javatime.OfDateTime(day, t, date.Zone()))
}

func (s *singleExecutionTime) previousPotentialMonth(date javatime.ZonedDateTime) (result, error) {
	nearest := s.months.previousValue(date.Month(), 0)
	if nearest.shifts > 0 {
		end := javatime.OfDateTime(javatime.MonthStart(date.Year(), 12).WithLastDayOfMonth(), javatime.MaxSeconds, date.Zone())
		return noMatch(end.MinusYears(int64(nearest.shifts)))
	}
	first, err := javatime.OfFields(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Zone())
	if err != nil {
		return result{}, err
	}
	return noMatch(first.MinusNanos(1))
}

func (s *singleExecutionTime) previousPotentialDayOfMonth(date javatime.ZonedDateTime, days *timeNode, highestHour,
	highestMinute, highestSecond int) (result, error) {
	nearest := days.previousValue(date.Day(), 0)
	if nearest.shifts > 0 {
		first := javatime.MonthStart(date.Year(), date.Month())
		d := javatime.OfDateTime(first, javatime.MaxSeconds, date.Zone()).MinusMonths(int64(nearest.shifts))
		return noMatch(d.WithLastDayOfMonth())
	}
	d, err := date.WithDayOfMonth(nearest.value)
	if err != nil {
		return result{}, err
	}
	t, err := javatime.TimeOf(highestHour, highestMinute, highestSecond)
	if err != nil {
		return result{}, err
	}
	return noMatch(d.WithTime(t).TruncatedTo(javatime.Seconds))
}

func previousPotentialValue(date javatime.ZonedDateTime, node *timeNode, unit javatime.Unit) (result, error) {
	d := date
	for i := 0; i < 2*unitRange(unit); i++ {
		d = d.Plus(-1, unit)
		if node.contains(d.Get(unit)) {
			return noMatch(d.TruncatedTo(unit).Plus(1, unit).MinusSeconds(1))
		}
	}
	return result{}, errNoSuchValue
}

func (s *singleExecutionTime) toEndOfPreviousMonth(date javatime.ZonedDateTime) (result, error) {
	previous := date.MinusMonths(1).WithLastDayOfMonth()
	d, err := javatime.OfFields(previous.Year(), previous.Month(), previous.Day(), s.hours.last(), s.minutes.last(),
		s.seconds.last(), 0, previous.Zone())
	if err != nil {
		return result{}, err
	}
	return noMatch(d)
}

// TimeToNextExecution returns the time from date to the next execution.
func (s *singleExecutionTime) TimeToNextExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	next, ok, err := s.NextExecution(date)
	if err != nil || !ok {
		return javatime.Duration{}, ok, err
	}
	return javatime.Between(date, next), true, nil
}

// TimeFromLastExecution returns the time from the last execution to date.
func (s *singleExecutionTime) TimeFromLastExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	last, ok, err := s.LastExecution(date)
	if err != nil || !ok {
		return javatime.Duration{}, ok, err
	}
	return javatime.Between(last, date), true, nil
}

// IsMatch reports whether date is an execution. Dialects without seconds only match whole minutes; dialects with
// seconds but without a year field only match whole seconds.
func (s *singleExecutionTime) IsMatch(date javatime.ZonedDateTime) (bool, error) {
	secondGranularity := s.definition.ContainsFieldDefinition(model.Second)
	quartzLike := secondGranularity && s.definition.ContainsFieldDefinition(model.Year) &&
		s.definition.ContainsFieldDefinition(model.DayOfWeek)
	if secondGranularity {
		if !quartzLike && date.Nano() > 0 {
			return false, nil
		}
		date = date.TruncatedTo(javatime.Seconds)
	} else {
		if date.Second() != 0 {
			return false, nil
		}
		truncated := date.TruncatedTo(javatime.Minutes)
		if !s.inExpectedRanges(truncated, truncated) {
			return false, nil
		}
		if date.Nano() != 0 {
			return false, nil
		}
		date = truncated
	}
	last, ok, err := s.LastExecution(date)
	if err != nil {
		return false, err
	}
	if ok {
		next, ok, err := s.NextExecution(last)
		if err != nil {
			return false, err
		}
		if ok {
			return next.Equal(date), nil
		}
		inRange := false
		if m, err := s.nextClosestMatch(date); err == nil {
			inRange = s.inExpectedRanges(m, date)
		} else if !errors.Is(err, errNoSuchValue) {
			return false, err
		}
		if m, err := s.previousClosestMatch(date); err == nil {
			inRange = s.inExpectedRanges(m, date)
		} else if !errors.Is(err, errNoSuchValue) {
			return false, err
		}
		return inRange, nil
	}
	m, err := s.nextClosestMatch(date.MinusSeconds(1))
	if errors.Is(err, errNoSuchValue) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.inExpectedRanges(m, date), nil
}

func (s *singleExecutionTime) inExpectedRanges(valid, date javatime.ZonedDateTime) bool {
	d := s.definition
	in := true
	if d.ContainsFieldDefinition(model.Year) {
		in = valid.Year() == date.Year()
	}
	if d.ContainsFieldDefinition(model.Month) {
		in = in && valid.Month() == date.Month()
	}
	if d.ContainsFieldDefinition(model.DayOfMonth) {
		in = in && valid.Day() == date.Day()
	}
	if d.ContainsFieldDefinition(model.DayOfWeek) {
		in = in && valid.DayOfWeek() == date.DayOfWeek()
	}
	if d.ContainsFieldDefinition(model.Hour) {
		in = in && valid.Hour() == date.Hour()
	}
	if d.ContainsFieldDefinition(model.Minute) {
		in = in && valid.Minute() == date.Minute()
	}
	if d.ContainsFieldDefinition(model.Second) {
		in = in && valid.Second() == date.Second()
	} else {
		in = in && date.Second() == 0
	}
	return in
}

// generateDays returns the days of date's month the cron runs on, or nil when there are none.
func (s *singleExecutionTime) generateDays(date javatime.ZonedDateTime) (*timeNode, error) {
	d := s.definition
	if s.daysAsDayOfYear() {
		return s.dayCandidatesUsingDoY(date)
	}
	dow := d.FieldDefinition(model.DayOfWeek)
	if dow != nil && d.FieldDefinition(model.DayOfMonth) != nil {
		var values []int
		var err error
		if dow.Constraints.SupportsSpecialChar(model.QuestionMarkChar) {
			values, err = s.dayCandidatesQuestionMarkSupported(date.Year(), date.Month(), *dow.MondayDoW)
		} else {
			values, err = s.dayCandidatesQuestionMarkNotSupported(date.Year(), date.Month(), *dow.MondayDoW)
		}
		if err != nil || len(values) == 0 {
			return nil, err
		}
		return newTimeNode(values)
	}
	if dow == nil {
		g, err := dayOfMonthGenerator(s.daysOfMonth, date.Year(), date.Month())
		if err != nil {
			return nil, err
		}
		values, err := candidates(g, 1, javatime.MonthLength(date.Year(), date.Month()))
		if err != nil || len(values) == 0 {
			return nil, err
		}
		return newTimeNode(values)
	}
	g, err := dayOfWeekGenerator(s.daysOfWeek, date.Year(), date.Month(), *dow.MondayDoW)
	if err != nil {
		return nil, err
	}
	values, err := candidates(g, 1, javatime.MonthLength(date.Year(), date.Month()))
	if err != nil {
		return nil, err
	}
	return newTimeNode(values)
}

func (s *singleExecutionTime) daysAsDayOfYear() bool {
	doy := s.definition.FieldDefinition(model.DayOfYear)
	if doy == nil {
		return false
	}
	if !doy.Constraints.SupportsSpecialChar(model.QuestionMarkChar) {
		return true
	}
	_, isQuestionMark := s.daysOfYear.Expression.(*model.QuestionMark)
	return !isQuestionMark
}

func (s *singleExecutionTime) dayCandidatesUsingDoY(reference javatime.ZonedDateTime) (*timeNode, error) {
	year, month := reference.Year(), reference.Month()
	lengthOfYear := 365
	if javatime.IsLeapYear(year) {
		lengthOfYear = 366
	}
	g, err := forCronField(s.daysOfYear)
	if err != nil {
		return nil, err
	}
	all, err := candidates(g, 1, lengthOfYear)
	if err != nil {
		return nil, err
	}
	low := javatime.MonthStart(year, month).DayOfYear()
	high := lengthOfYear + 1
	if month != 12 {
		high = javatime.MonthStart(year, month+1).DayOfYear()
	}
	var days []int
	for _, doy := range all {
		if doy >= low && doy < high {
			date, err := javatime.OfYearDay(year, doy)
			if err != nil {
				return nil, err
			}
			days = append(days, date.Day())
		}
	}
	if len(days) == 0 {
		return nil, nil
	}
	return newTimeNode(days)
}

func distinctSorted(values []int) []int {
	slices.Sort(values)
	return slices.Compact(values)
}

func (s *singleExecutionTime) domCandidates(year, month int) ([]int, error) {
	g, err := dayOfMonthGenerator(s.daysOfMonth, year, month)
	if err != nil {
		return nil, err
	}
	return candidates(g, 1, javatime.MonthLength(year, month))
}

func (s *singleExecutionTime) dowCandidates(year, month int, mondayDoW model.WeekDay) ([]int, error) {
	g, err := dayOfWeekGenerator(s.daysOfWeek, year, month, mondayDoW)
	if err != nil {
		return nil, err
	}
	return candidates(g, 1, javatime.MonthLength(year, month))
}

func (s *singleExecutionTime) dayCandidatesQuestionMarkNotSupported(year, month int, mondayDoW model.WeekDay) ([]int, error) {
	_, domAlways := s.daysOfMonth.Expression.(*model.Always)
	_, dowAlways := s.daysOfWeek.Expression.(*model.Always)
	switch {
	case domAlways && dowAlways:
		return s.domCandidates(year, month)
	case domAlways:
		return s.dowCandidates(year, month, mondayDoW)
	case dowAlways:
		return s.domCandidates(year, month)
	}
	dow, err := s.dowCandidates(year, month, mondayDoW)
	if err != nil {
		return nil, err
	}
	dom, err := s.domCandidates(year, month)
	if err != nil {
		return nil, err
	}
	if s.definition.IsMatchDayOfWeekAndDayOfMonth() {
		var both []int
		for _, d := range dom {
			if slices.Contains(dow, d) {
				both = append(both, d)
			}
		}
		return distinctSorted(both), nil
	}
	return distinctSorted(append(dow, dom...)), nil
}

func (s *singleExecutionTime) dayCandidatesQuestionMarkSupported(year, month int, mondayDoW model.WeekDay) ([]int, error) {
	_, domAlways := s.daysOfMonth.Expression.(*model.Always)
	_, dowAlways := s.daysOfWeek.Expression.(*model.Always)
	_, domQuestion := s.daysOfMonth.Expression.(*model.QuestionMark)
	_, dowQuestion := s.daysOfWeek.Expression.(*model.QuestionMark)
	switch {
	case domAlways && dowAlways:
		return s.domCandidates(year, month)
	case domQuestion:
		return s.dowCandidates(year, month, mondayDoW)
	case dowQuestion:
		return s.domCandidates(year, month)
	}
	dom, err := s.domCandidates(year, month)
	if err != nil {
		return nil, err
	}
	dow, err := s.dowCandidates(year, month, mondayDoW)
	if err != nil {
		return nil, err
	}
	var both []int
	for _, d := range dom {
		if slices.Contains(dow, d) {
			both = append(both, d)
		}
	}
	return distinctSorted(both), nil
}
