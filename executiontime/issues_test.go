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
// Ported to Go from com/cronutils/{Issue143Test,Issue200Test,Issue215Test,Issue218Test,Issue223Test,Issue228Test,
// Issue293Test,Issue305Test,Issue319Test,Issue329Test,Issue332Test,Issue340Test,Issue363Test,Issue382Test,
// Issue388Test,Issue394Test,Issue403Test,Issue404Test,Issue406Test,Issue418Test,Issue421Test,Issue430Test,
// Issue444Test,Issue446Test,Issue470Test,Issue480Test,Issue503Test,Issue512Test,Issue528Test,
// Issue55UnexpectedExecutionTimes,OpenIssuesTest}.java.
//
// The original tests ran in the system default time-zone; these use UTC. Tests that read the current time use fixed
// dates. Not ported: Issue367Test, Issue423Test and Issue539Test (disabled in the original), the disabled cases of
// Issue403Test and Issue404Test, and OpenIssuesTest's printing (it asserts nothing; the schedules are still
// computed).

package executiontime_test

import (
	"fmt"
	"testing"

	"github.com/jmrozanec/cron-utils/builder"
	"github.com/jmrozanec/cron-utils/descriptor"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/mapper"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func TestIssue143LastExecution(t *testing.T) {
	z := utc(t)
	current := at(t, z, 2016, 12, 20, 12, 0, 0)
	cases := []struct {
		expr string
		want javatime.ZonedDateTime
	}{
		{"0 0 12 31 12 ? *", at(t, z, 2015, 12, 31, 12, 0, 0)},
		{"0 0 12 ? 12 SAT#5 *", at(t, z, 2012, 12, 29, 12, 0, 0)},
		{"0 0 12 31 1/1 ? *", at(t, z, 2016, 10, 31, 12, 0, 0)},
		{"0 0 12 ? 1/1 SAT#5 *", at(t, z, 2016, 10, 29, 12, 0, 0)},
	}
	for _, c := range cases {
		et := forCron(t, mustParse(t, model.Quartz, c.expr))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", current, c.expr), c.want, last(t, et, current))
	}
}

func TestIssue200MatchIgnoresNanoseconds(t *testing.T) {
	cron := mustParse(t, model.Quartz, "00 00 10 * * ?")
	if err := cron.Validate(); err != nil {
		t.Fatalf("Validate() of %q: %v; want no error", "00 00 10 * * ?", err)
	}
	et := forCron(t, cron)
	for _, nano := range []int{3, 0} {
		d := atNano(t, utc(t), 1999, 7, 18, 10, 0, 0, nano)
		if !isMatch(t, et, d) {
			t.Errorf("IsMatch(%s) of %q = false, want true: nanoseconds must not affect matching", d, exprOf(et))
		}
	}
}

func checkNextExecution(t *testing.T, ct model.CronType, expr string, start, want javatime.ZonedDateTime) {
	t.Helper()
	et := forCron(t, mustParse(t, ct, expr))
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", start, expr), want, next(t, et, start))
}

func TestIssue215WorkdayBugWithNextMonth(t *testing.T) {
	z := utc(t)
	pairs := [][2]javatime.ZonedDateTime{
		{at(t, z, 2017, 7, 7, 10, 0, 0), at(t, z, 2017, 7, 10, 8, 0, 0)},
		{at(t, z, 2017, 8, 31, 10, 0, 0), at(t, z, 2017, 9, 1, 8, 0, 0)},
		{at(t, z, 2017, 6, 30, 10, 0, 0), at(t, z, 2017, 7, 3, 8, 0, 0)},
		{at(t, z, 2017, 9, 29, 10, 0, 0), at(t, z, 2017, 10, 2, 8, 0, 0)},
	}
	for _, p := range pairs {
		checkNextExecution(t, model.Quartz, "0 0 8 ? * MON-FRI", p[0], p[1])
		checkNextExecution(t, model.Cron4j, "0 8 * * MON-FRI", p[0], p[1])
	}
}

