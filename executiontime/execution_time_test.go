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
// Ported to Go from com/cronutils/model/time/{CompositeExecutionTimeTest,CrossMinuteExecutionTest,
// ExecutionTimeCron4jIntegrationTest,ExecutionTimeCustomDefinitionIntegrationTest,ExecutionTimeNanosTest}.java,
// com/cronutils/model/{ExecutionDatesTest,SingleCronTest}.java.
//
// Tests that started from the current time start from fixed reference times; the random case of
// ExecutionTimeCron4jIntegrationTest runs over a fixed set of day-of-week, day-of-month and month combinations. The
// disabled isMatch-without-seconds cases of ExecutionTimeNanosTest are not ported. SingleCronTest's overlap checks
// run from 2026-01-01.

package executiontime_test

import (
	"fmt"
	"testing"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

// references are the start times used where the original started from the current time.
func references(t *testing.T) []javatime.ZonedDateTime {
	z := utc(t)
	return []javatime.ZonedDateTime{
		atNano(t, z, 2024, 3, 13, 15, 42, 17, 123_000_000),
		atNano(t, z, 2025, 12, 31, 23, 59, 59, 0),
		atNano(t, zone(t, "America/New_York"), 2024, 11, 2, 22, 30, 0, 0),
	}
}

func TestCompositeExecutionTimeLastExecution(t *testing.T) {
	d := definition(model.Quartz)
	c, err := model.NewCompositeCron([]model.Cron{mustParseWith(t, d, "4 * * * * ? *"), mustParseWith(t, d, "6 0 0 * * ? *")})
	if err != nil {
		t.Fatalf("NewCompositeCron(%q, %q): %v", "4 * * * * ? *", "6 0 0 * * ? *", err)
	}
	date := atNano(t, javatime.UTC, 2022, 10, 15, 3, 2, 4, 2)
	got := last(t, forCron(t, c), date)
	if got.LocalDateTime().String() != "2022-10-15T03:02:04" || !got.Zone().Equal(date.Zone()) {
		t.Errorf("LastExecution(%s) of %q = %s, want 2022-10-15T03:02:04Z", date, c.AsString(), got)
	}
}

func TestCrossMinuteExecution(t *testing.T) {
	z := javatime.UTC
	cases := []struct {
		expr       string
		from, want javatime.ZonedDateTime
	}{
		{"*/8 * * * * ?", at(t, z, 2024, 9, 12, 10, 46, 40), at(t, z, 2024, 9, 12, 10, 46, 48)},
		{"*/8 * * * * ?", at(t, z, 2024, 9, 12, 10, 46, 57), at(t, z, 2024, 9, 12, 10, 47, 0)},
		{"*/8 * * * * ?", at(t, z, 2024, 9, 12, 10, 46, 48), at(t, z, 2024, 9, 12, 10, 46, 56)},
		{"0/15 * * * * ?", at(t, z, 2024, 9, 12, 10, 46, 55), at(t, z, 2024, 9, 12, 10, 47, 0)},
		{"0/30 * * * * ?", at(t, z, 2024, 9, 12, 10, 46, 35), at(t, z, 2024, 9, 12, 10, 47, 0)},
	}
	for _, c := range cases {
		checkNextExecution(t, model.Quartz, c.expr, c.from, c.want)
	}
}

func TestCron4jEveryWeekdayAt6(t *testing.T) {
	et := forCron(t, mustParse(t, model.Cron4j, "0 6 * * MON-FRI"))
	for _, lastRun := range references(t) {
		for i := 0; i < 8; i++ {
			n := next(t, et, lastRun)
			if n.DayOfWeek() >= 6 || !lastRun.IsBefore(n) {
				t.Errorf("NextExecution(%s) of %q = %s, want a later weekday", lastRun, exprOf(et), n)
			}
			lastRun = lastRun.PlusDays(1)
		}
	}
}

func TestCron4jEvery2Hours(t *testing.T) {
	et := forCron(t, mustParse(t, model.Cron4j, "0 0/2 * * *"))
	starts := append(references(t), atNano(t, zone(t, "America/Los_Angeles"), 2017, 3, 12, 0, 55, 50, 630))
	for _, lastRun := range starts {
		for i := 0; i < 36; i++ {
			n := next(t, et, lastRun)
			if n.Hour()%2 != 0 || !lastRun.IsBefore(n) {
				t.Errorf("NextExecution(%s) of %q = %s, want a later even hour", lastRun, exprOf(et), n)
			}
			lastRun = lastRun.PlusHours(1)
		}
	}
}

func TestCron4jEvery15Minutes(t *testing.T) {
	et := forCron(t, mustParse(t, model.Cron4j, "0/15 * * * *"))
	for _, lastRun := range references(t) {
		for i := 0; i < 75; i++ {
			n := next(t, et, lastRun)
			if n.Minute()%15 != 0 || !lastRun.IsBefore(n) {
				t.Errorf("NextExecution(%s) of %q = %s, want a later quarter hour", lastRun, exprOf(et), n)
			}
			lastRun = lastRun.PlusMinutes(1)
		}
	}
}

func TestCron4jDayFieldsOverrideAlways(t *testing.T) {
	monday := forCron(t, mustParse(t, model.Cron4j, "0 18 * * 1"))
	first := forCron(t, mustParse(t, model.Cron4j, "0 18 1 * *"))
	for _, now := range references(t) {
		if n := next(t, monday, now); n.DayOfWeek() != 1 || !now.IsBefore(n) {
			t.Errorf("NextExecution of 0 18 * * 1 from %s = %s, want a later Monday", now, n)
		}
		if n := next(t, first, now); n.Day() != 1 || !now.IsBefore(n) {
			t.Errorf("NextExecution of 0 18 1 * * from %s = %s, want a later first of the month", now, n)
		}
		n1 := next(t, monday, now)
		if n2 := next(t, monday, n1); !now.IsBefore(n1) || !n1.IsBefore(n2) {
			t.Errorf("executions of %q from %s: %s then %s, want increasing", exprOf(monday), now, n1, n2)
		}
	}
}

func TestCron4jFixedDayOfMonthAndDayOfWeek(t *testing.T) {
	z := utc(t)
	et := forCron(t, mustParse(t, model.Cron4j, "0 9 1 1 tue"))
	d := at(t, z, 2017, 1, 1, 0, 0, 0)
	for _, year := range []int{2019, 2030, 2036, 2041} {
		d = next(t, et, d)
		want := at(t, z, year, 1, 1, 9, 0, 0)
		assertZoned(t, fmt.Sprintf("NextExecution of %q (run toward %d)", exprOf(et), year), want, d)
	}
}

func TestCron4jDayOfMonthAndDayOfWeek(t *testing.T) {
	for _, c := range []struct{ dow, dom, month int }{{1, 15, 3}, {0, 29, 2}, {5, 13, 7}, {6, 31, 12}, {3, 1, 1}} {
		expr := fmt.Sprintf("0 0 %d %d %d", c.dom, c.month, c.dow)
		et := forCron(t, mustParse(t, model.Cron4j, expr))
		d := references(t)[0]
		for i := 1; i <= 20; i++ {
			d = next(t, et, d)
			if d.DayOfWeek()%7 != c.dow || d.Day() != c.dom || d.Month() != c.month {
				t.Fatalf("execution %d of %s = %s, want day %d of month %d on day of week %d (0 = Sunday)", i, expr, d, c.dom, c.month, c.dow)
			}
		}
	}
}

func TestCron4jStepsPastHour(t *testing.T) {
	checkNextExecution(t, model.Cron4j, "0 10/4 * * *", parseZoned(t, "2019-11-08T00:00Z"), parseZoned(t, "2019-11-08T10:00Z"))
	checkNextExecution(t, model.Cron4j, "20 23/1 * * *", parseZoned(t, "2019-11-08T23:20Z"), parseZoned(t, "2019-11-09T23:20Z"))
}

func secondsDefinition(t *testing.T) *model.CronDefinition {
	return mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().WithDayOfMonth().And().WithMonth().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).And())
}

