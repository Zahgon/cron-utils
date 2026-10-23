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
// Ported to Go from com/cronutils/model/time/generator/{AlwaysFieldValueGeneratorTest,AndFieldValueGeneratorTest,
// BetweenDayOfWeekValueGeneratorTest,BetweenFieldValueGeneratorTest,EveryDayOfWeekValueGeneratorTest,
// EveryFieldValueGeneratorTest,FieldValueGeneratorFactoryTest,FieldValueGeneratorTest,MockFieldValueGenerator,
// NullFieldValueGeneratorTest,OnDayOfMonthValueGeneratorLTest,OnDayOfMonthValueGeneratorLWTest,
// OnDayOfMonthValueGeneratorTest,OnDayOfMonthValueGeneratorWTest,OnDayOfWeekValueGeneratorHashTest,
// OnDayOfWeekValueGeneratorLTest,OnDayOfWeekValueGeneratorTest,OnFieldValueGeneratorTest}.java and
// com/cronutils/model/time/{TimeNodeTest,NearestValueTest}.java.
//
// Not ported: the matchesFieldExpressionClass and wrong-expression-type constructor cases (the Go generators are
// chosen by a type switch over the expression, so a generator cannot hold the wrong kind of expression), and the
// wrong-field-name factory cases (the Go factories do not check the field name). The random reference values of the
// original are replaced by every value of the same range.

package executiontime

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

func defaultConstraints() *model.FieldConstraints { return model.NewFieldConstraintsBuilder().Build() }

func field(name model.CronFieldName, e model.FieldExpression) *model.CronField {
	return model.NewCronField(name, e, defaultConstraints())
}

// describe names a generator's expression and, for day generators, the month it works on.
func describe(g generator) string {
	switch x := g.(type) {
	case *alwaysGenerator:
		return x.f.Expression.AsString()
	case *andGenerator:
		return x.f.Expression.AsString()
	case *betweenGenerator:
		return x.f.Expression.AsString()
	case *everyGenerator:
		return x.f.Expression.AsString()
	case *onGenerator:
		return x.f.Expression.AsString()
	case *onDayOfMonthGenerator:
		return fmt.Sprintf("%s in %d-%02d", x.f.Expression.AsString(), x.year, x.month)
	case *onDayOfWeekGenerator:
		return fmt.Sprintf("%s in %d-%02d", x.f.Expression.AsString(), x.year, x.month)
	case *betweenDayOfWeekGenerator:
		return fmt.Sprintf("%s in %d-%02d", x.f.Expression.AsString(), x.year, x.month)
	case *andDayOfWeekGenerator:
		return fmt.Sprintf("%s in %d-%02d", x.f.Expression.AsString(), x.year, x.month)
	case *everyDayOfWeekGenerator:
		return fmt.Sprintf("%s in %d-%02d", x.f.Expression.AsString(), x.year, x.month)
	}
	return fmt.Sprintf("%T", g)
}

func expectValue(t *testing.T, g generator, call string, want int, got int, err error) {
	t.Helper()
	if err != nil || got != want {
		t.Errorf("generator %s: %s = %d, %v; want %d", describe(g), call, got, err, want)
	}
}

func expectNoSuchValue(t *testing.T, g generator, call string, got int, err error) {
	t.Helper()
	if !errors.Is(err, errNoSuchValue) {
		t.Errorf("generator %s: %s = %d, %v; want no such value", describe(g), call, got, err)
	}
}

func expectMatch(t *testing.T, g generator, value int, want bool) {
	t.Helper()
	got, err := g.isMatch(value)
	if err != nil || got != want {
		t.Errorf("generator %s: isMatch(%d) = %v, %v; want %v", describe(g), value, got, err, want)
	}
}

func TestAlwaysGenerator(t *testing.T) {
	g := &alwaysGenerator{field(model.Hour, model.AlwaysExpression())}
	for j := 0; j < 10; j++ {
		v, err := g.next(j)
		expectValue(t, g, fmt.Sprintf("next(%d)", j), j+1, v, err)
	}
	for j := 1; j < 10; j++ {
		v, err := g.previous(j)
		expectValue(t, g, fmt.Sprintf("previous(%d)", j), j-1, v, err)
	}
	values, err := g.between(0, 10)
	if err != nil || !slices.Equal(values, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Errorf("generator %s: between(0, 10) = %v, %v; want 1 to 9", describe(g), values, err)
	}
	for v := 0; v < 10; v++ {
		expectMatch(t, g, v, true)
	}
}