func TestIssue215FridayToSaturday(t *testing.T) {
	z := utc(t)
	pairs := [][2]javatime.ZonedDateTime{
		{at(t, z, 2017, 3, 28, 0, 0, 0), at(t, z, 2017, 3, 31, 8, 0, 0)},
		{at(t, z, 2017, 3, 31, 9, 0, 0), at(t, z, 2017, 4, 1, 8, 0, 0)},
		{at(t, z, 2017, 7, 10, 0, 0, 0), at(t, z, 2017, 7, 14, 8, 0, 0)},
		{at(t, z, 2017, 7, 15, 0, 0, 0), at(t, z, 2017, 7, 15, 8, 0, 0)},
		{at(t, z, 2010, 12, 31, 0, 0, 0), at(t, z, 2010, 12, 31, 8, 0, 0)},
		{at(t, z, 2010, 12, 31, 9, 0, 0), at(t, z, 2011, 1, 1, 8, 0, 0)},
	}
	for _, p := range pairs {
		checkNextExecution(t, model.Quartz, "0 0 8 ? * FRI-SAT", p[0], p[1])
		checkNextExecution(t, model.Cron4j, "0 8 * * FRI-SAT", p[0], p[1])
	}
}

func TestIssue218ExecutionTimeOfOptionalDayOfWeek(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().WithMinutes().And().WithHours().And().WithDayOfWeek().Optional().And())
	et := forCron(t, mustParseWith(t, d, "0-59 7-16 MON-FRI"))
	date := at(t, utc(t), 2024, 5, 14, 9, 30, 0)
	if _, err := et.IsMatch(date); err != nil {
		t.Errorf("IsMatch(%s) of %q with an optional day-of-week field: %v; want no error", date, "0-59 7-16 MON-FRI", err)
	}
}

func TestIssue223EveryWednesdayNextExecution(t *testing.T) {
	for _, expr := range []string{"* * * * 3", "* * */1 * 3"} {
		et := forCron(t, mustParse(t, model.Unix, expr))
		got := next(t, et, parseZoned(t, "2017-09-05T11:31:55.407-05:00"))
		assertZoned(t, fmt.Sprintf("NextExecution(2017-09-05T11:31:55.407-05:00) of %q", expr), parseZoned(t, "2017-09-06T00:00-05:00"), got)
	}
}

func TestIssue228DayOfWeekAndDayOfMonthMatchedTogether(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithMinutes().WithStrictRange().And().
		WithHours().WithStrictRange().And().
		WithDayOfMonth().WithStrictRange().And().
		WithMonth().WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).WithStrictRange().And().
		MatchDayOfWeekAndDayOfMonth())
	start := parseZoned(t, "2017-09-29T14:46:01.166-07:00")
	cases := map[string]string{
		"0 9 1-7 * 1":     "2017-10-02T09:00-07:00",
		"0 9 1-7 * 1-5":   "2017-10-02T09:00-07:00",
		"0 9 1-7 * 6-7":   "2017-10-01T09:00-07:00",
		"0 9 8-14 * 1-5":  "2017-10-09T09:00-07:00",
		"0 9 22-28 * 6-7": "2017-10-22T09:00-07:00",
	}
	for expr, want := range cases {
		et := forCron(t, mustParseWith(t, d, expr))
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", start, expr), parseZoned(t, want), next(t, et, start))
	}
}

func TestIssue293LastExecutionBeforeFriday(t *testing.T) {
	z := utc(t)
	reference := at(t, z, 2017, 12, 1, 9, 30, 0)
	if reference.DayOfWeek() != 5 {
		t.Fatalf("2017-12-01 day of week = %d, want 5 (Friday)", reference.DayOfWeek())
	}
	want := at(t, z, 2017, 11, 30, 18, 15, 0)
	for _, expr := range []string{"15 18 * 1-11 *", "15 18 * 1-11 0-5", "15 18 * 1-11 4-5", "15 18 * 1-11 1-5"} {
		et := forCron(t, mustParse(t, model.Unix, expr))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", reference, expr), want, last(t, et, reference))
	}
}

func TestIssue305StepOverYears(t *testing.T) {
	z := utc(t)
	et := forCron(t, mustParse(t, model.Quartz, "0 0 0 15 8 ? 2015-2099/2"))
	end := at(t, z, 2020, 12, 31, 0, 0, 0)
	years := map[int]bool{}
	for d := next(t, et, at(t, z, 2015, 1, 1, 0, 0, 0)); !end.IsBefore(d); d = next(t, et, d) {
		years[d.Year()] = true
	}
	if len(years) != 3 || !years[2015] || !years[2017] || !years[2019] {
		t.Errorf("execution years of %q from 2015-01-01 to 2020-12-31 = %v, want 2015, 2017 and 2019", exprOf(et), years)
	}
}

