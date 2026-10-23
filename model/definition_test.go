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
// Ported to Go from com/cronutils/model/definition/{CronDefinitionBuilderTest,CronDefinitionTest,FieldDefinitionTest,
// CronDefinitionIssue25IntegrationTest,Issue343Test}.java, com/cronutils/model/field/CronFieldTest.java,
// com/cronutils/model/field/constraints/FieldConstraintsTest.java and
// com/cronutils/builder/CronBuilderQuartzIntegrationTest.java and com/cronutils/Issue459Test.java.
//
// Not ported: the null-argument cases (Go values cannot be null), FieldConstraintsTest (it only checks null
// arguments), the comparator-object cases (definitions and fields are ordered by their field name's Order) and
// FieldDefinitionBuilderTest (disabled in the original).

package model_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jmrozanec/cron-utils/builder"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func mustDefinition(t *testing.T, b *model.CronDefinitionBuilder) *model.CronDefinition {
	t.Helper()
	d, err := b.Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a custom definition: %v; want a definition", err)
	}
	return d
}

func mustParse(t *testing.T, d *model.CronDefinition, expression string) model.Cron {
	t.Helper()
	c, err := parser.New(d).Parse(expression)
	if err != nil {
		t.Fatalf("Parse(%q) with definition %s: %v", expression, fieldNames(d), err)
	}
	return c
}

// fieldNames describes a definition's fields with their ranges, for failure messages.
func fieldNames(d *model.CronDefinition) string {
	var s []string
	for _, f := range d.FieldDefinitions() {
		s = append(s, fmt.Sprintf("%s %d-%d", f.FieldName, f.Constraints.StartRange(), f.Constraints.EndRange()))
	}
	return "[" + strings.Join(s, ", ") + "]"
}

// definitionArgs describes NewCronDefinition arguments, for failure messages.
func definitionArgs(fields []*model.FieldDefinition, constraints []*model.CronConstraint, nicknames map[model.CronNickname]bool,
	matchDoWAndDoM bool) string {
	var f, c []string
	for _, fd := range fields {
		f = append(f, fmt.Sprintf("%s %d-%d", fd.FieldName, fd.Constraints.StartRange(), fd.Constraints.EndRange()))
	}
	for _, cc := range constraints {
		c = append(c, cc.Description)
	}
	return fmt.Sprintf("fields %v, constraints %q, %d nicknames, matchDayOfWeekAndDayOfMonth %v", f, c, len(nicknames), matchDoWAndDoM)
}

func TestCronDefinitionBuilderSingleField(t *testing.T) {
	cases := []struct {
		add  func(*model.CronDefinitionBuilder) *model.FieldDefinitionBuilder
		want model.CronFieldName
	}{
		{(*model.CronDefinitionBuilder).WithSeconds, model.Second},
		{(*model.CronDefinitionBuilder).WithMinutes, model.Minute},
		{(*model.CronDefinitionBuilder).WithHours, model.Hour},
		{(*model.CronDefinitionBuilder).WithDayOfMonth, model.DayOfMonth},
		{(*model.CronDefinitionBuilder).WithMonth, model.Month},
		{(*model.CronDefinitionBuilder).WithDayOfWeek, model.DayOfWeek},
		{(*model.CronDefinitionBuilder).WithYear, model.Year},
	}
	for _, c := range cases {
		d := mustDefinition(t, c.add(model.DefineCron()).And())
		fields := d.FieldDefinitions()
		if len(fields) != 1 || fields[0].FieldName != c.want {
			t.Errorf("definition built with only %s has fields %s, want [%s]", c.want, fieldNames(d), c.want)
		}
	}
}

func TestCronDefinitionBuilderLastFieldOptionalFalseByDefault(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().WithHours().And())
	if d.FieldDefinitions()[0].Optional {
		t.Errorf("definition built with only HOUR: field %s is optional, want mandatory by default", d.FieldDefinitions()[0].FieldName)
	}
}

