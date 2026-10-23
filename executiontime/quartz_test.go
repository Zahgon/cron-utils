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
// Ported to Go from com/cronutils/model/time/ExecutionTimeQuartzIntegrationTest.java.
//
// Tests that started from the current time start from fixed reference times. lastDayOfTheWeek also compared with
// the Quartz scheduler library; only the cron-utils expectation is kept. The disabled case #192 is not ported.

package executiontime_test

import (
	"fmt"
	"testing"

	"github.com/jmrozanec/cron-utils/descriptor"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func quartzET(t *testing.T, expr string) executiontimeT {
	t.Helper()
	return executiontimeT{forCron(t, mustParse(t, model.Quartz, expr)), expr}
}

// executiontimeT is an execution time with the expression it was built from, for messages.
type executiontimeT struct {
	et   executiontime.ExecutionTime
	expr string
}

func TestQuartzEverySecond(t *testing.T) {
	e := quartzET(t, "* * * * * ? *")
	for _, ref := range references(t) {
		now := ref.TruncatedTo(javatime.Seconds)
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", now, e.expr), now.PlusSeconds(1), next(t, e.et, now))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", now, e.expr), now.MinusSeconds(1), last(t, e.et, now))
		if d := timeToNext(t, e.et, now); d.Seconds() != 1 || d.Nanos() != 0 {
			t.Errorf("TimeToNextExecution(%s) of %q = %s, want PT1S", now, e.expr, d)
		}
	}
}

func TestQuartzDoesNotIgnoreMonthOrDayOfWeek(t *testing.T) {
	e := quartzET(t, "0 11 11 11 11 ?")
	got := next(t, e.et, at(t, javatime.UTC, 2015, 4, 15, 0, 0, 0))
	assertZoned(t, fmt.Sprintf("NextExecution(2015-04-15T00:00Z) of %q", e.expr), at(t, javatime.UTC, 2015, 11, 11, 11, 11, 0), got)
}

func TestQuartzHourlyIntervalTimeFromLastExecution(t *testing.T) {
	for _, now := range references(t) {
		previousHour := now.PlusHours(-1)
		e := quartzET(t, fmt.Sprintf("0 0 %d * * ?", previousHour.Hour()))
		if d := timeFromLast(t, e.et, now); d.Seconds() > 120*60 {
			t.Errorf("TimeFromLastExecution(%s) of %q = %s, want at most two hours", now, e.expr, d)
		}
	}
}

func TestQuartzShiftToNextDay(t *testing.T) {
	e := quartzET(t, "0/1 * * 1/1 * ? *")
	for _, now := range []javatime.ZonedDateTime{at(t, javatime.UTC, 2016, 8, 5, 23, 59, 59), at(t, javatime.UTC, 2015, 1, 31, 23, 59, 59)} {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", now, e.expr), now.PlusSeconds(1), next(t, e.et, now))
	}
}

func TestQuartzTimeShiftingProperlyDone(t *testing.T) {
	e := quartzET(t, "0 0/10 22 ? * *")
	for _, ref := range references(t) {
		start, err := javatime.OfFields(ref.Year(), ref.Month(), ref.Day(), 15, 27, ref.Second(), ref.Nano(), ref.Zone())
		if err != nil {
			t.Fatalf("javatime.OfFields(%04d-%02d-%02d 15:27:%02d.%09d, %s): %v", ref.Year(), ref.Month(), ref.Day(), ref.Second(), ref.Nano(), ref.Zone(), err)
		}
		if n := next(t, e.et, start); n.Hour() != 22 || n.Minute() != 0 {
			t.Errorf("NextExecution(%s) of %q = %s, want 22:00", start, e.expr, n)
		}
	}
}

func TestQuartzMonthRangeExecutionTime(t *testing.T) {
	quartzET(t, "0 0 0 * JUL-AUG ? *")
	quartzET(t, "0 0/15 * * * ?")
}

func TestQuartzLastAndNextDiffer(t *testing.T) {
	for _, expr := range []string{"0 0 3 ? * 6", "0 0 3 ? * *"} {
		e := quartzET(t, expr)
		for _, now := range references(t) {
			if l, n := last(t, e.et, now), next(t, e.et, now); l.Equal(n) {
				t.Errorf("LastExecution(%s) and NextExecution(%s) of %q are both %s, want different executions", now, now, expr, l)
			}
		}
	}
}