func TestAndGenerator(t *testing.T) {
	g := &andGenerator{field(model.Month, model.NewAnd(model.NewOn(0), model.NewOn(1), model.NewOn(2)))}
	for _, v := range []int{0, 1, 2} {
		got, err := g.next(v - 1)
		expectValue(t, g, fmt.Sprintf("next(%d)", v-1), v, got, err)
		got, err = g.previous(v + 1)
		expectValue(t, g, fmt.Sprintf("previous(%d)", v+1), v, got, err)
		expectMatch(t, g, v, true)
	}
	got, err := g.next(2)
	expectNoSuchValue(t, g, "next(2)", got, err)
	got, err = g.previous(0)
	expectNoSuchValue(t, g, "previous(0)", got, err)
	if values, err := g.between(0, 2); err != nil || !slices.Equal(values, []int{1}) {
		t.Errorf("generator %s: between(0, 2) = %v, %v; want [1]", describe(g), values, err)
	}
	expectMatch(t, g, 7, false)
}

func TestBetweenDayOfWeekGenerator(t *testing.T) {
	f := field(model.DayOfWeek, model.NewBetween(model.IntegerFieldValue(1), model.IntegerFieldValue(5)))
	wd, err := model.NewWeekDay(1, true)
	if err != nil {
		t.Fatalf("model.NewWeekDay(1, true): %v", err)
	}
	g, err := newBetweenDayOfWeekGenerator(f, 2016, 2, wd)
	if err != nil {
		t.Fatalf("newBetweenDayOfWeekGenerator(MON-FRI, 2016, 2): %v", err)
	}
	values, err := candidates(g, 1, 29)
	if err != nil {
		t.Fatalf("candidates of generator %s between 1 and 29: %v", describe(g), err)
	}
	for _, span := range [][2]int{{1, 5}, {8, 12}, {15, 19}, {22, 26}, {29, 29}} {
		for d := span[0]; d <= span[1]; d++ {
			if !slices.Contains(values, d) {
				t.Errorf("generator %s: candidates of MON-FRI in February 2016 = %v, missing %d", describe(g), values, d)
			}
		}
	}
}

func TestBetweenGenerator(t *testing.T) {
	g := &betweenGenerator{field(model.Hour, model.NewBetween(model.IntegerFieldValue(0), model.IntegerFieldValue(2)))}
	for j := -1; j < 2; j++ {
		v, err := g.next(j)
		expectValue(t, g, fmt.Sprintf("next(%d)", j), j+1, v, err)
	}
	v, err := g.next(2)
	expectNoSuchValue(t, g, "next(2)", v, err)
	for j := 3; j > 0; j-- {
		v, err := g.previous(j)
		expectValue(t, g, fmt.Sprintf("previous(%d)", j), j-1, v, err)
	}
	v, err = g.previous(0)
	expectNoSuchValue(t, g, "previous(0)", v, err)
	if values, err := g.between(0, 2); err != nil || !slices.Equal(values, []int{1}) {
		t.Errorf("generator %s: between(0, 2) = %v, %v; want [1]", describe(g), values, err)
	}
	if values, err := g.between(3, 12); err != nil || len(values) != 0 {
		t.Errorf("generator %s: between(3, 12) = %v, %v; want none", describe(g), values, err)
	}
	expectMatch(t, g, 0, true)
	expectMatch(t, g, 2, true)
	expectMatch(t, g, 7, false)
}