func TestIssue319LastExecutionAroundDaylightSavingChange(t *testing.T) {
	z := zone(t, "Europe/Berlin")
	et := forCron(t, mustParse(t, model.Unix, "00 02 * * * "))
	for month := 1; month < 13; month++ {
		for day := 1; day < 29; day++ {
			last(t, et, at(t, z, 2018, month, day, 2, 0, 0))
		}
	}
}

func TestIssue329ImpossibleDateTerminates(t *testing.T) {
	et := forCron(t, mustParse(t, model.Unix, "0 0 30 2 *"))
	noNext(t, et, at(t, utc(t), 2024, 3, 1, 0, 0, 0))
}

func TestIssue332IsMatchAroundDaylightSavingChange(t *testing.T) {
	et := forCron(t, mustParse(t, model.Unix, "0 6 * * *"))
	isMatch(t, et, at(t, zone(t, "America/Santiago"), 2018, 8, 12, 3, 0, 0))
}

func TestIssue340TimeFromLastExecutionWithDayOfWeekRange(t *testing.T) {
	et := forCron(t, mustParse(t, model.Quartz, "0 0 * ? * MON-SUN *"))
	now := at(t, utc(t), 2024, 3, 13, 15, 42, 17)
	d := timeFromLast(t, et, now)
	if d.Seconds() > 3600 {
		t.Errorf("TimeFromLastExecution(%s) of %q = %s, want at most one hour", now, exprOf(et), d)
	}
}

func TestIssue340DayOfWeekRollover(t *testing.T) {
	et := forCron(t, mustParse(t, model.Quartz, "0 0 5 ? * FRI-TUE *"))
	valid := map[int]bool{1: true, 2: true, 5: true, 6: true, 7: true}
	d := at(t, utc(t), 2024, 3, 13, 15, 42, 17)
	for i := 0; i < 100; i++ {
		d = next(t, et, d)
		if !valid[d.DayOfWeek()] || d.Hour() != 5 || d.Minute() != 0 || d.Second() != 0 {
			t.Fatalf("execution %d of %q = %s, want 05:00:00 on Friday to Tuesday", i, exprOf(et), d)
		}
	}
}

func TestIssue363EverySecondOfMinuteOne(t *testing.T) {
	z := javatime.UTC
	et := forCron(t, mustParse(t, model.Quartz, "* 1 * * * ?"))
	expected := at(t, z, 2019, 1, 1, 0, 1, 0)
	d := next(t, et, at(t, z, 2019, 1, 1, 0, 0, 0))
	assertZoned(t, fmt.Sprintf("NextExecution(2019-01-01T00:00Z) of %q", exprOf(et)), expected, d)
	for i := int64(1); i <= 59; i++ {
		from := d
		d = next(t, et, d)
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", from, exprOf(et)), expected.PlusSeconds(i), d)
	}
	assertZoned(t, fmt.Sprintf("NextExecution(2019-01-01T00:01:59Z) of %q", exprOf(et)), at(t, z, 2019, 1, 1, 1, 1, 0),
		next(t, et, at(t, z, 2019, 1, 1, 0, 1, 59)))
}

func TestIssue363EveryMinuteOne(t *testing.T) {
	z := javatime.UTC
	et := forCron(t, mustParse(t, model.Quartz, "0 1 * * * ?"))
	assertZoned(t, fmt.Sprintf("NextExecution(2019-01-01T00:01Z) of %q", exprOf(et)), at(t, z, 2019, 1, 1, 1, 1, 0), next(t, et, at(t, z, 2019, 1, 1, 0, 1, 0)))
}

func TestIssue382LastExecutionWithMillis(t *testing.T) {
	z := javatime.UTC
	et := forCron(t, mustParse(t, model.Unix, "0 0 * * WED"))
	date := at(t, z, 2019, 6, 12, 0, 0, 0)
	assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", date.PlusNanos(300_000_000), exprOf(et)), date, last(t, et, date.PlusNanos(300_000_000)))
}