func TestQuartzExecutionTimeForRanges(t *testing.T) {
	e := quartzET(t, "* 10-20 * * * ? 2099")
	if n := next(t, e.et, parseZoned(t, "2016-02-29T11:00:00.000-06:00")); n.Minute() != 10 {
		t.Errorf("NextExecution(2016-02-29T11:00-06:00) of %q = %s, want minute 10", e.expr, n)
	}
}

func TestQuartzLastExecutionTimeForFixedMonth(t *testing.T) {
	e := quartzET(t, "0 30 12 1 9 ? 2010")
	if l := last(t, e.et, parseZoned(t, "2016-01-08T11:00:00.000-06:00")); l.Month() != 9 {
		t.Errorf("LastExecution(2016-01-08T11:00-06:00) of %q = %s, want September", e.expr, l)
	}
}

func TestQuartzNextExecutionRightDayOfWeek(t *testing.T) {
	for _, expr := range []string{"0 * * ? 5 1 *", "0 * * ? * 1 2099"} {
		e := quartzET(t, expr)
		if n := next(t, e.et, parseZoned(t, "2016-03-06T20:17:28.000-03:00")); n.DayOfWeek() != 7 {
			t.Errorf("NextExecution(2016-03-06T20:17:28-03:00) of %q = %s, want a Sunday", expr, n)
		}
	}
}

func TestQuartzIllegalQuestionMarkValue(t *testing.T) {
	if _, err := parser.New(definition(model.Quartz)).Parse("0 0 12 1W ? *"); err == nil {
		t.Errorf("Parse(%q) with the Quartz definition: no error, want an error", "0 0 12 1W ? *")
	}
}

func TestQuartzNextExecutionProducingInvalidValues(t *testing.T) {
	e := quartzET(t, "0 0 18 ? * MON")
	n := next(t, e.et, parseZoned(t, "2016-03-18T19:02:51.424+09:00"))
	nn := next(t, e.et, n)
	for _, d := range []javatime.ZonedDateTime{n, nn} {
		if d.DayOfWeek() != 1 || d.Hour() != 18 {
			t.Errorf("execution of %q after 2016-03-18T19:02:51.424+09:00 = %s, want a Monday at 18:00", e.expr, d)
		}
	}
}

func TestQuartzMultipleMinuteIntervalTimeFromLastExecution(t *testing.T) {
	e := quartzET(t, "* 8-10,23-25,38-40,53-55 * * * ? *")
	cases := []struct {
		hour, minute, second int
		want                 int64
	}{
		{3, 1, 0, 301}, {13, 8, 4, 1}, {13, 11, 0, 1}, {13, 12, 2, 63},
	}
	for _, c := range cases {
		now := at(t, javatime.UTC, 2024, 3, 13, c.hour, c.minute, c.second)
		if d := timeFromLast(t, e.et, now); d.Seconds() != c.want {
			t.Errorf("TimeFromLastExecution(%s) of %q = %s, want %ds", now, e.expr, d, c.want)
		}
	}
}

func TestQuartzMultipleHourIntervalMatch(t *testing.T) {
	e := quartzET(t, "* * 21-23,0-4 * * ?")
	for hour, want := range map[int]bool{20: false, 21: true, 0: true, 4: true, 5: false} {
		d := at(t, javatime.UTC, 2014, 9, 20, hour, 0, 0)
		if got := isMatch(t, e.et, d); got != want {
			t.Errorf("IsMatch(%s) of %q = %v, want %v", d, e.expr, got, want)
		}
	}
}

func TestQuartzDaylightSavingsSwitch(t *testing.T) {
	e := quartzET(t, "* 0/2 * * * ?")
	denver := zone(t, "America/Denver")
	before := next(t, e.et, at(t, denver, 2016, 3, 13, 1, 59, 59))
	if before.Hour() != 3 || before.Minute() != 0 {
		t.Fatalf("NextExecution(2016-03-13T01:59:59 America/Denver) of %q = %s, want 03:00", e.expr, before)
	}
	if after := next(t, e.et, before.PlusMinutes(1)); after.Hour() != 3 || after.Minute() != 2 {
		t.Errorf("NextExecution(%s) of %q = %s, want 03:02", before.PlusMinutes(1), e.expr, after)
	}
	if n := next(t, e.et, parseZoned(t, "2016-03-14T00:00:59Z")); n.Hour() != 0 || n.Minute() != 2 {
		t.Errorf("NextExecution(2016-03-14T00:00:59Z) of %q = %s, want 00:02", e.expr, n)
	}
}