func TestEveryDayOfWeekGenerator(t *testing.T) {
	f := field(model.DayOfWeek, model.NewEveryOf(model.NewBetween(model.IntegerFieldValue(1), model.IntegerFieldValue(5)), 2))
	wd, err := model.NewWeekDay(1, false)
	if err != nil {
		t.Fatalf("model.NewWeekDay(1, false): %v", err)
	}
	g, err := newEveryDayOfWeekGenerator(f, 2018, 10, wd)
	if err != nil {
		t.Fatalf("newEveryDayOfWeekGenerator(1-5/2, 2018, 10): %v", err)
	}
	for _, c := range [][2]int{{1, 3}, {3, 5}, {5, 8}} {
		v, err := g.next(c[0])
		expectValue(t, g, fmt.Sprintf("next(%d)", c[0]), c[1], v, err)
	}
	for _, c := range [][2]int{{31, 29}, {29, 26}, {26, 24}, {24, 22}} {
		v, err := g.previous(c[0])
		expectValue(t, g, fmt.Sprintf("previous(%d)", c[0]), c[1], v, err)
	}
	values, err := g.between(1, 31)
	if err != nil {
		t.Fatalf("generator %s: between(1, 31): %v", describe(g), err)
	}
	for _, d := range values {
		if dow := javatime.MonthStart(2018, 10).PlusDays(int64(d - 1)).DayOfWeek(); dow != 1 && dow != 3 && dow != 5 {
			t.Errorf("generator %s: between(1, 31) holds %d, day of week %d; want only Monday, Wednesday or Friday", describe(g), d, dow)
		}
	}
	if slices.Contains(values, 1) || slices.Contains(values, 31) {
		t.Errorf("generator %s: between(1, 31) = %v holds an extreme; want neither 1 nor 31", describe(g), values)
	}
	for _, d := range []int{3, 5, 8, 10, 12, 15, 17, 19, 22, 24, 26, 29} {
		if !slices.Contains(values, d) {
			t.Errorf("generator %s: between(1, 31) = %v, missing %d", describe(g), values, d)
		}
	}
	expectMatch(t, g, 16, false)
	expectMatch(t, g, 17, true)
	expectMatch(t, g, 18, false)
}

func TestEveryGenerator(t *testing.T) {
	const step = 7
	g, err := newEveryGenerator(field(model.Hour, model.NewEvery(step)))
	if err != nil {
		t.Fatalf("newEveryGenerator(hour */%d): %v", step, err)
	}
	for j := 1; j <= 10; j++ {
		for back := 1; back <= 3; back++ {
			v, err := g.next(step*j - back)
			expectValue(t, g, fmt.Sprintf("next(%d)", step*j-back), j*step, v, err)
		}
	}
	for j := 0; j < 10; j++ {
		for ahead := 1; ahead <= 3; ahead++ {
			v, err := g.previous(step*j + ahead)
			expectValue(t, g, fmt.Sprintf("previous(%d)", step*j+ahead), j*step, v, err)
		}
	}
	if values, err := g.between(0, step*7); err != nil || len(values) != 6 {
		t.Errorf("generator %s: between(0, 49) = %v, %v; want 6 values", describe(g), values, err)
	}
	expectMatch(t, g, step, true)
	expectMatch(t, g, step+1, false)
}

// stubGenerator matches every value and lists none between the extremes.
type stubGenerator struct{}

func (stubGenerator) next(int) (int, error)           { return 0, nil }
func (stubGenerator) previous(int) (int, error)       { return 0, nil }
func (stubGenerator) between(int, int) ([]int, error) { return nil, nil }
func (stubGenerator) isMatch(int) (bool, error)       { return true, nil }

func TestCandidatesIncludeMatchingExtremes(t *testing.T) {
	values, err := candidates(stubGenerator{}, 1, 2)
	if err != nil || !slices.Equal(values, []int{1, 2}) {
		t.Errorf("candidates(stub matching every value, 1, 2) = %v, %v; want [1 2]", values, err)
	}
}