func TestCronDefinitionBuilderRegister(t *testing.T) {
	b := model.DefineCron()
	fd := &model.FieldDefinition{FieldName: model.Second,
		Constraints: model.NewFieldConstraintsBuilder().WithValidRange(0, 1).WithStrictRange().Build()}
	if err := b.Register(fd); err != nil {
		t.Fatalf("Register(SECOND 0-1 strict): %v", err)
	}
	d := mustDefinition(t, b)
	if fields := d.FieldDefinitions(); len(fields) != 1 || fields[0] != fd {
		t.Errorf("FieldDefinitions() after Register(SECOND 0-1 strict) = %s, want [SECOND 0-1] (the registered definition)", fieldNames(d))
	}
}

func TestCronDefinitionBuilderPredefinedDefinitions(t *testing.T) {
	for _, ct := range []model.CronType{model.Unix, model.Quartz, model.Cron4j, model.Spring, model.Spring53} {
		if model.InstanceDefinitionFor(ct) == nil {
			t.Errorf("InstanceDefinitionFor(%s) = nil, want a definition", ct)
		}
	}
}

func TestCronDefinitionShouldNotAcceptQuestionMark(t *testing.T) {
	if _, err := parser.New(model.InstanceDefinitionFor(model.Unix)).Parse("* * * * ?"); err == nil {
		t.Errorf("Parse(%q) with the Unix definition: no error, want an error", "* * * * ?")
	}
}

func TestCronDefinitionShouldNotAcceptMultipleOptionalFields(t *testing.T) {
	_, err := model.DefineCron().
		WithMinutes().And().
		WithHours().And().
		WithDayOfMonth().Optional().And().
		WithMonth().Optional().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).And().
		Instance()
	if err == nil {
		t.Errorf("Instance() of MINUTE, HOUR, optional DAY_OF_MONTH, optional MONTH, DAY_OF_WEEK: no error, want an error (mandatory field after optional ones)")
	}
}

func mustBuild(t *testing.T, b *builder.CronBuilder) model.Cron {
	t.Helper()
	c, err := b.Instance()
	if err != nil {
		t.Fatalf("CronBuilder.Instance() for a field-by-field cron: %v; want a cron", err)
	}
	return c
}

func TestCronDefinitionBuilderWeekdayOffset(t *testing.T) {
	d := model.InstanceDefinitionFor(model.Quartz)
	for day, want := range map[model.Weekday]string{model.Friday: "0 0 12 ? * 6 *", model.Sunday: "0 0 12 ? * 1 *"} {
		cron := mustBuild(t, builder.Cron(d).
			WithYear(model.AlwaysExpression()).
			WithMonth(model.AlwaysExpression()).
			WithDoW(model.NewOn(day.NumberFor(d))).
			WithDoM(model.QuestionMarkExpression()).
			WithHour(model.NewOn(12)).
			WithMinute(model.NewOn(0)).
			WithSecond(model.NewOn(0)))
		if got := cron.AsString(); got != want {
			t.Errorf("Quartz cron built for weekday %d (day-of-week %d) = %q, want %q", day, day.NumberFor(d), got, want)
		}
	}
}

func TestCronDefinitionBuilderSpringSchedule(t *testing.T) {
	cron := mustBuild(t, builder.Cron(model.InstanceDefinitionFor(model.Spring)).
		WithDoW(model.QuestionMarkExpression()).
		WithMonth(model.AlwaysExpression()).
		WithDoM(model.AlwaysExpression()).
		WithHour(model.AlwaysExpression()).
		WithMinute(model.NewEveryOf(model.NewOn(0), 5)).
		WithSecond(model.AlwaysExpression()))
	if got := cron.AsString(); got != "* 0/5 * * * ?" {
		t.Errorf("AsString() of a Spring cron built from fields * 0/5 * * * ? = %q, want %q", got, "* 0/5 * * * ?")
	}
}

func spring53At0(dom, dow model.FieldExpression) *builder.CronBuilder {
	return builder.Cron(model.InstanceDefinitionFor(model.Spring53)).
		WithDoW(dow).
		WithMonth(model.AlwaysExpression()).
		WithDoM(dom).
		WithHour(model.NewOn(0)).
		WithMinute(model.NewOn(0)).
		WithSecond(model.NewOn(0))
}

