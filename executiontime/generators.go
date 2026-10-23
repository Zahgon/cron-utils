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
// Ported to Go from com/cronutils/model/time/generator/*.java.

package executiontime

import (
	"errors"
	"slices"
	"strconv"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

// errNoSuchValue reports that a generator has no further value in the asked direction.
var errNoSuchValue = errors.New("no such value")

// generator produces the values a field expression matches.
type generator interface {
	// next returns the first matching value after reference.
	next(reference int) (int, error)
	// previous returns the last matching value before reference.
	previous(reference int) (int, error)
	// between returns matching values strictly between start and end.
	between(start, end int) ([]int, error)
	// isMatch reports whether value matches.
	isMatch(value int) (bool, error)
}

// candidates returns the sorted distinct matching values from start to end, both included.
func candidates(g generator, start, end int) ([]int, error) {
	values, err := g.between(start, end)
	if err != nil {
		return nil, err
	}
	for _, v := range []int{start, end} {
		ok, err := g.isMatch(v)
		if err != nil {
			return nil, err
		}
		if ok {
			values = append(values, v)
		}
	}
	slices.Sort(values)
	return slices.Compact(values), nil
}

// forCronField returns the generator of a field expression.
func forCronField(f *model.CronField) (generator, error) {
	switch e := f.Expression.(type) {
	case *model.Always:
		return &alwaysGenerator{f}, nil
	case *model.And:
		return &andGenerator{f}, nil
	case *model.Between:
		return &betweenGenerator{f}, nil
	case *model.Every:
		return newEveryGenerator(f)
	case *model.On:
		if e.SpecialChar.Value() != model.None {
			return nil, cronerr.Argument("Cannot create instance for On instance with %s value", e.SpecialChar)
		}
		return &onGenerator{f}, nil
	}
	return nullGenerator{}, nil
}

func dayOfMonthGenerator(f *model.CronField, year, month int) (generator, error) {
	if on, ok := f.Expression.(*model.On); ok && on.SpecialChar.Value() != model.None {
		return &onDayOfMonthGenerator{f, year, month}, nil
	}
	return forCronField(f)
}

func dayOfWeekGenerator(f *model.CronField, year, month int, mondayDoW model.WeekDay) (generator, error) {
	switch f.Expression.(type) {
	case *model.On:
		return &onDayOfWeekGenerator{f, year, month, mondayDoW}, nil
	case *model.Between:
		return newBetweenDayOfWeekGenerator(f, year, month, mondayDoW)
	case *model.And:
		return &andDayOfWeekGenerator{f, year, month, mondayDoW}, nil
	case *model.Every:
		return newEveryDayOfWeekGenerator(f, year, month, mondayDoW)
	}
	return forCronField(f)
}

// alwaysGenerator matches every value of the field's range.
type alwaysGenerator struct{ f *model.CronField }

func (g *alwaysGenerator) next(reference int) (int, error) {
	if v := reference + 1; v <= g.f.Constraints.EndRange() {
		return v, nil
	}
	return 0, errNoSuchValue
}

func (g *alwaysGenerator) previous(reference int) (int, error) {
	if v := reference - 1; v >= g.f.Constraints.StartRange() {
		return v, nil
	}
	return 0, errNoSuchValue
}

func (g *alwaysGenerator) between(start, end int) ([]int, error) {
	var values []int
	for j := start + 1; j < end; j++ {
		values = append(values, j)
	}
	return values, nil
}

func (g *alwaysGenerator) isMatch(value int) (bool, error) {
	return g.f.Field == model.DayOfWeek || g.f.Constraints.IsInRange(value), nil
}

// andGenerator matches the values any listed expression matches.
type andGenerator struct{ f *model.CronField }

func (g *andGenerator) members() []model.FieldExpression {
	return g.f.Expression.(*model.And).Expressions()
}

func (g *andGenerator) member(e model.FieldExpression) (generator, error) {
	f := model.NewCronField(g.f.Field, e, g.f.Constraints)
	switch e.(type) {
	case *model.Always:
		return &alwaysGenerator{f}, nil
	case *model.Between:
		return &betweenGenerator{f}, nil
	case *model.Every:
		return newEveryGenerator(f)
	case *model.On:
		return &onGenerator{f}, nil
	}
	return nil, cronerr.Argument("FieldExpression %s not supported!", model.TypeName(e))
}

func (g *andGenerator) computeCandidates(step func(generator) (int, error)) ([]int, error) {
	var out []int
	for _, e := range g.members() {
		m, err := g.member(e)
		if err != nil {
			return nil, err
		}
		v, err := step(m)
		if errors.Is(err, errNoSuchValue) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if v >= 0 {
			out = append(out, v)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (g *andGenerator) next(reference int) (int, error) {
	values, err := g.computeCandidates(func(m generator) (int, error) { return m.next(reference) })
	if err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, errNoSuchValue
	}
	return values[0], nil
}

func (g *andGenerator) previous(reference int) (int, error) {
	values, err := g.computeCandidates(func(m generator) (int, error) { return m.previous(reference) })
	if err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, errNoSuchValue
	}
	return values[len(values)-1], nil
}

func (g *andGenerator) between(start, end int) ([]int, error) {
	var values []int
	reference, err := g.next(start)
	for err == nil && reference < end {
		values = append(values, reference)
		reference, err = g.next(reference)
	}
	if err != nil && !errors.Is(err, errNoSuchValue) {
		return nil, err
	}
	return values, nil
}

func (g *andGenerator) isMatch(value int) (bool, error) {
	for _, e := range g.members() {
		m, err := g.member(e)
		if err != nil {
			return false, err
		}
		ok, err := m.isMatch(value)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// betweenGenerator matches the values of a range.
type betweenGenerator struct{ f *model.CronField }

func (g *betweenGenerator) bounds() (from, to int, err error) {
	b := g.f.Expression.(*model.Between)
	if from, err = mapValue(b.From); err != nil {
		return 0, 0, err
	}
	to, err = mapValue(b.To)
	return from, to, err
}

// mapValue returns a range bound's integer.
func mapValue(v model.FieldValue) (int, error) {
	if i, ok := v.(model.IntegerFieldValue); ok {
		return int(i), nil
	}
	return 0, cronerr.Argument("Non integer values at intervals are not fully supported yet.")
}

func (g *betweenGenerator) next(reference int) (int, error) {
	b := g.f.Expression.(*model.Between)
	from, err := mapValue(b.From)
	if err != nil {
		return 0, err
	}
	candidate := reference + 1
	if candidate < from {
		candidate = from
	}
	to, err := mapValue(b.To)
	if err != nil {
		return 0, err
	}
	if candidate > to {
		return 0, errNoSuchValue
	}
	return candidate, nil
}

func (g *betweenGenerator) previous(reference int) (int, error) {
	b := g.f.Expression.(*model.Between)
	to, err := mapValue(b.To)
	if err != nil {
		return 0, err
	}
	candidate := reference - 1
	if candidate > to {
		candidate = to
	}
	from, err := mapValue(b.From)
	if err != nil {
		return 0, err
	}
	if candidate < from {
		return 0, errNoSuchValue
	}
	return candidate, nil
}

func (g *betweenGenerator) between(start, end int) ([]int, error) {
	expressionStart, expressionEnd, err := g.bounds()
	if err != nil {
		return nil, err
	}
	var values []int
	if start > expressionEnd || expressionStart > end {
		return values, nil
	}
	rangeStart, rangeEnd := start, end
	if expressionEnd < end {
		rangeEnd = expressionEnd
	}
	if expressionStart > start {
		rangeStart = expressionStart
	}
	if rangeStart != start {
		values = append(values, rangeStart)
	}
	reference, err := g.next(rangeStart)
	for err == nil && reference < rangeEnd {
		values = append(values, reference)
		reference, err = g.next(reference)
	}
	if errors.Is(err, errNoSuchValue) {
		return values, nil
	}
	if err != nil {
		return nil, err
	}
	if rangeEnd != end {
		values = append(values, reference)
	}
	return values, nil
}

func (g *betweenGenerator) isMatch(value int) (bool, error) {
	from, to, err := g.bounds()
	if err != nil {
		return false, err
	}
	return value >= from && value <= to, nil
}

// everyGenerator matches the values of a stepped expression.
type everyGenerator struct {
	f        *model.CronField
	from, to int
}

func newEveryGenerator(f *model.CronField) (*everyGenerator, error) {
	e := f.Expression.(*model.Every)
	g := &everyGenerator{f: f}
	switch inner := e.Expression.(type) {
	case *model.Between:
		from, err := mapValue(inner.From)
		if err != nil {
			return nil, err
		}
		to, err := mapValue(inner.To)
		if err != nil {
			return nil, err
		}
		g.from = max(f.Constraints.StartRange(), from)
		g.to = min(f.Constraints.EndRange(), to)
	case *model.On:
		g.from = inner.Time.Value()
		g.to = f.Constraints.EndRange()
	default:
		g.from = f.Constraints.StartRange()
		g.to = f.Constraints.EndRange()
	}
	return g, nil
}

func (g *everyGenerator) period() int { return g.f.Expression.(*model.Every).Period.Value() }

func (g *everyGenerator) offset() int {
	if on, ok := g.f.Expression.(*model.Every).Expression.(*model.On); ok {
		return on.Time.Value()
	}
	return g.from
}

func (g *everyGenerator) next(reference int) (int, error) {
	if reference >= g.to {
		return 0, errNoSuchValue
	}
	var next int
	if offset := g.offset(); reference >= offset {
		period := g.period()
		if period == 0 {
			return 0, cronerr.New(cronerr.IllegalArgument, "/ by zero")
		}
		next = reference + (period - (reference-offset)%period)
	} else {
		next = offset
	}
	if next < g.from {
		return g.from, nil
	}
	if next > g.to {
		return 0, errNoSuchValue
	}
	return next, nil
}

func (g *everyGenerator) previous(reference int) (int, error) {
	if reference < g.from {
		return 0, errNoSuchValue
	}
	if reference > g.to {
		return g.to, nil
	}
	period := g.period()
	if period == 0 {
		return 0, cronerr.New(cronerr.IllegalArgument, "/ by zero")
	}
	if remainder := (reference - g.from) % period; remainder != 0 {
		return reference - remainder, nil
	}
	return reference - period, nil
}

func (g *everyGenerator) between(start, end int) ([]int, error) {
	var values []int
	offset := g.offset()
	if start < offset && offset < end {
		values = append(values, offset)
	}
	reference, err := g.next(start)
	for err == nil && reference < end {
		if reference != offset {
			values = append(values, reference)
		}
		reference, err = g.next(reference)
	}
	if err != nil && !errors.Is(err, errNoSuchValue) {
		return nil, err
	}
	return values, nil
}

func (g *everyGenerator) isMatch(value int) (bool, error) {
	start := g.offset()
	period := g.period()
	if value < start {
		return false, nil
	}
	if period == 0 {
		return false, cronerr.New(cronerr.IllegalArgument, "/ by zero")
	}
	return (value-start)%period == 0 && value >= g.from && value <= g.to, nil
}

// onGenerator matches a single value.
type onGenerator struct{ f *model.CronField }

func (g *onGenerator) time() int { return g.f.Expression.(*model.On).Time.Value() }

func (g *onGenerator) next(reference int) (int, error) {
	if t := g.time(); t > reference {
		return t, nil
	}
	return 0, errNoSuchValue
}

func (g *onGenerator) previous(reference int) (int, error) {
	if t := g.time(); t < reference {
		return t, nil
	}
	return 0, errNoSuchValue
}

func (g *onGenerator) between(start, end int) ([]int, error) {
	if t := g.time(); t > start && t < end {
		return []int{t}, nil
	}
	return nil, nil
}

func (g *onGenerator) isMatch(value int) (bool, error) { return g.time() == value, nil }

// nullGenerator matches nothing.
type nullGenerator struct{}

func (nullGenerator) next(int) (int, error)           { return 0, errNoSuchValue }
func (nullGenerator) previous(int) (int, error)       { return 0, errNoSuchValue }
func (nullGenerator) between(int, int) ([]int, error) { return nil, nil }
func (nullGenerator) isMatch(int) (bool, error)       { return false, nil }

// calendarBetween collects the values next produces from start up to end.
func calendarBetween(next func(int) (int, error), start, end int) ([]int, error) {
	var values []int
	reference, err := next(start)
	for err == nil && reference < end {
		values = append(values, reference)
		reference, err = next(reference)
	}
	if err != nil && !errors.Is(err, errNoSuchValue) {
		return nil, err
	}
	return values, nil
}

// onDayOfMonthGenerator matches L, L-n, nW and LW days of a month.
type onDayOfMonthGenerator struct {
	f           *model.CronField
	year, month int
}

func (g *onDayOfMonthGenerator) value() (int, error) {
	on := g.f.Expression.(*model.On)
	day := on.Time.Value()
	switch on.SpecialChar.Value() {
	case model.L:
		daysBefore := max(on.Nth.Value(), 0)
		return javatime.MonthLength(g.year, g.month) - daysBefore, nil
	case model.W:
		date, err := javatime.DateOf(g.year, g.month, day)
		if err != nil {
			return 0, err
		}
		switch date.DayOfWeek() {
		case 6:
			if day == 1 {
				return 3, nil
			}
			return day - 1, nil
		case 7:
			if day+1 <= date.LengthOfMonth() {
				return day + 1, nil
			}
		}
		return day, nil
	case model.LW:
		last := javatime.MonthStart(g.year, g.month).WithLastDayOfMonth()
		if diff := last.DayOfWeek() - 5; diff > 0 {
			return last.PlusDays(int64(-diff)).Day(), nil
		}
		return last.Day(), nil
	}
	return 0, errNoSuchValue
}

func (g *onDayOfMonthGenerator) next(reference int) (int, error) {
	v, err := g.value()
	if err != nil {
		return 0, err
	}
	if v <= reference {
		return 0, errNoSuchValue
	}
	return v, nil
}

func (g *onDayOfMonthGenerator) previous(reference int) (int, error) {
	v, err := g.value()
	if err != nil {
		return 0, err
	}
	if v >= reference {
		return 0, errNoSuchValue
	}
	return v, nil
}

func (g *onDayOfMonthGenerator) between(start, end int) ([]int, error) {
	return calendarBetween(g.next, start, end)
}

func (g *onDayOfMonthGenerator) isMatch(value int) (bool, error) {
	v, err := g.value()
	if errors.Is(err, errNoSuchValue) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == value, nil
}

// onDayOfWeekGenerator matches a weekday, the nth weekday (#) or the last weekday (L) of a month.
type onDayOfWeekGenerator struct {
	f           *model.CronField
	year, month int
	mondayDoW   model.WeekDay
}

func (g *onDayOfWeekGenerator) value(reference int) (int, error) {
	on := g.f.Expression.(*model.On)
	switch on.SpecialChar.Value() {
	case model.Hash:
		return g.hashValue(on), nil
	case model.L:
		if on.Time.Value() == -1 {
			return g.noneValue(7, reference), nil
		}
		return g.lastValue(on), nil
	case model.None:
		return g.noneValue(on.Time.Value(), reference), nil
	}
	return 0, errNoSuchValue
}

func (g *onDayOfWeekGenerator) firstOfMonthDoW() int {
	return javatime.MonthStart(g.year, g.month).DayOfWeek()
}

func (g *onDayOfWeekGenerator) baseDay(requiredDoW int) int {
	baseDay := 1
	diff := g.firstOfMonthDoW() - requiredDoW
	if diff < 0 {
		baseDay -= diff
	}
	if diff > 0 {
		baseDay += 7 - diff
	}
	return baseDay
}

func (g *onDayOfWeekGenerator) hashValue(on *model.On) int {
	required := model.WeekDayMapping(g.mondayDoW, model.Java8WeekDay, on.Time.Value())
	return (on.Nth.Value()-1)*7 + g.baseDay(required)
}

func (g *onDayOfWeekGenerator) lastValue(on *model.On) int {
	last := javatime.MonthStart(g.year, g.month).WithLastDayOfMonth()
	lastDoW := last.DayOfWeek()
	required := model.WeekDayMapping(g.mondayDoW, model.Java8WeekDay, on.Time.Value())
	diff := lastDoW - required
	switch {
	case diff == 0:
		return last.Day()
	case diff < 0:
		return last.PlusDays(-int64(lastDoW + 7 - required)).Day()
	}
	return last.PlusDays(-int64(diff)).Day()
}

func (g *onDayOfWeekGenerator) noneValue(time, reference int) int {
	baseDay := g.baseDay(model.WeekDayMapping(g.mondayDoW, model.Java8WeekDay, time))
	if reference < 1 {
		return baseDay
	}
	for baseDay <= reference {
		baseDay += 7
	}
	return baseDay
}

func (g *onDayOfWeekGenerator) next(reference int) (int, error) {
	v, err := g.value(reference)
	if err != nil {
		return 0, err
	}
	if v <= reference {
		return 0, errNoSuchValue
	}
	return v, nil
}

func (g *onDayOfWeekGenerator) previous(reference int) (int, error) {
	v, err := g.value(reference)
	if err != nil {
		return 0, err
	}
	if v >= reference {
		return 0, errNoSuchValue
	}
	return v, nil
}

func (g *onDayOfWeekGenerator) between(start, end int) ([]int, error) {
	return calendarBetween(g.next, start, end)
}

func (g *onDayOfWeekGenerator) isMatch(value int) (bool, error) {
	v, err := g.value(value - 1)
	if errors.Is(err, errNoSuchValue) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == value, nil
}

// betweenDayOfWeekGenerator matches the days of a month that fall in a range of weekdays; a range that runs past
// the end of the week wraps around (FRI-TUE).
type betweenDayOfWeekGenerator struct {
	f           *model.CronField
	year, month int
	mondayDoW   model.WeekDay
	valid       []int
}

func newBetweenDayOfWeekGenerator(f *model.CronField, year, month int, mondayDoW model.WeekDay) (*betweenDayOfWeekGenerator, error) {
	b := f.Expression.(*model.Between)
	from, ok := b.From.(model.IntegerFieldValue)
	if !ok {
		return nil, cronerr.New(cronerr.TypeMismatch, "")
	}
	to, ok := b.To.(model.IntegerFieldValue)
	if !ok {
		return nil, cronerr.New(cronerr.TypeMismatch, "")
	}
	g := &betweenDayOfWeekGenerator{f: f, year: year, month: month, mondayDoW: mondayDoW}
	add := func(lo, hi int) {
		for v := lo; v <= hi; v++ {
			if !slices.Contains(g.valid, v) {
				g.valid = append(g.valid, v)
			}
		}
	}
	if to >= from {
		add(int(from), int(to))
	} else {
		add(int(from), f.Constraints.EndRange())
		add(f.Constraints.StartRange(), int(to))
	}
	slices.Sort(g.valid)
	return g, nil
}

func (g *betweenDayOfWeekGenerator) next(int) (int, error)     { return 0, nil }
func (g *betweenDayOfWeekGenerator) previous(int) (int, error) { return 0, nil }

func (g *betweenDayOfWeekGenerator) between(start, end int) ([]int, error) {
	constraints := model.NewFieldConstraintsBuilder().ForField(model.DayOfWeek).Build()
	field := parser.NewCronParserField(model.DayOfWeek, constraints, false)
	var values []int
	for _, day := range g.valid {
		cf, err := field.Parse(strconv.Itoa(day))
		if err != nil {
			return nil, err
		}
		days, err := candidates(&onDayOfWeekGenerator{cf, g.year, g.month, g.mondayDoW}, start, end)
		if err != nil {
			return nil, err
		}
		values = append(values, days...)
	}
	slices.Sort(values)
	return values, nil
}

func (g *betweenDayOfWeekGenerator) isMatch(value int) (bool, error) {
	date, err := javatime.DateOf(g.year, g.month, value)
	if err != nil {
		return false, err
	}
	cronDoW := date.DayOfWeek()%7 + g.mondayDoW.MondayDoWValue() - 1
	return slices.Contains(g.valid, cronDoW), nil
}

// andDayOfWeekGenerator matches the days any listed day-of-week expression matches.
type andDayOfWeekGenerator struct {
	f           *model.CronField
	year, month int
	mondayDoW   model.WeekDay
}

func (g *andDayOfWeekGenerator) next(int) (int, error)     { return 0, nil }
func (g *andDayOfWeekGenerator) previous(int) (int, error) { return 0, nil }
func (g *andDayOfWeekGenerator) isMatch(int) (bool, error) { return false, nil }

func (g *andDayOfWeekGenerator) between(start, end int) ([]int, error) {
	var values []int
	for _, e := range g.f.Expression.(*model.And).Expressions() {
		member, err := dayOfWeekGenerator(model.NewCronField(model.DayOfWeek, e, g.f.Constraints), g.year, g.month, g.mondayDoW)
		if err != nil {
			return nil, err
		}
		days, err := candidates(member, start, end)
		if err != nil {
			return nil, err
		}
		values = append(values, days...)
	}
	return values, nil
}

// everyDayOfWeekGenerator matches the days of a month whose weekday a stepped day-of-week expression selects.
type everyDayOfWeekGenerator struct {
	*everyGenerator
	year, month    int
	lastDayOfMonth int
	valid          map[int]bool
}

func newEveryDayOfWeekGenerator(f *model.CronField, year, month int, mondayDoW model.WeekDay) (*everyDayOfWeekGenerator, error) {
	base, err := newEveryGenerator(f)
	if err != nil {
		return nil, err
	}
	period := base.period()
	if period <= 0 || period >= 8 {
		return nil, cronerr.Argument("Cron Expression for day of week has an invalid period.")
	}
	g := &everyDayOfWeekGenerator{everyGenerator: base, year: year, month: month,
		lastDayOfMonth: javatime.MonthLength(year, month), valid: map[int]bool{}}
	for day := base.from; day <= base.to; day += period {
		javaDay := model.WeekDayMapping(mondayDoW, model.Java8WeekDay, day)
		if err := javatime.CheckDayOfWeek(javaDay); err != nil {
			return nil, err
		}
		g.valid[javaDay] = true
	}
	if len(g.valid) == 0 {
		return nil, cronerr.Argument("Collection is empty")
	}
	return g, nil
}

func (g *everyDayOfWeekGenerator) next(reference int) (int, error) {
	day := reference
	for {
		day++
		if g.matches(day) || day > g.lastDayOfMonth {
			break
		}
	}
	if day > g.lastDayOfMonth {
		return 0, errNoSuchValue
	}
	return day, nil
}

func (g *everyDayOfWeekGenerator) previous(reference int) (int, error) {
	day := reference
	for {
		day--
		if g.matches(day) || day <= 0 {
			break
		}
	}
	if day <= 0 {
		return 0, errNoSuchValue
	}
	return day, nil
}

func (g *everyDayOfWeekGenerator) between(start, end int) ([]int, error) {
	var values []int
	reference := start
	for reference < end {
		next, err := g.next(reference)
		if err != nil {
			break
		}
		reference = next
		if reference < end {
			values = append(values, reference)
		}
	}
	return values, nil
}

func (g *everyDayOfWeekGenerator) isMatch(value int) (bool, error) {
	return g.matches(value), nil
}

// matches reports whether the day of the month falls on one of the selected weekdays.
func (g *everyDayOfWeekGenerator) matches(value int) bool {
	if value > g.lastDayOfMonth || value < 1 {
		return false
	}
	return g.valid[javatime.MonthStart(g.year, g.month).PlusDays(int64(value-1)).DayOfWeek()]
}