func TestGeneratorFactory(t *testing.T) {
	kinds := []struct {
		e    model.FieldExpression
		want string
	}{
		{model.AlwaysExpression(), "*executiontime.alwaysGenerator"},
		{model.NewAnd(model.NewOn(1)), "*executiontime.andGenerator"},
		{model.NewBetween(model.IntegerFieldValue(1), model.IntegerFieldValue(7)), "*executiontime.betweenGenerator"},
		{model.NewOn(3), "*executiontime.onGenerator"},
		{model.QuestionMarkExpression(), "executiontime.nullGenerator"},
	}
	for _, k := range kinds {
		g, err := forCronField(field(model.Hour, k.e))
		if err != nil || typeName(g) != k.want {
			t.Errorf("forCronField(%s) = %s, %v; want %s", k.e.AsString(), typeName(g), err, k.want)
		}
	}
	for _, sc := range model.SpecialChars {
		if sc == model.None {
			continue
		}
		if _, err := forCronField(field(model.Hour, model.NewOnSpecial(1, sc, 2))); err == nil {
			t.Errorf("forCronField(hour 1 with special char %s): no error, want an error", sc)
		}
	}
	for _, sc := range []model.SpecialChar{model.L, model.W, model.LW, model.Hash} {
		g, err := dayOfMonthGenerator(field(model.DayOfMonth, model.NewOnSpecial(1, sc, 2)), 2015, 1)
		if err != nil || typeName(g) != "*executiontime.onDayOfMonthGenerator" {
			t.Errorf("dayOfMonthGenerator(day of month 1 with special char %s, 2015, 1) = %s, %v; want *executiontime.onDayOfMonthGenerator", sc, typeName(g), err)
		}
	}
	if g, err := dayOfMonthGenerator(field(model.DayOfMonth, model.NewOn(1)), 2015, 1); err != nil || typeName(g) != "*executiontime.onGenerator" {
		t.Errorf("dayOfMonthGenerator(day of month 1, 2015, 1) = %s, %v; want *executiontime.onGenerator", typeName(g), err)
	}
	wd, _ := model.NewWeekDay(1, false)
	dow := []struct {
		e    model.FieldExpression
		want string
	}{
		{model.NewOn(1), "*executiontime.onDayOfWeekGenerator"},
		{model.NewBetween(model.IntegerFieldValue(1), model.IntegerFieldValue(7)), "*executiontime.betweenDayOfWeekGenerator"},
		{model.NewAnd(model.NewOn(1)), "*executiontime.andDayOfWeekGenerator"},
	}
	for _, k := range dow {
		g, err := dayOfWeekGenerator(field(model.DayOfWeek, k.e), 2015, 1, wd)
		if err != nil || typeName(g) != k.want {
			t.Errorf("dayOfWeekGenerator(%s) = %s, %v; want %s", k.e.AsString(), typeName(g), err, k.want)
		}
	}
}

func typeName(g generator) string {
	switch g.(type) {
	case *alwaysGenerator:
		return "*executiontime.alwaysGenerator"
	case *andGenerator:
		return "*executiontime.andGenerator"
	case *betweenGenerator:
		return "*executiontime.betweenGenerator"
	case *onGenerator:
		return "*executiontime.onGenerator"
	case nullGenerator:
		return "executiontime.nullGenerator"
	case *onDayOfMonthGenerator:
		return "*executiontime.onDayOfMonthGenerator"
	case *onDayOfWeekGenerator:
		return "*executiontime.onDayOfWeekGenerator"
	case *betweenDayOfWeekGenerator:
		return "*executiontime.betweenDayOfWeekGenerator"
	case *andDayOfWeekGenerator:
		return "*executiontime.andDayOfWeekGenerator"
	}
	return "unknown"
}

func TestNullGenerator(t *testing.T) {
	g := nullGenerator{}
	for v := 0; v < 10; v++ {
		got, err := g.next(v)
		expectNoSuchValue(t, g, fmt.Sprintf("next(%d)", v), got, err)
		got, err = g.previous(v)
		expectNoSuchValue(t, g, fmt.Sprintf("previous(%d)", v), got, err)
		expectMatch(t, g, v, false)
	}
	if values, err := g.between(0, 1<<31-1); err != nil || len(values) != 0 {
		t.Errorf("generator %s: between(0, %d) = %v, %v; want none", describe(g), 1<<31-1, values, err)
	}
}

func dayOfMonth(special model.SpecialChar, time, year, month int) *onDayOfMonthGenerator {
	return &onDayOfMonthGenerator{field(model.DayOfMonth, model.NewOnSpecial(time, special, model.DefaultNth)), year, month}
}

func TestOnDayOfMonthLastDay(t *testing.T) {
	g := dayOfMonth(model.L, model.DefaultNth, 2015, 2)
	last := javatime.MonthLength(2015, 2)
	v, err := g.next(1)
	expectValue(t, g, "next(1)", last, v, err)
	v, err = g.next(last)
	expectNoSuchValue(t, g, "next(last)", v, err)
	v, err = g.previous(last + 1)
	expectValue(t, g, "previous(last+1)", last, v, err)
	v, err = g.previous(last)
	expectNoSuchValue(t, g, "previous(last)", v, err)
	if values, err := g.between(1, 32); err != nil || !slices.Equal(values, []int{last}) {
		t.Errorf("generator %s: between(1, 32) = %v, %v; want [%d]", describe(g), values, err, last)
	}
	expectMatch(t, g, last, true)
	expectMatch(t, g, last-1, false)
}

