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
// Ported to Go from com/cronutils/model/time/generator/ExecutionTimeUnixIntegrationTest.java.

package executiontime_test

import (
	"fmt"
	"testing"

	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

func unixET(t *testing.T, expr string) executiontimeT {
	t.Helper()
	return executiontimeT{forCron(t, mustParse(t, model.Unix, expr)), expr}
}

func TestUnixIsMatch(t *testing.T) {
	cases := map[string]string{"* * * * *": "2016-02-29T11:00:00.000-06:00", "0 * * * 1-5": "2016-03-04T11:00:00.000-06:00"}
	for expr, d := range cases {
		if !isMatch(t, unixET(t, expr).et, parseZoned(t, d)) {
			t.Errorf("IsMatch(%s) of %q = false, want true", d, expr)
		}
	}
}

func TestUnixNextExecutions(t *testing.T) {
	cases := []struct{ expr, from, want string }{
		{"*/10 * * * *", "2015-09-05T13:43:00.000-07:00", "2015-09-05T13:50:00.000-07:00"},
		{"0 3 * * 3", "2015-09-17T00:00:00.000-07:00", "2015-09-23T03:00:00.000-07:00"},
		{"* * * * 1", "2015-10-13T17:26:54.468-07:00", "2015-10-19T00:00:00.000-07:00"},
		{"* * */3 */4 */5", "2015-12-10T16:32:56.586-08:00", "2016-01-01T00:00:00.000-08:00"},
		{"* * */4 * *", "2015-12-10T16:32:56.586-08:00", "2015-12-13T00:00:00.000-08:00"},
		{"0 * * * 1-5", "2016-02-29T11:00:00.000-06:00", "2016-02-29T12:00:00.000-06:00"},
		{"0 8 * * 1", "2014-11-30T00:00:00Z", "2014-12-01T08:00:00Z"},
		{"1 0 * * tue", "2016-05-24T01:02:50Z", "2016-05-31T00:01:00Z"},
	}
	for _, c := range cases {
		assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", c.from, c.expr), parseZoned(t, c.want), next(t, unixET(t, c.expr).et, parseZoned(t, c.from)))
	}
}

func TestUnixLastExecutions(t *testing.T) {
	cases := []struct{ expr, from, want string }{
		{"0 3 * * 3", "2015-09-17T00:00:00.000-07:00", "2015-09-16T03:00:00.000-07:00"},
		{"* * * * 1", "2015-10-13T17:26:54.468-07:00", "2015-10-12T23:59:00.000-07:00"},
		{"0 11 * * 1", "2015-11-02T00:10:00Z", "2015-10-26T11:00:00Z"},
		{"* * * * 1,2", "2015-11-10T17:01:00Z", "2015-11-10T17:00:00Z"},
		{"* * * * 1,2-3", "2015-11-10T17:01:00Z", "2015-11-10T17:00:00Z"},
	}
	for _, c := range cases {
		assertZoned(t, fmt.Sprintf("LastExecution(%s) of %q", c.from, c.expr), parseZoned(t, c.want), last(t, unixET(t, c.expr).et, parseZoned(t, c.from)))
	}
}

func TestUnixEveryTwoMinutesRollsOverHour(t *testing.T) {
	e := unixET(t, "*/2 * * * *")
	n := next(t, e.et, parseZoned(t, "2015-09-05T13:56:00.000-07:00"))
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", n, e.expr), n.PlusMinutes(2), next(t, e.et, n))
}

func TestUnixNextExecutionDayOfWeekSteps(t *testing.T) {
	e := unixET(t, "0 0 * * */4")
	n := next(t, e.et, parseZoned(t, "2016-01-28T16:32:56.586-08:00"))
	assertZoned(t, fmt.Sprintf("NextExecution(2016-01-28T16:32:56.586-08:00) of %q", e.expr), parseZoned(t, "2016-01-31T00:00:00.000-08:00"), n)
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", n, e.expr), parseZoned(t, "2016-02-04T00:00:00.000-08:00"), next(t, e.et, n))
	every3 := unixET(t, "0 0 * * */3")
	assertZoned(t, fmt.Sprintf("NextExecution(2018-02-08T00:00Z) of %q", every3.expr), at(t, zone(t, "UTC"), 2018, 2, 10, 0, 0, 0),
		next(t, every3.et, at(t, zone(t, "UTC"), 2018, 2, 8, 0, 0, 0)))
}

func TestUnixDaylightSaving(t *testing.T) {
	e := unixET(t, "0 17 * * *")
	ny := zone(t, "America/New_York")
	lastRun := at(t, ny, 2016, 3, 12, 17, 0, 0)
	n := next(t, e.et, lastRun)
	if d := javatime.Between(lastRun, n); d.ToHours() != 23 || !n.Zone().Equal(ny) {
		t.Errorf("NextExecution(%s) of %q = %s, want 23 hours later in the same zone", lastRun, e.expr, n)
	}
	phoenix := zone(t, "America/Phoenix")
	now := at(t, phoenix, 2016, 3, 12, 17, 0, 0)
	l := last(t, e.et, now)
	if d := javatime.Between(l, now); d.ToHours() != 24 || !l.Zone().Equal(phoenix) {
		t.Errorf("LastExecution(%s) of %q = %s, want 24 hours earlier in the same zone", now, e.expr, l)
	}
}

func TestUnixWrongNextExecutionOnDSTEnd(t *testing.T) {
	saoPaulo := zone(t, "America/Sao_Paulo")
	date := javatime.OfInstant(1456020000, 0, saoPaulo)
	want := javatime.OfInstant(1456020060, 0, saoPaulo)
	assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", date, "* * * * *"), want, next(t, unixET(t, "* * * * *").et, date))
}

func TestUnixDSTOverlap(t *testing.T) {
	ny := zone(t, "America/New_York")
	dstEnds := map[int]int{2015: 1, 2016: 6, 2017: 5, 2018: 4, 2019: 3, 2020: 1, 2021: 7, 2022: 6, 2023: 5, 2024: 3,
		2025: 2, 2026: 1}
	edt, est := mustOffset(t, "-04:00"), mustOffset(t, "-05:00")
	e := unixET(t, "30 1 1-7 11 *")
	date := at(t, ny, 2015, 11, 1, 0, 0, 0)
	for year := 2015; year <= 2026; year++ {
		pastEnd := false
		for day := 1; day < 8; day++ {
			offset := edt
			if pastEnd {
				offset = est
			}
			if dstEnds[year] == day {
				pastEnd = true
			}
			want := javatime.FromTime(at(t, offset, year, 11, day, 1, 30, 0).Time(), ny)
			n := next(t, e.et, date)
			assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", date, e.expr), want, n)
			date = n
		}
	}
}

func TestUnixDSTGap(t *testing.T) {
	ny := zone(t, "America/New_York")
	dstStarts := map[int]int{2015: 8, 2016: 13, 2017: 12, 2018: 11, 2019: 10, 2020: 8, 2021: 14, 2022: 13, 2023: 12,
		2024: 10, 2025: 9, 2026: 8}
	e := unixET(t, "15 2 7-14 3 *")
	date := at(t, ny, 2015, 3, 7, 0, 0, 0)
	for year := 2015; year <= 2026; year++ {
		for day := 7; day < 15; day++ {
			if day == dstStarts[year] {
				continue
			}
			from := date
			date = next(t, e.et, date)
			assertZoned(t, fmt.Sprintf("NextExecution(%s) of %q", from, e.expr), at(t, ny, year, 3, day, 2, 15, 0), date)
		}
	}
}

func TestUnixTimeFromLastExecution(t *testing.T) {
	e := unixET(t, "0 12 * * *")
	vienna := zone(t, "Europe/Vienna")
	exact := at(t, vienna, 2016, 12, 2, 12, 0, 0)
	if d := timeFromLast(t, e.et, exact); d.Seconds() != 24*3600 || d.Nanos() != 0 {
		t.Errorf("TimeFromLastExecution(%s) of %q = %s, want PT24H", exact, e.expr, d)
	}
	fuzzy := exact.PlusNanos(300_000_000)
	if d := timeFromLast(t, e.et, fuzzy); d.Seconds() != 0 || d.Nanos() != 300_000_000 {
		t.Errorf("TimeFromLastExecution(%s) of %q = %s, want PT0.3S", fuzzy, e.expr, d)
	}
}

func TestUnixInvalidDayInMonth(t *testing.T) {
	next(t, unixET(t, "45 1,13 * * *").et, parseZoned(t, "2016-05-24T01:02:50Z"))
	noNext(t, unixET(t, "0 0 31 2 *").et, parseZoned(t, "2015-09-17T00:00:00.000-07:00"))
}