func TestIssue388LastAndNextExecutionWithDayOfWeekAndDayOfMonth(t *testing.T) {
	z := javatime.UTC
	d := at(t, z, 2019, 7, 1, 8, 0, 0)
	lastMonthFirstSaturday := at(t, z, 2019, 6, 1, 8, 0, 0)
	nextMonthFirstSaturday := at(t, z, 2019, 7, 6, 8, 0, 0)
	for _, c := range []struct {
		ct   model.CronType
		expr string
	}{{model.Spring, "0 0 8 1-7 * SAT"}, {model.Quartz, "0 0 8 ? * SAT#1"}} {
		et := forCron(t, mustParse(t, c.ct, c.expr))
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", d, c.expr), nextMonthFirstSaturday, next(t, et, d))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", d, c.expr), lastMonthFirstSaturday, last(t, et, d))
	}
}

func TestIssue394EveryMondayAt0900(t *testing.T) {
	et := forCron(t, mustParse(t, model.Spring, "0 0 9 * * MON"))
	from := at(t, utc(t), 2024, 2, 14, 11, 12, 13)
	first := next(t, et, from)
	if first.DayOfWeek() != 1 || first.Hour() != 9 {
		t.Fatalf("NextExecution(%s) of %q = %s, want a Monday at 09:00", from, exprOf(et), first)
	}
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", first, exprOf(et)), first.PlusDays(7), next(t, et, first))
}

func TestIssue403EveryTwoHours(t *testing.T) {
	z := zone(t, "US/Central")
	now := at(t, z, 2019, 10, 30, 18, 8, 50)
	cases := map[string]string{"0 * 0/2 * * ?": "2019-10-30T18:09", "0 0 0/2 * * ?": "2019-10-30T20:00"}
	for expr, want := range cases {
		got := next(t, forCron(t, mustParse(t, model.Quartz, expr)), now)
		if got.LocalDateTime().String() != want {
			t.Errorf("NextExecution(%s) of %q = %s, want local time %s", now, expr, got, want)
		}
	}
}

func TestIssue404TimeFromLastExecution(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().WithMinutes().And().WithHours().And().WithDayOfWeek().And())
	et := forCron(t, mustParseWith(t, d, "0 0 *"))
	saturday := at(t, zone(t, "America/Sao_Paulo"), 2019, 11, 2, 0, 0, 1)
	if got := timeFromLast(t, et, saturday); got.Seconds() != 1 {
		t.Errorf("TimeFromLastExecution(%s) of %q = %s, want PT1S", saturday, exprOf(et), got)
	}
	noon := at(t, utc(t), 2019, 11, 3, 16, 0, 0)
	if got := timeFromLast(t, et, noon); got.ToHours() != 16 {
		t.Errorf("TimeFromLastExecution(%s) of %q = %s, want PT16H", noon, exprOf(et), got)
	}
}

func TestIssue406DayOfWeekIsApplied(t *testing.T) {
	z := utc(t)
	checkNextExecution(t, model.Spring, "0 0 1 * * MON-FRI", at(t, z, 2019, 1, 5, 3, 0, 0), at(t, z, 2019, 1, 7, 1, 0, 0))
}

func TestIssue418QuartzEvery7DaysStartingSunday(t *testing.T) {
	z := utc(t)
	et := forCron(t, mustParse(t, model.Quartz, "0 0 2 ? * 1/7 *"))
	first := next(t, et, at(t, z, 2020, 4, 1, 3, 0, 0))
	assertZoned(t, fmt.Sprintf("NextExecution(2020-04-01T03:00Z) of %q", exprOf(et)), at(t, z, 2020, 4, 5, 2, 0, 0), first)
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", first, exprOf(et)), at(t, z, 2020, 4, 12, 2, 0, 0), next(t, et, first))
}

