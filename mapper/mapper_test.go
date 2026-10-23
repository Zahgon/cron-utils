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
// Ported to Go from com/cronutils/mapper/{ConstantsMapperTest,CronMapperIntegrationTest,CronMapperTest,
// MappingOptionalFieldsTest,WeekDayTest}.java and com/cronutils/Issue605Test.java.
//
// Not ported: CronMapperTest's null-definition cases (Go values cannot be null).

package mapper

import (
	"testing"

	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func mustParse(t *testing.T, ct model.CronType, expression string) model.Cron {
	t.Helper()
	c, err := parser.New(model.InstanceDefinitionFor(ct)).Parse(expression)
	if err != nil {
		t.Fatalf("Parse(%q) with the %s definition: %v", expression, ct, err)
	}
	return c
}

func mustMap(t *testing.T, m *CronMapper, c model.Cron) string {
	t.Helper()
	mapped, err := m.Map(c)
	if err != nil {
		t.Fatalf("Map(%q): %v; want a mapped cron", c.AsString(), err)
	}
	return mapped.AsString()
}

func TestWeekDayMappingQuartzToJava8(t *testing.T) {
	for j := 2; j < 8; j++ {
		if got := WeekDayMapping(QuartzWeekDay, Java8WeekDay, j); got != j-1 {
			t.Errorf("WeekDayMapping(quartz, java8, %d) = %d, want %d", j, got, j-1)
		}
	}
	if got := WeekDayMapping(QuartzWeekDay, Java8WeekDay, 1); got != 7 {
		t.Errorf("WeekDayMapping(quartz, java8, 1) = %d, want 7", got)
	}
}

func TestWeekDayMappingJava8ToQuartz(t *testing.T) {
	for j := 1; j < 7; j++ {
		if got := WeekDayMapping(Java8WeekDay, QuartzWeekDay, j); got != j+1 {
			t.Errorf("WeekDayMapping(java8, quartz, %d) = %d, want %d", j, got, j+1)
		}
	}
	if got := WeekDayMapping(Java8WeekDay, QuartzWeekDay, 7); got != 1 {
		t.Errorf("WeekDayMapping(java8, quartz, 7) = %d, want 1", got)
	}
}

func TestWeekDayMappingQuartzToCrontab(t *testing.T) {
	for j := 1; j < 7; j++ {
		if got := WeekDayMapping(QuartzWeekDay, CrontabWeekDay, j); got != j-1 {
			t.Errorf("WeekDayMapping(quartz, crontab, %d) = %d, want %d", j, got, j-1)
		}
	}
}

func TestWeekDayMappingCrontabToQuartz(t *testing.T) {
	for j := 0; j < 7; j++ {
		if got := WeekDayMapping(CrontabWeekDay, QuartzWeekDay, j); got != j+1 {
			t.Errorf("WeekDayMapping(crontab, quartz, %d) = %d, want %d", j, got, j+1)
		}
	}
}

func TestWeekDayMappingCrontabAndJava8(t *testing.T) {
	for j := 1; j < 7; j++ {
		if got := WeekDayMapping(CrontabWeekDay, Java8WeekDay, j); got != j {
			t.Errorf("WeekDayMapping(crontab, java8, %d) = %d, want %d", j, got, j)
		}
		if got := WeekDayMapping(Java8WeekDay, CrontabWeekDay, j); got != j {
			t.Errorf("WeekDayMapping(java8, crontab, %d) = %d, want %d", j, got, j)
		}
	}
	if got := WeekDayMapping(CrontabWeekDay, Java8WeekDay, 0); got != 7 {
		t.Errorf("WeekDayMapping(crontab, java8, 0) = %d, want 7", got)
	}
	if got := WeekDayMapping(Java8WeekDay, CrontabWeekDay, 7); got != 0 {
		t.Errorf("WeekDayMapping(java8, crontab, 7) = %d, want 0", got)
	}
}

func TestWeekDay(t *testing.T) {
	if _, err := model.NewWeekDay(-1, false); err == nil {
		t.Errorf("NewWeekDay(-1, false): no error, want an error")
	}
	source, err := model.NewWeekDay(1, false)
	if err != nil || source.MondayDoWValue() != 1 {
		t.Fatalf("NewWeekDay(1, false) = %v, %v; want a week day with Monday = 1", source, err)
	}
	cases := []struct {
		monday     int
		zero       bool
		value, out int
	}{
		{1, true, 7, 0},
		{0, true, 1, 0},
		{1, false, 7, 7},
		{2, false, 7, 1},
	}
	for _, c := range cases {
		target, err := model.NewWeekDay(c.monday, c.zero)
		if err != nil {
			t.Fatalf("NewWeekDay(%d, %v): %v", c.monday, c.zero, err)
		}
		if got := source.MapTo(c.value, target); got != c.out {
			t.Errorf("WeekDay(1, false).MapTo(%d, WeekDay(%d, %v)) = %d, want %d", c.value, c.monday, c.zero, got, c.out)
		}
	}
}

func TestCronMapperIntegration(t *testing.T) {
	cases := []struct {
		m    *CronMapper
		from model.CronType
		expr string
		want string
	}{
		{FromCron4jToQuartz(), model.Cron4j, "30 8 10 6 *", "0 30 8 10 6 ? *"},
		{FromCron4jToQuartz(), model.Cron4j, "0 11,16 * * *", "0 0 11,16 * * ? *"},
		{FromCron4jToQuartz(), model.Cron4j, "0 9-18 * * 1-3", "0 0 9-18 ? * 2-4 *"},
		{FromQuartzToCron4j(), model.Quartz, "5 30 8 10 6 ? 1984", "30 8 10 6 *"},
		{FromQuartzToCron4j(), model.Quartz, "5 0 11,16 * * ? 1984", "0 11,16 * * *"},
		{FromQuartzToCron4j(), model.Quartz, "5 0 9-18 ? * 1-3 1984", "0 9-18 * * 0-2"},
		{FromQuartzToSpring(), model.Quartz, "5 0 9-18 ? * 1-3 1984", "5 0 9-18 ? * 0-2"},
		{FromUnixToQuartz(), model.Unix, "* * * * 3,5-6,*/2,2/3,7/4", "0 * * ? * 4,6-7,*/2,3/3,1/4 *"},
		{FromUnixToQuartz(), model.Unix, "0 0 * * 1", "0 0 0 ? * 2 *"},
	}
	for _, c := range cases {
		if got := mustMap(t, c.m, mustParse(t, c.from, c.expr)); got != c.want {
			t.Errorf("Map(%s %q) = %q, want %q", c.from, c.expr, got, c.want)
		}
	}
}

func TestCronMapperEveryMinuteUnixToQuartz(t *testing.T) {
	got := mustMap(t, FromUnixToQuartz(), mustParse(t, model.Unix, "* * * * *"))
	if got != "0 * * * * ? *" && got != "0 * * ? * * *" {
		t.Errorf("Map(Unix %q) to Quartz = %q, want %q or %q", "* * * * *", got, "0 * * * * ? *", "0 * * ? * * *")
	}
}

func TestCronMapperFieldFunctions(t *testing.T) {
	f := model.NewCronField(model.Second, model.NewOn(5), model.NewFieldConstraintsBuilder().Build())
	zero, err := onZero(model.Second)(f)
	if on, ok := zero.Expression.(*model.On); err != nil || zero.Field != model.Second || !ok || on.Time != 0 {
		t.Errorf("onZero(SECOND) of field SECOND %q = %+v, %v; want SECOND with the value 0", f.Expression.AsString(), zero, err)
	}
	always, err := alwaysField(model.Second)(f)
	if _, ok := always.Expression.(*model.Always); err != nil || always.Field != model.Second || !ok {
		t.Errorf("alwaysField(SECOND) of field SECOND %q = %+v, %v; want SECOND *", f.Expression.AsString(), always, err)
	}
	same, err := SameCron(model.NewSingleCron(model.InstanceDefinitionFor(model.Quartz), []*model.CronField{f}))
	if err != nil || same == nil {
		t.Errorf("SameCron(Quartz cron with field SECOND %q) = %v, %v; want the cron", f.Expression.AsString(), same, err)
	}
}

func TestMappingOptionalFields(t *testing.T) {
	target, err := model.DefineCron().
		WithMinutes().WithStrictRange().WithValidRange(0, 60).And().
		WithHours().WithStrictRange().And().
		WithDayOfMonth().SupportsL().WithStrictRange().And().
		WithMonth().WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).SupportsHash().SupportsL().
		WithStrictRange().And().
		WithYear().Optional().And().
		Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a Unix-like definition with an optional year: %v", err)
	}
	m := New(model.InstanceDefinitionFor(model.Quartz), target, SameCron)
	if got := mustMap(t, m, mustParse(t, model.Quartz, "5 0 9-18 ? * 1-3")); got != "0 9-18 * * 0-2" {
		t.Errorf("Map(Quartz %q) to a Unix-like definition with an optional year = %q, want %q", "5 0 9-18 ? * 1-3", got, "0 9-18 * * 0-2")
	}
}

func TestDayOfWeekMappingIssue605(t *testing.T) {
	cases := []struct {
		from model.CronType
		m    *CronMapper
		want string
	}{
		{model.Quartz, FromQuartzToCron4j(), "0 0 * * 4#1"},
		{model.Quartz, FromQuartzToSpring(), "0 0 0 ? * 4#1"},
		{model.Quartz, FromQuartzToUnix(), "0 0 * * 4#1"},
		{model.Spring, FromSpringToQuartz(), "0 0 0 ? * 6#1 *"},
	}
	for _, c := range cases {
		if got := mustMap(t, c.m, mustParse(t, c.from, "0 0 0 ? * 5#1")); got != c.want {
			t.Errorf("Map(\"0 0 0 ? * 5#1\") from %s = %q, want %q", c.from, got, c.want)
		}
	}
}