func TestCronDefinitionBuilderSpring53Schedules(t *testing.T) {
	always := model.AlwaysExpression()
	cases := []struct {
		dom, dow model.FieldExpression
		want     string
	}{
		{model.NewOnSpecial(model.DefaultNth, model.L, model.DefaultNth), always, "0 0 0 L * *"},
		{model.NewBetween(model.SpecialCharFieldValue(model.L), model.IntegerFieldValue(3)), always, "0 0 0 L-3 * *"},
		{always, model.NewOnSpecial(5, model.L, model.DefaultNth), "0 0 0 * * 5L"},
		{model.NewOnSpecial(1, model.W, model.DefaultNth), always, "0 0 0 1W * *"},
		{model.NewOnSpecial(model.DefaultNth, model.LW, model.DefaultNth), always, "0 0 0 LW * *"},
		{model.NewOnSpecial(model.DefaultNth, model.QuestionMarkChar, model.DefaultNth), model.NewOnSpecial(5, model.Hash, 2),
			"0 0 0 ? * 5#2"},
	}
	for _, c := range cases {
		if got := mustBuild(t, spring53At0(c.dom, c.dow)).AsString(); got != c.want {
			t.Errorf("AsString() of a Spring 5.3 cron built from day of month %q and day of week %q = %q, want %q", c.dom.AsString(), c.dow.AsString(), got, c.want)
		}
	}
}

func TestCronDefinitionBuilderSpring53BuiltMatchesParsed(t *testing.T) {
	d := model.InstanceDefinitionFor(model.Spring53)
	cases := []struct {
		dom, dow model.FieldExpression
		parsed   string
	}{
		{model.AlwaysExpression(), model.NewOnSpecial(4, model.L, model.DefaultNth), "0 0 0 * * THUL"},
		{model.NewOnSpecial(model.DefaultNth, model.QuestionMarkChar, model.DefaultNth), model.NewOnSpecial(1, model.Hash, 1),
			"0 0 0 ? * MON#1"},
	}
	for _, c := range cases {
		built := mustBuild(t, spring53At0(c.dom, c.dow)).AsString()
		if parsed := mustParse(t, d, c.parsed).AsString(); built != parsed {
			t.Errorf("Spring 5.3 cron built from day of month %q and day of week %q = %q, want %q (Parse(%q).AsString())", c.dom.AsString(), c.dow.AsString(), built, parsed, c.parsed)
		}
	}
}

func TestCronDefinitionBuilderSpring53Nicknames(t *testing.T) {
	d := model.InstanceDefinitionFor(model.Spring53)
	for _, n := range []string{"@yearly", "@annually", "@monthly", "@weekly", "@daily", "@midnight", "@hourly"} {
		mustParse(t, d, n)
	}
	mustParse(t, mustDefinition(t, model.DefineCron().WithSupportedNickname(model.Reboot)), "@reboot")
}

func TestCronDefinitionIssue25(t *testing.T) {
	d := mustDefinition(t, model.DefineCron().WithMinutes().And().WithHours().And().WithDayOfWeek().And())
	cron := mustParse(t, d, "0 18 1")
	for name, want := range map[model.CronFieldName]string{model.Minute: "0", model.Hour: "18", model.DayOfWeek: "1"} {
		f, err := cron.Retrieve(name)
		if err != nil || f == nil || f.Expression.AsString() != want {
			t.Errorf("Retrieve(%s) of %q = %v, %v; want %q", name, "0 18 1", f, err, want)
		}
	}
	if _, err := executiontime.ForCron(cron); err != nil {
		t.Errorf("executiontime.ForCron(%q): %v; want no error", cron.AsString(), err)
	}
}

func TestCronDefinitionConstructorRejectsEmptyFields(t *testing.T) {
	if _, err := model.NewCronDefinition(nil, nil, map[model.CronNickname]bool{}, false); err == nil {
		t.Errorf("NewCronDefinition(nil fields): no error, want an error")
	}
}