func TestIssue418InvalidWeekDays(t *testing.T) {
	p := parser.New(definition(model.Quartz))
	cases := map[string]string{"0 0 2 ? * 0/7 *": "Value 0 not in range [1, 7]", "0 0 2 ? * 1/8 *": "Period 8 not in range [1, 7]"}
	for expr, suffix := range cases {
		_, err := p.Parse(expr)
		if err == nil || !hasSuffix(err.Error(), suffix) {
			t.Errorf("Parse(%q) error = %v, want one ending in %q", expr, err, suffix)
		}
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func monthsDefinition(t *testing.T) *model.CronDefinition {
	return mustDefinition(t, model.DefineCron().
		WithMinutes().And().
		WithHours().And().
		WithDayOfWeek().SupportsQuestionMark().And().
		WithDayOfMonth().SupportsL().SupportsQuestionMark().And().
		WithDayOfYear().SupportsQuestionMark().And().
		WithMonth().And().
		MatchDayOfWeekAndDayOfMonth().
		WithCronValidation(model.EnsureEitherDayOfWeekOrDayOfMonth()).
		WithCronValidation(model.EnsureEitherDayOfYearOrMonth()))
}

func mustBuild(t *testing.T, b *builder.CronBuilder) model.Cron {
	t.Helper()
	c, err := b.Instance()
	if err != nil {
		t.Fatalf("CronBuilder.Instance() for a field-by-field cron: %v; want a cron", err)
	}
	return c
}

func everyMonth(d *model.CronDefinition, now javatime.ZonedDateTime, months model.FieldExpression) *builder.CronBuilder {
	return builder.Cron(d).
		WithMinute(model.NewOn(now.Minute())).
		WithHour(model.NewOn(now.Hour())).
		WithDoW(model.QuestionMarkExpression()).
		WithDoM(model.NewOn(now.Day())).
		WithDoY(model.QuestionMarkExpression()).
		WithMonth(months)
}

func TestIssue421IntervalsEveryFifthMonth(t *testing.T) {
	d := monthsDefinition(t)
	now := at(t, utc(t), 2020, 2, 10, 0, 0, 0)
	cases := []struct {
		months model.FieldExpression
		want   [][2]int
	}{
		{model.NewEveryOf(model.NewOn(now.Month()), 5), [][2]int{{2020, 7}, {2020, 12}, {2021, 2}}},
		{model.NewEvery(5), [][2]int{{2020, 6}, {2020, 11}, {2021, 1}}},
	}
	for _, c := range cases {
		cron := mustBuild(t, everyMonth(d, now, c.months))
		et := forCron(t, cron)
		run := now
		for _, want := range c.want {
			from := run
			run = next(t, et, run)
			if run.Year() != want[0] || run.Month() != want[1] {
				t.Errorf("NextExecution(%s) of %q = %s, want %d-%02d", from, cron.AsString(), run, want[0], want[1])
			}
		}
	}
}

func TestIssue421DescriptionEveryXMonths(t *testing.T) {
	d := monthsDefinition(t)
	now := at(t, utc(t), 2024, 6, 3, 14, 25, 0)
	desc, err := descriptor.InstanceFor("en-US")
	if err != nil {
		t.Fatalf("descriptor.InstanceFor(%q): %v", "en-US", err)
	}
	for _, months := range []int{3, 6} {
		cron := mustBuild(t, everyMonth(d, now, model.NewEvery(model.IntegerFieldValue(months))))
		got, err := desc.Describe(cron)
		want := "every " + itoa(months) + " months"
		if err != nil || !contains(got, want) {
			t.Errorf("Describe(%s) = %q, %v; want it to contain %q", cron.AsString(), got, err, want)
		}
	}
}

func TestIssue430NoLastExecutionBeforeFirstYear(t *testing.T) {
	et := forCron(t, mustParse(t, model.Quartz, "0 0 12 30 6 ? 2020/10"))
	d := at(t, javatime.UTC, 2020, 6, 30, 12, 0, 0)
	noLast(t, et, d)
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", d, exprOf(et)), d.PlusYears(10), next(t, et, d))
}

func TestIssue444NextExecutionSameDay(t *testing.T) {
	et := forCron(t, mustParse(t, model.Quartz, "0 40 08 ? * *"))
	start := javatime.OfInstant(1601426291, 898_000_000, utc(t))
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", start, exprOf(et)), at(t, utc(t), 2020, 9, 30, 8, 40, 0), next(t, et, start))
}

func TestIssue446IntervalsEverySixMonths(t *testing.T) {
	d := monthsDefinition(t)
	now := at(t, utc(t), 2020, 4, 25, 0, 0, 0)
	et := forCron(t, mustBuild(t, everyMonth(d, now, model.NewEveryOf(model.NewOn(now.Month()), 6))))
	run := next(t, et, now)
	if run.Year() != 2020 || run.Month() != 10 {
		t.Errorf("first NextExecution(%s) of %q = %s, want 2020-10", now, exprOf(et), run)
	}
	from := run
	run = next(t, et, from)
	if run.Year() != 2021 || run.Month() != 4 {
		t.Errorf("second NextExecution(%s) of %q = %s, want 2021-04", from, exprOf(et), run)
	}
}

type hourMinuteOffset struct{ hour, minute, offset int }

func checkRuns(t *testing.T, et executiontime.ExecutionTime, start javatime.ZonedDateTime, want []hourMinuteOffset) {
	t.Helper()
	run := start
	for i, w := range want {
		run = next(t, et, run)
		if run.Hour() != w.hour || run.Minute() != w.minute || (w.offset != 0 && run.Offset() != w.offset) {
			t.Errorf("run %d of %q from %s = %s, want %02d:%02d (offset %ds)", i+1, exprOf(et), start, run, w.hour, w.minute, w.offset)
		}
	}
}

func TestIssue446DaylightSavingOverlapMinuteNextRun(t *testing.T) {
	d := monthsDefinition(t)
	rome := zone(t, "Europe/Rome")
	start := javatime.FromTime(at(t, mustOffset(t, "+02:00"), 2020, 10, 25, 1, 10, 0).Time(), rome)
	cron := mustBuild(t, builder.Cron(d).
		WithMinute(model.NewEveryOf(model.NewOn(start.Minute()), 30)).
		WithHour(model.AlwaysExpression()).
		WithDoW(model.QuestionMarkExpression()).
		WithDoM(model.NewOn(start.Day())).
		WithDoY(model.QuestionMarkExpression()).
		WithMonth(model.NewOn(start.Month())))
	checkRuns(t, forCron(t, cron), start, []hourMinuteOffset{
		{1, 40, 0}, {2, 10, 7200}, {2, 40, 7200}, {2, 10, 3600}, {2, 40, 3600}, {3, 10, 0}, {3, 40, 0},
	})
}

func TestIssue446DaylightSavingOverlapHourNextRun(t *testing.T) {
	d := monthsDefinition(t)
	rome := zone(t, "Europe/Rome")
	everyHour := func(now javatime.ZonedDateTime) model.Cron {
		return mustBuild(t, builder.Cron(d).
			WithMinute(model.NewOn(now.Minute())).
			WithHour(model.NewEveryOf(model.NewOn(now.Hour()), 1)).
			WithDoW(model.QuestionMarkExpression()).
			WithDoM(model.NewOn(now.Day())).
			WithDoY(model.QuestionMarkExpression()).
			WithMonth(model.NewOn(now.Month())))
	}
	plus2 := mustOffset(t, "+02:00")
	dayBefore := javatime.FromTime(at(t, plus2, 2020, 10, 24, 1, 0, 0).Time(), rome)
	checkRuns(t, forCron(t, everyHour(dayBefore)), dayBefore, []hourMinuteOffset{{2, 0, 7200}, {3, 0, 7200}})
	changeDay := javatime.FromTime(at(t, plus2, 2020, 10, 25, 1, 0, 0).Time(), rome)
	checkRuns(t, forCron(t, everyHour(changeDay)), changeDay, []hourMinuteOffset{{2, 0, 7200}, {2, 0, 3600}, {3, 0, 3600}})
}

func mustOffset(t *testing.T, id string) javatime.Zone {
	t.Helper()
	z, err := javatime.OffsetOf(id)
	if err != nil {
		t.Fatalf("OffsetOf(%q): %v", id, err)
	}
	return z
}

func TestIssue470WithoutYearInPattern(t *testing.T) {
	d := at(t, utc(t), 2021, 3, 23, 0, 0, 0)
	et := forCron(t, mustParse(t, model.Quartz, "0 0 0 23 3 ? *"))
	next(t, et, d)
	last(t, et, d)
	if !isMatch(t, et, d) {
		t.Errorf("IsMatch(%s) of %q = false, want true", d, exprOf(et))
	}
}

func TestIssue470WithYearInPattern(t *testing.T) {
	d := at(t, utc(t), 2021, 3, 23, 0, 0, 0)
	et := forCron(t, mustParse(t, model.Quartz, "0 0 0 23 3 ? 2021"))
	noNext(t, et, d)
	noLast(t, et, d)
	if !isMatch(t, et, d) {
		t.Errorf("IsMatch(%s) of %q = false, want true", d, exprOf(et))
	}
}

func TestIssue480WeeklyNextRun(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithMinutes().And().
		WithHours().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(1).SupportsQuestionMark().And().
		WithDayOfMonth().SupportsL().SupportsQuestionMark().And().
		WithDayOfYear().SupportsQuestionMark().And().
		WithMonth().And().
		WithYear().Optional().WithValidRange(1970, 2099).And().
		MatchDayOfWeekAndDayOfMonth().
		WithCronValidation(model.EnsureEitherDayOfWeekOrDayOfMonth()).
		WithCronValidation(model.EnsureEitherDayOfYearOrMonth()))
	sunday := at(t, utc(t), 2021, 6, 27, 0, 0, 0)
	if sunday.DayOfWeek() != 7 {
		t.Fatalf("2021-06-27 day of week = %d, want 7", sunday.DayOfWeek())
	}
	cron := mustBuild(t, builder.Cron(d).
		WithMinute(model.NewOn(sunday.Minute())).
		WithHour(model.NewOn(sunday.Hour())).
		WithDoW(model.NewOn(sunday.DayOfWeek())).
		WithDoM(model.QuestionMarkExpression()).
		WithDoY(model.QuestionMarkExpression()).
		WithMonth(model.AlwaysExpression()).
		WithYear(model.AlwaysExpression()))
	run := next(t, forCron(t, cron), sunday)
	if want := sunday.PlusDays(7).TruncatedTo(javatime.Minutes); run.TruncatedTo(javatime.Minutes).Compare(want) != 0 {
		t.Errorf("NextExecution(%s) of %q = %s, want %s", sunday, cron.AsString(), run, want)
	}
}

func TestIssue503CustomQuarterlySchedule(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().WithValidRange(1, 31).SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().And().
		WithYear().WithValidRange(1900, 2099).Optional().And().
		WithDayOfYear().SupportsQuestionMark().WithValidRange(1, 366).Optional().And().
		WithCronValidation(model.EnsureEitherDayOfYearOrMonth()).
		WithCronValidation(model.EnsureEitherDayOfWeekOrDayOfMonth()))
	z := utc(t)
	var want []javatime.ZonedDateTime
	for i := 0; i < 4; i++ {
		date, err := javatime.OfYearDay(2017, 47+i*91)
		if err != nil {
			t.Fatalf("javatime.OfYearDay(2017, %d): %v", 47+i*91, err)
		}
		want = append(want, javatime.OfDateTime(date, javatime.Midnight, z))
	}
	for _, expr := range []string{"0 0 0 ? * ? * 47/91", "0 0 0 ? * ? 2017 47/91"} {
		et := forCron(t, mustParseWith(t, d, expr))
		end := want[len(want)-1].PlusDays(1)
		var got []javatime.ZonedDateTime
		for d := want[0].MinusSeconds(1); ; {
			n, ok, err := et.NextExecution(d)
			if err != nil {
				t.Fatalf("NextExecution(%s) of %q: %v; want no error", d, expr, err)
			}
			if !ok || !n.IsBefore(end) {
				break
			}
			got = append(got, n)
			d = n
		}
		if len(got) != len(want) {
			t.Fatalf("executions of %q from %s = %v, want %v", expr, want[0].MinusSeconds(1), got, want)
		}
		for i := range want {
			assertZoned(t, fmt.Sprintf("execution %d of %q", i, expr), want[i], got[i])
		}
	}
}