func TestOnDayOfMonthLastWeekday(t *testing.T) {
	for _, c := range []struct{ month, weekday int }{{5, 29}, {2, 27}, {8, 31}} {
		g := dayOfMonth(model.LW, model.DefaultNth, 2015, c.month)
		v, err := g.next(1)
		expectValue(t, g, "next(1)", c.weekday, v, err)
		v, err = g.next(31)
		expectNoSuchValue(t, g, "next(31)", v, err)
		v, err = g.previous(31)
		if c.weekday == 31 {
			expectNoSuchValue(t, g, "previous(31)", v, err)
		} else {
			expectValue(t, g, "previous(31)", c.weekday, v, err)
		}
		v, err = g.previous(1)
		expectNoSuchValue(t, g, "previous(1)", v, err)
		if values, err := g.between(1, 32); err != nil || !slices.Equal(values, []int{c.weekday}) {
			t.Errorf("generator %s: between(1, 32) in month %d = %v, %v; want [%d]", describe(g), c.month, values, err, c.weekday)
		}
		expectMatch(t, g, c.weekday, true)
		expectMatch(t, g, c.weekday-1, false)
	}
}

func TestOnDayOfMonthWithoutSpecialChar(t *testing.T) {
	g := &onDayOfMonthGenerator{field(model.DayOfMonth, model.NewOn(3)), 2015, 2}
	for v := 0; v < 10; v++ {
		got, err := g.next(v)
		expectNoSuchValue(t, g, fmt.Sprintf("next(%d)", v), got, err)
		got, err = g.previous(v)
		expectNoSuchValue(t, g, fmt.Sprintf("previous(%d)", v), got, err)
	}
}

func TestOnDayOfMonthNearestWeekday(t *testing.T) {
	for _, c := range []struct{ month, day, weekday int }{{2, 15, 16}, {2, 7, 6}, {8, 1, 3}} {
		g := dayOfMonth(model.W, c.day, 2015, c.month)
		v, err := g.next(1)
		if c.weekday > 1 {
			expectValue(t, g, "next(1)", c.weekday, v, err)
		}
		v, err = g.next(18)
		expectNoSuchValue(t, g, "next(18)", v, err)
		v, err = g.previous(18)
		expectValue(t, g, "previous(18)", c.weekday, v, err)
		v, err = g.previous(1)
		expectNoSuchValue(t, g, "previous(1)", v, err)
		if values, err := g.between(1, 32); err != nil || !slices.Equal(values, []int{c.weekday}) {
			t.Errorf("generator %s: between(1, 32) for %dW in month %d = %v, %v; want [%d]", describe(g), c.day, c.month, values, err, c.weekday)
		}
		expectMatch(t, g, c.weekday, true)
		expectMatch(t, g, c.weekday-1, false)
	}
}

func dayOfWeek(t *testing.T, on *model.On, year, month int) *onDayOfWeekGenerator {
	return &onDayOfWeekGenerator{field(model.DayOfWeek, on), year, month, model.QuartzWeekDay}
}

func checkSingleDay(t *testing.T, name string, g *onDayOfWeekGenerator, day int) {
	t.Helper()
	v, err := g.next(1)
	expectValue(t, g, name+" next(1)", day, v, err)
	v, err = g.next(day)
	expectNoSuchValue(t, g, name+" next(day)", v, err)
	v, err = g.previous(day + 1)
	expectValue(t, g, name+" previous(day+1)", day, v, err)
	v, err = g.previous(day)
	expectNoSuchValue(t, g, name+" previous(day)", v, err)
	if values, err := g.between(1, day+1); err != nil || len(values) == 0 || values[0] != day {
		t.Errorf("generator %s: %s between(1, %d) = %v, %v; want %d first", describe(g), name, day+1, values, err, day)
	}
	expectMatch(t, g, day, true)
	expectMatch(t, g, day+1, false)
}