func TestQuartzDayOfMonthEdges(t *testing.T) {
	z := zone(t, "UTC")
	cases := []struct {
		expr       string
		from, want javatime.ZonedDateTime
	}{
		{"0 0 0 31 * ?", at(t, z, 2016, 11, 1, 0, 0, 0), at(t, z, 2016, 12, 31, 0, 0, 0)},
		{"0 * * ? * *", at(t, z, 2016, 8, 31, 10, 10, 0), at(t, z, 2016, 8, 31, 10, 11, 0)},
		{"0 * * ? * *", at(t, z, 2016, 8, 30, 23, 59, 0), at(t, z, 2016, 8, 31, 0, 0, 0)},
		{"0 0 8 31 * ?", at(t, utc(t), 2017, 4, 10, 0, 0, 0), at(t, utc(t), 2017, 5, 31, 8, 0, 0)},
		{"0 0 8 31 * ?", at(t, utc(t), 2017, 1, 10, 0, 0, 0), at(t, utc(t), 2017, 1, 31, 8, 0, 0)},
	}
	for _, c := range cases {
		checkNextExecution(t, model.Quartz, c.expr, c.from, c.want)
	}
}

func TestQuartzFirstWorkDayOfMonth(t *testing.T) {
	e := quartzET(t, "0 0 12 1W * ? *")
	if n := next(t, e.et, parseZoned(t, "2016-03-29T00:00:59Z")); n.Day() != 1 {
		t.Errorf("NextExecution(2016-03-29T00:00:59Z) of %q = %s, want April 1st", e.expr, n)
	}
}

func TestQuartzDayOfWeekNamesAndNumbers(t *testing.T) {
	friday := at(t, javatime.UTC, 2016, 4, 22, 0, 0, 0)
	byNumber := next(t, quartzET(t, "0 0 12 ? * 2,3,4,5,6 *").et, friday)
	byName := next(t, quartzET(t, "0 0 12 ? * MON,TUE,WED,THU,FRI *").et, friday)
	if !byNumber.Equal(byName) {
		t.Errorf("NextExecution(%s) of %q = %s, of %q = %s; want the same", friday, "0 0 12 ? * 2,3,4,5,6 *", byNumber, "0 0 12 ? * MON,TUE,WED,THU,FRI *", byName)
	}
}

func TestQuartzMinimumInterval(t *testing.T) {
	cases := map[string]int64{
		"* * * * * ?": 1, "*   *    *  *       * ?": 1, "0/1 * * * * ?": 1, "*/1 * * * * ?": 1,
		"0 * * * * ?": 60, "0 */1 * * * ?": 60, "0 */5 * * * ?": 300, "0 0 * * * ?": 3600, "0 0 */3 * * ?": 10800,
		"0 0 0 * * ?": 86400,
	}
	coolDay := at(t, javatime.UTC, 2016, 1, 1, 0, 0, 0)
	for expr, want := range cases {
		e := quartzET(t, expr)
		if d := timeToNext(t, e.et, next(t, e.et, coolDay)); d.Seconds() != want || d.Nanos() != 0 {
			t.Errorf("minimum interval of %q = %s, want %ds", expr, d, want)
		}
	}
}