func TestIssue512PreviousMonthIsShorter(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().
		WithYear().And().WithMonth().And().WithDayOfMonth().And().WithHours().And().WithMinutes().And().WithSeconds().And())
	cron := mustBuild(t, builder.Cron(d).
		WithYear(model.NewOn(2022)).WithMonth(model.NewOn(3)).WithDoM(model.NewOn(30)).
		WithHour(model.NewOn(9)).WithMinute(model.NewOn(55)).WithSecond(model.NewOn(0)))
	noLast(t, forCron(t, cron), at(t, utc(t), 2022, 3, 30, 9, 54, 0))
}

func rebootDefinition(t *testing.T) *model.CronDefinition {
	return mustDefinition(t, model.DefineCron().WithSupportedNickname(model.Reboot))
}

func TestIssue528RebootExecutionTime(t *testing.T) {
	et := forCron(t, mustParseWith(t, rebootDefinition(t), "@reboot"))
	now := at(t, utc(t), 2024, 1, 1, 0, 0, 0)
	noNext(t, et, now)
	noLast(t, et, now)
}

func TestIssue528RebootDescription(t *testing.T) {
	desc, err := descriptor.InstanceFor("en-GB")
	if err != nil {
		t.Fatalf("descriptor.InstanceFor(%q): %v", "en-GB", err)
	}
	got, err := desc.Describe(mustParseWith(t, rebootDefinition(t), "@reboot"))
	if err != nil || got != "on reboot" {
		t.Errorf("Describe(@reboot) = %q, %v; want %q", got, err, "on reboot")
	}
}