func TestCustomDefinitionHalfMinutes(t *testing.T) {
	d := secondsDefinition(t)
	z := javatime.UTC
	assertZoned(t, "NextExecution(2015-08-28T12:05:44Z) of \"*/30 * * * * *\"", at(t, z, 2015, 8, 28, 12, 6, 0),
		next(t, forCron(t, mustParseWith(t, d, "*/30 * * * * *")), at(t, z, 2015, 8, 28, 12, 5, 44)))
	assertZoned(t, "NextExecution(2015-08-28T12:05:14Z) of \"0/30 * * * * *\"", at(t, z, 2015, 8, 28, 12, 5, 30),
		next(t, forCron(t, mustParseWith(t, d, "0/30 * * * * *")), at(t, z, 2015, 8, 28, 12, 5, 14)))
}

func TestCustomDefinitionEveryTwoHours(t *testing.T) {
	et := forCron(t, mustParseWith(t, secondsDefinition(t), "0 0 */2 * * *"))
	from := parseZoned(t, "2015-08-28T12:05:14.000-03:00")
	got := next(t, et, from)
	if want := parseZoned(t, "2015-08-28T14:00:00.000-03:00"); got.Compare(want) != 0 {
		t.Errorf("NextExecution(%s) of %q = %s, want %s", from, exprOf(et), got, want)
	}
}