func TestQuartzDayOfWeekFiltersIssue110(t *testing.T) {
	z := zone(t, "UTC")
	wednesdayNov9 := at(t, z, 2016, 11, 9, 1, 1, 0)
	startOfThursdayNov10 := wednesdayNov9.PlusDays(1).TruncatedTo(javatime.Days)
	thursdayOct27 := at(t, z, 2016, 10, 27, 23, 55, 0)
	for _, expr := range []string{"0 0/1 * ? * 5", "0 0/1 * ? * 2,5", "0 0/1 * ? * THU", "0 0/1 * ? * THU,SAT",
		"0 0/1 * ? * 5-6", "0 0/1 * ? * THU-FRI"} {
		checkNextExecution(t, model.Quartz, expr, wednesdayNov9, startOfThursdayNov10)
		checkNextExecution(t, model.Quartz, expr, thursdayOct27, thursdayOct27.PlusMinutes(1))
	}
	endOfThursdayNov3 := at(t, z, 2016, 11, 3, 23, 59, 0)
	cases := []struct {
		expr string
		from javatime.ZonedDateTime
	}{
		{"0 0/1 * ? * 5", endOfThursdayNov3},
		{"0 0/1 * ? * 2,5", endOfThursdayNov3.PlusDays(4)},
		{"0 0/1 * ? * THU", endOfThursdayNov3},
		{"0 0/1 * ? * THU,SAT", endOfThursdayNov3.PlusDays(2)},
		{"0 0/1 * ? * 5-6", endOfThursdayNov3.PlusDays(1)},
		{"0 0/1 * ? * THU-FRI", endOfThursdayNov3.PlusDays(1)},
	}
	for _, c := range cases {
		checkNextExecution(t, model.Quartz, c.expr, c.from, startOfThursdayNov10)
	}
}

func TestQuartzDescriptionOfDayOfWeek(t *testing.T) {
	d, err := descriptor.Instance()
	if err != nil {
		t.Fatalf("descriptor.Instance() for the default locale: %v", err)
	}
	got, err := d.Describe(mustParse(t, model.Quartz, "0 0 8 ? * SUN *"))
	if err != nil || got != "at 08:00 at Sunday day" {
		t.Errorf("Describe(%q) = %q, %v; want %q", "0 0 8 ? * SUN *", got, err, "at 08:00 at Sunday day")
	}
}

func TestQuartzNextExecutionTimeProperlySet(t *testing.T) {
	e := quartzET(t, "0 5/15 * * * ? *")
	start := atNano(t, javatime.UTC, 2016, 7, 30, 15, 0, 0, 527)
	if l := last(t, e.et, start).String(); l != "2016-07-30T14:50Z" {
		t.Errorf("LastExecution(%s) of %q = %s, want 2016-07-30T14:50Z", start, e.expr, l)
	}
	if n := next(t, e.et, start).String(); n != "2016-07-30T15:05Z" {
		t.Errorf("NextExecution(%s) of %q = %s, want 2016-07-30T15:05Z", start, e.expr, n)
	}
	e2 := quartzET(t, "0 3/27 10-14 * * ? *")
	if n := next(t, e2.et, at(t, javatime.UTC, 2016, 1, 1, 10, 0, 0)).String(); n != "2016-01-01T10:03Z" {
		t.Errorf("NextExecution(2016-01-01T10:00Z) of %q = %s, want 2016-01-01T10:03Z", e2.expr, n)
	}
}

func TestQuartzMatchOverRangeOfDates(t *testing.T) {
	mustParse(t, model.Quartz, "0 0 0 31 * ?")
	e := quartzET(t, "* * * 05 05 ? 2004")
	z := zone(t, "UTC")
	end := at(t, z, 2004, 5, 6, 1, 0, 0)
	for start := at(t, z, 2004, 5, 5, 23, 55, 0); start.Compare(end) < 0; start = start.PlusMinutes(1) {
		isMatch(t, e.et, start)
	}
	weekend := quartzET(t, "0 0 10 ? * SAT-SUN")
	for _, now := range references(t) {
		next(t, weekend.et, now)
	}
}

func TestQuartzLastDayOfTheWeek(t *testing.T) {
	e := quartzET(t, "0 0 0 ? * L *")
	thursday := at(t, zone(t, "UTC"), 2016, 12, 22, 0, 0, 0)
	n := next(t, e.et, thursday)
	if want := thursday.PlusDays(2); n.Compare(want) != 0 {
		t.Errorf("NextExecution of %s = %s, want %s (Saturday)", e.expr, n, want)
	}
}

func TestQuartzLastExecutionDec31(t *testing.T) {
	e := quartzET(t, "0 0 12 31 12 ? *")
	for _, now := range references(t) {
		last(t, e.et, now)
	}
}