func unixLike(nicknames ...model.CronNickname) *model.CronDefinitionBuilder {
	b := model.DefineCron().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).WithStrictRange().And().
		WithMonth().WithValidRange(1, 12).WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).WithStrictRange().And()
	for _, n := range nicknames {
		b.WithSupportedNickname(n)
	}
	return b
}

func TestIssue528RebootMapping(t *testing.T) {
	cron := mustParseWith(t, rebootDefinition(t), "@reboot")
	mapped, err := mapper.SameCronMapper(mustDefinition(t, unixLike(model.Reboot))).Map(cron)
	if err != nil || mapped.AsString() != cron.AsString() {
		t.Errorf("Map(%q) to a Unix-like definition with @reboot = %v, %v; want %q", cron.AsString(), mapped, err, cron.AsString())
	}
	if _, err := mapper.SameCronMapper(mustDefinition(t, unixLike())).Map(cron); err == nil {
		t.Errorf("Map(%q) to a Unix-like definition without @reboot = no error; want an error", cron.AsString())
	}
}

func issue55Definition(t *testing.T) *model.CronDefinition {
	return mustDefinition(t, model.DefineCron().
		WithMinutes().And().
		WithHours().And().
		WithDayOfMonth().SupportsHash().SupportsL().SupportsW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithIntMapping(7, 0).SupportsHash().SupportsL().SupportsW().SupportsQuestionMark().And().
		WithYear().Optional().And().
		WithCronValidation(&model.CronConstraint{
			Description: "Both, a day-of-week AND a day-of-month parameter, are not supported.",
			Validate: func(c model.Cron) (bool, error) {
				dom, _ := c.Retrieve(model.DayOfMonth)
				dow, _ := c.Retrieve(model.DayOfWeek)
				_, domQ := dom.Expression.(*model.QuestionMark)
				_, dowQ := dow.Expression.(*model.QuestionMark)
				if !domQ {
					return dowQ, nil
				}
				return !dowQ, nil
			},
		}))
}

