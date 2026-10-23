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
// Ported to Go from com/cronutils/model/time/{ExecutionTimeQuartzWithDayOfYearExtensionIntegrationTest,
// LastExecutionWithDifferentMonthLengthsTest}.java and com/cronutils/model/definition/TestCronDefinitionsFactory.java.

package executiontime_test

import (
	"fmt"
	"testing"

	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

const (
	biWeeklyFromFirstDayOfYear             = "0 0 0 ? * ? * 1/14"
	firstQuarterBiWeeklyFromFirstDayOfYear = "0 0 0 ? 1-3 ? * 1/14"
	withoutDayOfYear                       = "0 0 0 1 * ? *"
	withoutSpecificDayOfYear               = "0 0 0 1 * ? * ?"
)

// dayOfYearDefinition is Quartz with an optional day-of-year field after the optional year.
func dayOfYearDefinition(t *testing.T) *model.CronDefinition {
	return mustDefinition(t, model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().And().
		WithYear().WithValidRange(1970, 2099).Optional().And().
		WithDayOfYear().SupportsQuestionMark().WithValidRange(1, 366).Optional().And().
		WithCronValidation(model.EnsureEitherDayOfYearOrMonth()).
		WithCronValidation(model.EnsureEitherDayOfWeekOrDayOfMonth()))
}

func TestDayOfYearForCron(t *testing.T) {
	d := dayOfYearDefinition(t)
	for _, e := range []string{biWeeklyFromFirstDayOfYear, firstQuarterBiWeeklyFromFirstDayOfYear, withoutDayOfYear,
		withoutSpecificDayOfYear} {
		forCron(t, mustParseWith(t, d, e))
	}
}

func TestDayOfYearNextAndLastEveryTwoWeeks(t *testing.T) {
	et := forCron(t, mustParseWith(t, dayOfYearDefinition(t), biWeeklyFromFirstDayOfYear))
	for i := 1; i < 30; i++ {
		now := at(t, javatime.UTC, 2017, 10, i, 0, 0, 0)
		period := (now.Date().DayOfYear() - 1) % 14
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", now, exprOf(et)), now.PlusDays(int64(14-period)), next(t, et, now))
		back := period
		if back == 0 {
			back = 14
		}
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", now, exprOf(et)), now.PlusDays(-int64(back)), last(t, et, now))
	}
}

func TestDayOfYearLastExecutionIssue249(t *testing.T) {
	et := forCron(t, mustParseWith(t, dayOfYearDefinition(t), biWeeklyFromFirstDayOfYear))
	for _, z := range []javatime.Zone{javatime.UTC, zone(t, "America/Argentina/Buenos_Aires")} {
		assertZoned(t, fmt.Sprintf("LastExecution(2017-10-07T00:00 %s) of %q", z, exprOf(et)), at(t, z, 2017, 9, 24, 0, 0, 0), last(t, et, at(t, z, 2017, 10, 7, 0, 0, 0)))
	}
}

func TestDayOfYearExecutionTimesEveryTwoWeeks(t *testing.T) {
	z := javatime.UTC
	var want []javatime.ZonedDateTime
	for _, md := range [][2]int{{1, 1}, {1, 15}, {1, 29}, {2, 12}, {2, 26}, {3, 12}, {3, 26}, {4, 9}, {4, 23}, {5, 7}, {5, 21}, {6, 4}} {
		want = append(want, at(t, z, 2017, md[0], md[1], 0, 0, 0))
	}
	et := forCron(t, mustParseWith(t, dayOfYearDefinition(t), biWeeklyFromFirstDayOfYear))
	for _, w := range want {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", w.PlusDays(-1), exprOf(et)), w, next(t, et, w.PlusDays(-1)))
	}
	for i := 1; i < len(want); i++ {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", want[i-1], exprOf(et)), want[i], next(t, et, want[i-1]))
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", want[i], exprOf(et)), want[i-1], last(t, et, want[i]))
	}
}

func TestDayOfYearQuartzCompatibilityIssue188(t *testing.T) {
	d := dayOfYearDefinition(t)
	quartzET := forCron(t, mustParse(t, model.Quartz, withoutDayOfYear))
	for _, e := range []string{withoutDayOfYear, withoutSpecificDayOfYear} {
		et := forCron(t, mustParseWith(t, d, e))
		start := at(t, javatime.UTC, 2017, 1, 1, 0, 0, 0).MinusSeconds(1)
		for i := 0; i < 12; i++ {
			want := next(t, quartzET, start)
			assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q (want Quartz %q)", start, e, withoutDayOfYear), want, next(t, et, start))
			start = want.PlusSeconds(1)
		}
	}
}

func TestDayOfYearIncrementsGreaterThanDaysOfMonthIssue190(t *testing.T) {
	et := forCron(t, mustParseWith(t, dayOfYearDefinition(t), "0 0 0 ? * ? * 1/56"))
	start := at(t, javatime.UTC, 2017, 1, 1, 0, 0, 0)
	for i := 0; i < 6; i++ {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", start.MinusSeconds(1), exprOf(et)), start, next(t, et, start.MinusSeconds(1)))
		start = start.PlusDays(56)
	}
}

func TestDayOfYearCustomQuarterlySchedule(t *testing.T) {
	et := forCron(t, mustParseWith(t, dayOfYearDefinition(t), "0 0 0 ? * ? * 47/91"))
	start := at(t, javatime.UTC, 2015, 12, 31, 0, 0, 0)
	for j := 0; j < 4; j++ {
		date, err := javatime.OfYearDay(2016, 47+j*91)
		if err != nil {
			t.Fatalf("javatime.OfYearDay(2016, %d): %v", 47+j*91, err)
		}
		want := javatime.OfDateTime(date, javatime.Midnight, javatime.UTC)
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", start, exprOf(et)), want, next(t, et, start))
		start = want.PlusSeconds(1)
	}
}

func TestLastExecutionWithDifferentMonthLengths(t *testing.T) {
	z := utc(t)
	cases := []struct {
		expr string
		from []javatime.ZonedDateTime
	}{
		{"0 30 23 * * ? 2022", []javatime.ZonedDateTime{at(t, z, 2023, 11, 24, 12, 0, 0)}},
		{"0 30 23 L * ? 2022", []javatime.ZonedDateTime{at(t, z, 2023, 11, 24, 12, 0, 0), at(t, z, 2023, 10, 24, 12, 0, 0)}},
		{"0 30 23 31 * ? 2022", []javatime.ZonedDateTime{at(t, z, 2023, 11, 24, 12, 0, 0),
			at(t, z, 2023, 10, 24, 12, 0, 0), at(t, z, 2023, 9, 24, 12, 0, 0)}},
	}
	want := at(t, z, 2022, 12, 31, 23, 30, 0)
	for _, c := range cases {
		et := forCron(t, mustParse(t, model.Quartz, c.expr))
		for _, from := range c.from {
			l := last(t, et, from)
			assertInstant(t, fmt.Sprintf("LastExecution(%s) of %q", from, c.expr), want, l)
			if c.expr != "0 30 23 L * ? 2022" {
				noNext(t, et, l)
			}
		}
	}
}