func TestQuartzPeriodicIncrementalHours(t *testing.T) {
	z := zone(t, "UTC")
	e := quartzET(t, "0 0 16-19/2 * * ?")
	want := []javatime.ZonedDateTime{
		at(t, z, 2016, 12, 27, 16, 0, 0), at(t, z, 2016, 12, 27, 18, 0, 0), at(t, z, 2016, 12, 28, 16, 0, 0),
		at(t, z, 2016, 12, 28, 18, 0, 0), at(t, z, 2016, 12, 29, 16, 0, 0),
	}
	d := at(t, z, 2016, 12, 27, 8, 15, 0)
	for _, w := range want {
		from := d
		d = next(t, e.et, d)
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", from, e.expr), w, d)
	}
	d = d.PlusSeconds(1)
	for i := len(want) - 1; i >= 0; i-- {
		from := d
		d = last(t, e.et, d)
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", from, e.expr), want[i], d)
	}
}

func TestQuartzLastExecutionIssue312(t *testing.T) {
	e := quartzET(t, "0 0 20 * * ? *")
	for _, now := range references(t) {
		l := last(t, e.et, now)
		if l.Hour() != 20 || l.Minute() != 0 || l.Second() != 0 {
			t.Errorf("LastExecution(%s) of %q = %s, want 20:00", now, e.expr, l)
		}
		wantDay := now.Date()
		if now.Hour() < 20 {
			wantDay = wantDay.PlusDays(-1)
		}
		if l.Date() != wantDay {
			t.Errorf("LastExecution(%s) of %q = %s, want on %s", now, e.expr, l, wantDay)
		}
	}
}

func TestQuartzLastExecutionIssue402(t *testing.T) {
	z := javatime.UTC
	e := quartzET(t, "0 0 0 1 3 ? 2001-2020/2")
	current := at(t, z, 2015, 1, 15, 0, 0, 0)
	n := next(t, e.et, current)
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", current, e.expr), at(t, z, 2015, 3, 1, 0, 0, 0), n)
	assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", current, e.expr), at(t, z, 2013, 3, 1, 0, 0, 0), last(t, e.et, current))
	assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", n, e.expr), at(t, z, 2013, 3, 1, 0, 0, 0), last(t, e.et, n))
	nextBeforeStart := next(t, e.et, at(t, z, 1996, 1, 15, 0, 0, 0))
	assertZoned(t, fmt.Sprintf("NextExecution(1996-01-15T00:00Z) of %q", e.expr), at(t, z, 2001, 3, 1, 0, 0, 0), nextBeforeStart)
	noLast(t, e.et, nextBeforeStart)
	afterEnd := at(t, z, 2025, 1, 15, 0, 0, 0)
	noNext(t, e.et, afterEnd)
	assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", afterEnd, e.expr), at(t, z, 2019, 3, 1, 0, 0, 0), last(t, e.et, afterEnd))
}

// fourthSunday returns the fourth Sunday of a month.
func fourthSunday(year, month int) int {
	first := javatime.MonthStart(year, month).DayOfWeek()
	return 1 + (7-first)%7 + 21
}

func TestQuartzLastExecutionIssue424(t *testing.T) {
	e := quartzET(t, "0 0 12 ? * SUN#4 2020")
	d := at(t, javatime.UTC, 2021, 1, 1, 0, 0, 0)
	for index := 0; index < 12; index++ {
		from := d
		d = last(t, e.et, d)
		want := fmt.Sprintf("2020-%02d-%02dT12:00", 12-index, fourthSunday(2020, 12-index))
		if d.LocalDateTime().String() != want {
			t.Errorf("LastExecution(%s) of %q = %s, want %s", from, e.expr, d, want)
		}
	}
}

func TestQuartzBoundary(t *testing.T) {
	z := javatime.UTC
	e := quartzET(t, "0 0 12 * * ?")
	assertZoned(t, fmt.Sprintf("NextExecution(1900-01-01T00:00Z) of %q", e.expr), at(t, z, 1970, 1, 1, 12, 0, 0), next(t, e.et, at(t, z, 1900, 1, 1, 0, 0, 0)))
	assertZoned(t, fmt.Sprintf("LastExecution(2150-01-01T00:00Z) of %q", e.expr), at(t, z, 2099, 12, 31, 12, 0, 0), last(t, e.et, at(t, z, 2150, 1, 1, 0, 0, 0)))
}