func TestCustomDefinitionBetweenDayOfWeekFirstDayOfMonth(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().And().
		WithMonth().And().
		WithDayOfWeek().WithMondayDoWValue(1).WithValidRange(1, 7).SupportsHash().SupportsL().And().
		WithYear().Optional().And())
	et := forCron(t, mustParseWith(t, d, "30 3 * * MON-FRI"))
	if got := next(t, et, parseZoned(t, "1970-01-01T00:00:00.000-03:00")); got.Day() != 1 {
		t.Errorf("NextExecution(1970-01-01T00:00-03:00) of %q = %s, want day 1", exprOf(et), got)
	}
	if got := next(t, et, parseZoned(t, "1970-01-01T12:00:00.000-03:00")); got.Day() != 2 {
		t.Errorf("NextExecution(1970-01-01T12:00-03:00) of %q = %s, want day 2", exprOf(et), got)
	}
}

func TestCustomDefinitionMatchWithoutPreviousOrNextOccurrence(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithDayOfMonth().SupportsL().SupportsW().And().WithMonth().And().WithYear().And())
	et := forCron(t, mustParseWith(t, d, "05 05 2004"))
	z := utc(t)
	end := at(t, z, 2004, 5, 6, 1, 0, 0)
	for start := at(t, z, 2004, 5, 5, 23, 55, 0); start.Compare(end) < 0; start = start.PlusMinutes(1) {
		if got := isMatch(t, et, start); got != (start.Day() == 5) {
			t.Errorf("IsMatch(%s) of %q = %v, want %v", start, exprOf(et), got, start.Day() == 5)
		}
	}
}

func TestCustomDefinitionOptionalFields(t *testing.T) {
	three := mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().Optional().And().
		WithMonth().Optional().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().
		Optional().And().
		WithYear().WithValidRange(2000, 2099).Optional().And())
	mustParseWith(t, three, "* * 4 3")
	five := mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().
		Optional().And().
		WithYear().WithValidRange(2000, 2099).Optional().And())
	mustParseWith(t, five, "* * 4 3 *")
}

type nanosCase struct {
	now        javatime.ZonedDateTime
	match      bool
	next, last javatime.ZonedDateTime
}

func TestExecutionTimeNanosWithSeconds(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().WithDayOfMonth().And().WithMonth().And().
		WithDayOfWeek().And())
	et := forCron(t, mustParseWith(t, d, "0,1,3 * * * * *"))
	z := javatime.UTC
	ts := func(m, s, n int) javatime.ZonedDateTime { return atNano(t, z, 2024, 2, 7, 13, m, s, n) }
	cases := []nanosCase{
		{ts(46, 0, 0), true, ts(46, 1, 0), ts(45, 3, 0)},
		{ts(46, 0, 999999999), false, ts(46, 1, 0), ts(46, 0, 0)},
		{ts(46, 1, 0), true, ts(46, 3, 0), ts(46, 0, 0)},
		{ts(46, 1, 999999999), false, ts(46, 3, 0), ts(46, 1, 0)},
		{ts(46, 2, 0), false, ts(46, 3, 0), ts(46, 1, 0)},
		{ts(46, 2, 999999999), false, ts(46, 3, 0), ts(46, 1, 0)},
		{ts(46, 3, 0), true, ts(47, 0, 0), ts(46, 1, 0)},
		{ts(46, 3, 999999999), false, ts(47, 0, 0), ts(46, 3, 0)},
	}
	for _, c := range cases {
		if got := isMatch(t, et, c.now); got != c.match {
			t.Errorf("IsMatch(%s) of %q = %v, want %v", c.now, exprOf(et), got, c.match)
		}
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", c.now, exprOf(et)), c.next, next(t, et, c.now))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", c.now, exprOf(et)), c.last, last(t, et, c.now))
	}
}