func field(name model.CronFieldName, optional bool) *model.FieldDefinition {
	return &model.FieldDefinition{FieldName: name, Constraints: model.NewFieldConstraintsBuilder().ForField(name).Build(),
		Optional: optional}
}

func TestCronDefinitionLastFieldOptional(t *testing.T) {
	fields := []*model.FieldDefinition{field(model.Second, false), field(model.Minute, false), field(model.Hour, true)}
	d, err := model.NewCronDefinition(fields, nil, map[model.CronNickname]bool{}, false)
	if err != nil {
		t.Fatalf("NewCronDefinition(SECOND, MINUTE, optional HOUR): %v", err)
	}
	got := d.FieldDefinitions()
	if !got[len(got)-1].Optional {
		t.Errorf("NewCronDefinition(SECOND, MINUTE, optional HOUR): last field %s is mandatory, want optional", got[len(got)-1].FieldName)
	}
	if _, err := model.NewCronDefinition([]*model.FieldDefinition{field(model.Hour, true)}, nil, nil, false); err == nil {
		t.Errorf("NewCronDefinition(optional HOUR alone): no error, want an error")
	}
}

func TestCronDefinitionGetFieldDefinitions(t *testing.T) {
	f := field(model.Second, false)
	d, err := model.NewCronDefinition([]*model.FieldDefinition{f}, nil, nil, false)
	if err != nil {
		t.Fatalf("NewCronDefinition(SECOND): %v", err)
	}
	if got := d.FieldDefinitions(); len(got) != 1 || got[0] != f {
		t.Errorf("FieldDefinitions() of NewCronDefinition(SECOND) = %s, want [SECOND 0-59] (the one definition)", fieldNames(d))
	}
}

func TestCronDefinitionEquality(t *testing.T) {
	if !model.InstanceDefinitionFor(model.Quartz).Equal(model.InstanceDefinitionFor(model.Quartz)) {
		t.Errorf("InstanceDefinitionFor(Quartz).Equal(InstanceDefinitionFor(Quartz)) = false, want true")
	}
	seconds := func(start, end int) []*model.FieldDefinition {
		return []*model.FieldDefinition{{FieldName: model.Second,
			Constraints: model.NewFieldConstraintsBuilder().ForField(model.Second).WithValidRange(start, end).Build()}}
	}
	simple := func() []*model.FieldDefinition { return seconds(0, 59) }
	one := func() []*model.CronConstraint {
		return []*model.CronConstraint{model.EnsureEitherDayOfWeekOrDayOfMonth()}
	}
	two := func() []*model.CronConstraint { return append(one(), model.EnsureEitherDayOfYearOrMonth()) }
	oneNick := func() map[model.CronNickname]bool { return map[model.CronNickname]bool{model.Annually: true} }
	twoNicks := func() map[model.CronNickname]bool {
		return map[model.CronNickname]bool{model.Annually: true, model.Daily: true}
	}
	cases := []struct {
		f1, f2     []*model.FieldDefinition
		c1, c2     []*model.CronConstraint
		n1, n2     map[model.CronNickname]bool
		m1, m2     bool
		shouldBeEq bool
	}{
		{simple(), simple(), two(), two(), twoNicks(), twoNicks(), true, true, true},
		{seconds(0, 59), seconds(1, 60), two(), two(), twoNicks(), twoNicks(), true, true, false},
		{simple(), simple(), one(), two(), twoNicks(), twoNicks(), true, true, false},
		{simple(), simple(), two(), two(), oneNick(), twoNicks(), true, true, false},
		{simple(), simple(), two(), two(), twoNicks(), twoNicks(), false, true, false},
		{simple(), simple(), two(), one(), twoNicks(), oneNick(), false, true, false},
	}
	for i, c := range cases {
		a1, a2 := definitionArgs(c.f1, c.c1, c.n1, c.m1), definitionArgs(c.f2, c.c2, c.n2, c.m2)
		d1, err := model.NewCronDefinition(c.f1, c.c1, c.n1, c.m1)
		if err != nil {
			t.Fatalf("case %d: NewCronDefinition(%s): %v", i, a1, err)
		}
		d2, err := model.NewCronDefinition(c.f2, c.c2, c.n2, c.m2)
		if err != nil {
			t.Fatalf("case %d: NewCronDefinition(%s): %v", i, a2, err)
		}
		if got := d1.Equal(d2); got != c.shouldBeEq {
			t.Errorf("case %d: NewCronDefinition(%s).Equal(NewCronDefinition(%s)) = %v, want %v", i, a1, a2, got, c.shouldBeEq)
		}
	}
}