func countInstants(t *testing.T, et executiontime.ExecutionTime, start, end javatime.ZonedDateTime) int {
	t.Helper()
	n := 0
	for d := next(t, et, start); d.IsBefore(end); d = next(t, et, d) {
		n++
	}
	return n
}

func TestIssue55OnceEveryThreeDaysNoInstantsWithinTwoDays(t *testing.T) {
	start := at(t, javatime.UTC, 0, 1, 1, 0, 0, 0)
	et := forCron(t, mustParseWith(t, issue55Definition(t), "0 0 */3 * ?"))
	if n := countInstants(t, et, start, start.PlusDays(2)); n != 0 {
		t.Errorf("executions of %q from %s over two days = %d, want 0", exprOf(et), start, n)
	}
}

func TestIssue55OnceAMonthTwelveInstantsInYear(t *testing.T) {
	start := at(t, javatime.UTC, 0, 1, 1, 0, 0, 0)
	et := forCron(t, mustParseWith(t, issue55Definition(t), "0 12 L * ?"))
	if n := countInstants(t, et, start, start.PlusYears(1)); n != 12 {
		t.Errorf("executions of %q from %s over a year = %d, want 12", exprOf(et), start, n)
	}
}

func TestOpenIssuesNextWeekday(t *testing.T) {
	for _, start := range []string{"2015-11-20T15:15Z", "2015-11-27T15:15Z"} {
		for dow := 0; dow <= 6; dow++ {
			expr := "0 6 * * " + itoa(dow)
			got := next(t, forCron(t, mustParse(t, model.Unix, expr)), parseZoned(t, start))
			if got.DayOfWeek()%7 != dow || got.Hour() != 6 {
				t.Errorf("NextExecution(%s) of %q = %s, want day of week %d (0 = Sunday) at 06:00", start, expr, got, dow)
			}
		}
	}
}