func TestExecutionTimeNanosWithoutSeconds(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithMinutes().And().WithHours().And().WithDayOfMonth().And().WithMonth().And().WithDayOfWeek().And())
	et := forCron(t, mustParseWith(t, d, "0,1,3 * * * *"))
	z := javatime.UTC
	ts := func(h, m, n int) javatime.ZonedDateTime { return atNano(t, z, 2024, 2, 7, h, m, 0, n) }
	cases := []nanosCase{
		{now: ts(13, 0, 0), next: ts(13, 1, 0), last: ts(12, 3, 0)},
		{now: ts(13, 0, 999999999), next: ts(13, 1, 0), last: ts(13, 0, 0)},
		{now: ts(13, 1, 0), next: ts(13, 3, 0), last: ts(13, 0, 0)},
		{now: ts(13, 1, 999999999), next: ts(13, 3, 0), last: ts(13, 1, 0)},
		{now: ts(13, 2, 0), next: ts(13, 3, 0), last: ts(13, 1, 0)},
		{now: ts(13, 2, 999999999), next: ts(13, 3, 0), last: ts(13, 1, 0)},
		{now: ts(13, 3, 0), next: ts(14, 0, 0), last: ts(13, 1, 0)},
		{now: ts(13, 3, 999999999), next: ts(14, 0, 0), last: ts(13, 3, 0)},
	}
	for _, c := range cases {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", c.now, exprOf(et)), c.next, next(t, et, c.now))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", c.now, exprOf(et)), c.last, last(t, et, c.now))
	}
}

func TestExecutionCountBetweenDates(t *testing.T) {
	et := forCron(t, mustParse(t, model.Cron4j, "0 * * * *"))
	start := at(t, utc(t), 2022, 1, 1, 0, 0, 0)
	n, err := executiontime.CountExecutions(et, start, start.PlusDays(10))
	if err != nil || n != 240 {
		t.Errorf("CountExecutions of %q from %s over 10 days = %d, %v; want 240", exprOf(et), start, n, err)
	}
}

func TestExecutionDatesBetweenDates(t *testing.T) {
	z := utc(t)
	et := forCron(t, mustParse(t, model.Cron4j, "0 * * * *"))
	start := at(t, z, 2022, 1, 1, 0, 0, 0)
	dates, err := executiontime.ExecutionDates(et, start, start.PlusDays(1))
	if err != nil || len(dates) != 24 {
		t.Fatalf("ExecutionDates of %q from %s over a day = %d dates, %v; want 24", exprOf(et), start, len(dates), err)
	}
	for i := 1; i < 24; i++ {
		assertZoned(t, fmt.Sprintf("execution %d of %q from %s", i, exprOf(et), start), at(t, z, 2022, 1, 1, i, 0, 0), dates[i-1])
	}
}

func TestExecutionDatesRejectEmptyWindow(t *testing.T) {
	et := forCron(t, mustParse(t, model.Cron4j, "0 * * * *"))
	start := at(t, utc(t), 2022, 1, 1, 0, 0, 0)
	for _, end := range []javatime.ZonedDateTime{start.PlusDays(-1), at(t, utc(t), 2022, 1, 1, 0, 0, 0)} {
		if _, err := executiontime.ExecutionDates(et, start, end); !cronerr.Is(err, cronerr.IllegalArgument) {
			t.Errorf("ExecutionDates(%s, %s) of %q: error %v, want an invalid-argument error", start, end, exprOf(et), err)
		}
	}
}

func TestSingleCronOverlap(t *testing.T) {
	now := at(t, utc(t), 2026, 1, 1, 0, 0, 0)
	cases := []struct {
		a, b string
		want bool
	}{
		{"0 0 10 1/1 * ? *", "0 0 10 ? * MON *", true},
		{"0 0 10 ? * MON *", "0 0 12 ? * TUE *", false},
		{"* * * * * ? *", "0 * * * * ? *", true},
		{"0 0 12 25 12 ? 2026", "0 0 12 ? 12 FRI 2026", true},
		{"0 0 12 25 12 ? 2025", "0 0 12 ? 12 FRI 2025", false},
		{"0 0/5 * * * ? *", "0 0/10 * * * ? *", true},
		{"0 0 10 ? * MON *", "0 0 11 ? * MON *", false},
		{"0 0 * * * ? *", "0 0 12 * * ? *", true},
		{"0 0 10 1 1 ? *", "0 0 10 1 1 ? 2027", true},
		{"0 0 10 1 1 ? *", "0 0 10 1 2 ? 2027", false},
		{"0 0 12 1 1 ? 2025", "0 0 12 1 1 ? 2026", false},
	}
	for _, c := range cases {
		a, b := mustParse(t, model.Quartz, c.a), mustParse(t, model.Quartz, c.b)
		for _, pair := range [][2]model.Cron{{a, b}, {b, a}} {
			got, err := executiontime.Overlap(pair[0], pair[1], now)
			if err != nil || got != c.want {
				t.Errorf("Overlap(%s, %s) = %v, %v; want %v", pair[0].AsString(), pair[1].AsString(), got, err, c.want)
			}
		}
	}
}