func TestFieldDefinitionValues(t *testing.T) {
	constraints := model.NewFieldConstraintsBuilder().Build()
	fd := &model.FieldDefinition{FieldName: model.Second, Constraints: constraints}
	if fd.FieldName != model.Second || fd.Constraints != constraints {
		t.Errorf("FieldDefinition{FieldName: SECOND, Constraints: %p} = FieldName %s, Constraints %p; want FieldName %s, Constraints %p",
			constraints, fd.FieldName, fd.Constraints, model.Second, constraints)
	}
	if model.DayOfMonth.Order() <= model.Second.Order() {
		t.Errorf("DAY_OF_MONTH.Order() = %d, SECOND.Order() = %d; want DAY_OF_MONTH after SECOND", model.DayOfMonth.Order(), model.Second.Order())
	}
}

func TestSpringExpressionsIssue343(t *testing.T) {
	d := model.InstanceDefinitionFor(model.Spring)
	for _, e := range []string{"0 0 * * * *", "*/10 * * * * *", "0 0 8-10 * * *", "0 0 6,19 * * *", "0 0/30 8-10 * * *",
		"0 0 9-17 * * MON-FRI", "0 0 0 25 12 ?"} {
		mustParse(t, d, e)
	}
}

func TestCronFieldValues(t *testing.T) {
	e := model.NewOn(3)
	f := model.NewCronField(model.Second, e, model.NewFieldConstraintsBuilder().Build())
	if f.Field != model.Second || f.Expression != model.FieldExpression(e) {
		t.Errorf("NewCronField(SECOND, On(3)) = field %s, expression %q; want field %s, expression %q", f.Field, f.Expression.AsString(), model.Second, e.AsString())
	}
}

func quartzBuilt(t *testing.T, dom, dow model.FieldExpression) model.Cron {
	return mustBuild(t, builder.Cron(model.InstanceDefinitionFor(model.Quartz)).
		WithSecond(model.NewOn(0)).
		WithMinute(model.NewOn(1)).
		WithHour(model.NewOn(13)).
		WithDoM(dom).
		WithMonth(model.AlwaysExpression()).
		WithDoW(dow).
		WithYear(model.AlwaysExpression()))
}

func TestCronBuilderQuartzDayFields(t *testing.T) {
	q := model.QuestionMarkExpression()
	cases := []struct {
		dom, dow model.FieldExpression
		want     string
	}{
		{model.AlwaysExpression(), q, "0 1 13 * * ? *"},
		{model.NewOn(1), q, "0 1 13 1 * ? *"},
		{q, model.NewOn(1), "0 1 13 ? * 1 *"},
	}
	for _, c := range cases {
		cron := quartzBuilt(t, c.dom, c.dow)
		if err := cron.Validate(); err != nil {
			t.Errorf("Validate() of %q: %v; want no error", cron.AsString(), err)
		}
		if got := cron.AsString(); got != c.want {
			t.Errorf("AsString() of a Quartz cron built from day of month %q and day of week %q = %q, want %q", c.dom.AsString(), c.dow.AsString(), got, c.want)
		}
	}
}

func TestCronBuilderRejectsNegativeValuesIssue459(t *testing.T) {
	if _, err := model.OnTime(-1); err == nil {
		t.Errorf("OnTime(-1): no error, want an error")
	}
	on, err := model.OnTime(5)
	if err != nil || on.Time != 5 {
		t.Errorf("OnTime(5) = %v, %v; want an On with time 5", on, err)
	}
}