func TestOnDayOfWeekHash(t *testing.T) {
	cases := []struct{ month, time, nth, day int }{{5, 2, 4, 25}, {2, 6, 4, 27}, {2, 1, 3, 15}}
	for _, c := range cases {
		checkSingleDay(t, "hash", dayOfWeek(t, model.NewOnSpecial(c.time, model.Hash, c.nth), 2015, c.month), c.day)
	}
}

func TestOnDayOfWeekLast(t *testing.T) {
	cases := []struct{ month, time, day int }{{2, 6, 27}, {3, 6, 27}, {7, 6, 31}}
	for _, c := range cases {
		checkSingleDay(t, "last", dayOfWeek(t, model.NewOnSpecial(c.time, model.L, model.DefaultNth), 2015, c.month), c.day)
	}
}

func TestOnGenerator(t *testing.T) {
	g := &onGenerator{field(model.Hour, model.NewOn(3))}
	v, err := g.next(1)
	expectValue(t, g, "next(1)", 3, v, err)
	v, err = g.next(3)
	expectNoSuchValue(t, g, "next(3)", v, err)
	v, err = g.previous(4)
	expectValue(t, g, "previous(4)", 3, v, err)
	v, err = g.previous(3)
	expectNoSuchValue(t, g, "previous(3)", v, err)
	if values, err := g.between(1, 32); err != nil || !slices.Equal(values, []int{3}) {
		t.Errorf("generator %s: between(1, 32) = %v, %v; want [3]", describe(g), values, err)
	}
	expectMatch(t, g, 3, true)
	expectMatch(t, g, 2, false)
}

func TestNearestValue(t *testing.T) {
	n := nearestValue{value: 1, shifts: 1}
	if n.value != 1 || n.shifts != 1 {
		t.Errorf("nearestValue{value: 1, shifts: 1} = %+v, want value 1, shifts 1", n)
	}
}

func TestTimeNode(t *testing.T) {
	node, err := newTimeNode([]int{2, 4, 6})
	if err != nil {
		t.Fatalf("newTimeNode([2 4 6]): %v", err)
	}
	if !slices.Equal(node.values, []int{2, 4, 6}) {
		t.Errorf("newTimeNode([2 4 6]).values = %v, want [2 4 6]", node.values)
	}
	nextCases := []struct{ ref, shifts, value, shift int }{
		{2, 0, 2, 0}, {4, 0, 4, 0}, {6, 0, 6, 0}, {2, 1, 4, 0}, {4, 1, 6, 0}, {6, 1, 2, 1}, {4, 2, 2, 1},
	}
	for _, c := range nextCases {
		if got := node.nextValue(c.ref, c.shifts); got.value != c.value || got.shifts != c.shift {
			t.Errorf("timeNode [2 4 6]: nextValue(%d, %d) = %+v, want value %d shifts %d", c.ref, c.shifts, got, c.value, c.shift)
		}
	}
	previousCases := []struct{ ref, shifts, value, shift int }{
		{2, 0, 2, 0}, {4, 0, 4, 0}, {6, 0, 6, 0}, {2, 1, 6, 1}, {4, 1, 2, 0}, {6, 1, 4, 0}, {4, 2, 6, 1},
		{5, 1, 4, 0}, {1, 0, 6, 1},
	}
	for _, c := range previousCases {
		if got := node.previousValue(c.ref, c.shifts); got.value != c.value || got.shifts != c.shift {
			t.Errorf("timeNode [2 4 6]: previousValue(%d, %d) = %+v, want value %d shifts %d", c.ref, c.shifts, got, c.value, c.shift)
		}
	}
}

func TestTimeNodeValueFromList(t *testing.T) {
	list := []int{1, 2, 3, 4}
	shift := 0
	if v := valueFromList(list, -1, &shift); shift != 1 || v != 4 {
		t.Errorf("valueFromList([1 2 3 4], -1) = %d with %d shifts, want 4 with 1", v, shift)
	}
	shift = 0
	if v := valueFromList(list, 1, &shift); shift != 0 || v != 2 {
		t.Errorf("valueFromList([1 2 3 4], 1) = %d with %d shifts, want 2 with 0", v, shift)
	}
	if _, err := newTimeNode(nil); err == nil {
		t.Errorf("newTimeNode(nil): no error, want an error")
	}
}
