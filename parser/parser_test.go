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
// Ported to Go from com/cronutils/parser/{CronParserCron4JIntegrationTest,CronParserNicknameTest,
// CronParserQuartzIntegrationTest,CronParserSpringIntegrationTest,CronParserTest}.java,
// com/cronutils/model/field/FieldParserTest.java and com/cronutils/{Issue413Test,Issue499Test}.java.
//
// CronParserTest built its parsers on mocked definitions; here they are built on real definitions with the same
// fields. CronParserSpringIntegrationTest also parsed with Spring's own CronExpression; only the cron-utils parse is
// kept. Not ported: CronParserFieldTest and FieldParserRandomTest (disabled in the original) and the null-argument
// cases.

package parser_test

import (
	"strings"
	"testing"

	"github.com/jmrozanec/cron-utils/builder"
	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func newParser(ct model.CronType) *parser.CronParser {
	return parser.New(model.InstanceDefinitionFor(ct))
}

func mustParse(t *testing.T, p *parser.CronParser, expression string) model.Cron {
	t.Helper()
	c, err := p.Parse(expression)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expression, err)
	}
	return c
}

func expectParseError(t *testing.T, p *parser.CronParser, expression string) error {
	t.Helper()
	_, err := p.Parse(expression)
	if !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("Parse(%q) error = %v, want an invalid-argument error", expression, err)
	}
	return err
}

func TestCron4jParser(t *testing.T) {
	p := newParser(model.Cron4j)
	for _, e := range []string{"* 1,2,3,4,5,6 * 1,2,3 *", "* 1 1,2 * 4", "* 1 1-2 * 4", "* * * * sun"} {
		mustParse(t, p, e)
	}
	expectParseError(t, p, "* 1 5-2 * 4")
	if got := mustParse(t, p, "* * L * *").AsString(); got != "* * L * *" {
		t.Errorf("Parse(Cron4j %q).AsString() = %q, want %q", "* * L * *", got, "* * L * *")
	}
}

func nicknameDefinition(t *testing.T) *model.CronDefinition {
	b := model.DefineCron().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).WithStrictRange().And().
		WithMonth().WithValidRange(1, 12).WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).WithStrictRange().And()
	for _, n := range []model.CronNickname{model.Yearly, model.Annually, model.Monthly, model.Weekly, model.Midnight,
		model.Daily, model.Hourly} {
		b.WithSupportedNickname(n)
	}
	d, err := b.Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a Unix-like definition with nicknames: %v", err)
	}
	return d
}

func TestNicknames(t *testing.T) {
	d := nicknameDefinition(t)
	p := parser.New(d)
	on := func(v int) model.FieldExpression { return model.NewOn(v) }
	always := model.AlwaysExpression()
	cases := []struct {
		nickname                      string
		minute, hour, dom, month, dow model.FieldExpression
	}{
		{"@yearly", on(0), on(0), on(1), on(1), always},
		{"@annually", on(0), on(0), on(1), on(1), always},
		{"@monthly", on(0), on(0), on(1), always, always},
		{"@weekly", on(0), on(0), always, always, on(0)},
		{"@midnight", on(0), on(0), always, always, always},
		{"@daily", on(0), on(0), always, always, always},
		{"@hourly", on(0), always, always, always, always},
	}
	for _, c := range cases {
		expected, err := builder.Cron(d).WithMinute(c.minute).WithHour(c.hour).WithDoM(c.dom).WithMonth(c.month).
			WithDoW(c.dow).Instance()
		if err != nil {
			t.Fatalf("CronBuilder.Instance() for the expansion of %s: %v", c.nickname, err)
		}
		if parsed := mustParse(t, p, c.nickname); !parsed.Equivalent(expected) {
			t.Errorf("Parse(%q) = %q, want the equivalent of %q", c.nickname, parsed.AsString(), expected.AsString())
		}
	}
}

func TestQuartzParserInvalidChars(t *testing.T) {
	p := newParser(model.Quartz)
	for _, e := range []string{"* * * * $ ?", "* * * * $W ?", "* * * * $#3 ?", "* * * * 3#$ ?", "* * * W-3 * ?",
		"0 30 17 4 1 * 2016"} {
		expectParseError(t, p, e)
	}
}

func TestQuartzParserAcceptedExpressions(t *testing.T) {
	p := newParser(model.Quartz)
	for _, e := range []string{"* * * L-3 * ?", "* * * 3L * ?", "0 0/10 22 L * ?", "0 0 0 * JUL-AUG ? *",
		"0 0 0 * JAN-FEB ? *", "0 0 0 * JAN ? *", "0 0 0 ? * MON-FRI *", "0 0 0 ? * MON *", "0 0 0 * JUL ? *",
		"0 0 12 ? * SUN-SAT", "0 0 0 ? * MON,TUE *", "0 0 1 ? 1/1 MONL *"} {
		mustParse(t, p, e)
	}
}

func TestQuartzParserIntervals(t *testing.T) {
	p := newParser(model.Quartz)
	cases := []struct{ expr, from, want string }{
		{"0/20 * * * * ?", "2005-08-09T18:32:42", "2005-08-09T18:32:40Z"},
		{"0 0/7 * * * ?", "2005-08-09T18:32:42", "2005-08-09T18:28Z"},
	}
	for _, c := range cases {
		et, err := executiontime.ForCron(mustParse(t, p, c.expr))
		if err != nil {
			t.Fatalf("executiontime.ForCron(%q): %v", c.expr, err)
		}
		from, err := javatime.ParseZoned(c.from, javatime.UTC)
		if err != nil {
			t.Fatalf("javatime.ParseZoned(%q, UTC): %v", c.from, err)
		}
		got, ok, err := et.LastExecution(from)
		if err != nil || !ok || got.String() != c.want {
			t.Errorf("LastExecution(%s) of %q = %s, %v, %v; want %s", from, c.expr, got, ok, err, c.want)
		}
	}
	et, err := executiontime.ForCron(mustParse(t, p, "0 15 10 L-7 * ?"))
	if err != nil {
		t.Fatalf("executiontime.ForCron(%q): %v", "0 15 10 L-7 * ?", err)
	}
	from, err := javatime.ParseZoned("2017-01-31T10:00", javatime.UTC)
	if err != nil {
		t.Fatalf("javatime.ParseZoned(%q, UTC): %v", "2017-01-31T10:00", err)
	}
	if got, ok, err := et.NextExecution(from); err != nil || !ok || got.String() != "2017-02-21T10:15Z" {
		t.Errorf("NextExecution(%s) of %q = %s, %v, %v; want 2017-02-21T10:15Z", from, "0 15 10 L-7 * ?", got, ok, err)
	}
}

func TestQuartzParserErrorMessages(t *testing.T) {
	p := newParser(model.Quartz)
	cases := []struct {
		expr string
		want string
	}{
		{"* * * * $ ?", "Invalid chars in expression! Expression: $ Invalid chars: $"},
		{"0/1 * * * * *", "Invalid cron expression: 0/1 * * * * *. Invalid cron expression: Both, a day-of-week AND a day-of-month parameter, must have at least one with a '?' character."},
		{"* * -1 * * ?", "Invalid expression! Expression: -1 does not describe a range. Negative numbers are not allowed."},
		{"*/ * * * * ?", "Missing steps for expression: */"},
	}
	for _, c := range cases {
		err := expectParseError(t, p, c.expr)
		if err != nil && !strings.Contains(err.Error(), c.want) {
			t.Errorf("Parse(%q) error = %q, want it to contain %q", c.expr, err, c.want)
		}
	}
	if _, err := p.Parse("* *"); err == nil || err.Error() != "Cron expression contains 2 parts but we expect one of [6, 7]" {
		t.Errorf("Parse(Quartz %q) error = %v, want %q", "* *", err, "Cron expression contains 2 parts but we expect one of [6, 7]")
	}
	if err := expectParseError(t, p, "* * -1 * * ?"); err != nil &&
		!strings.HasSuffix(err.Error(), "Invalid expression! Expression: -1 does not describe a range. Negative numbers are not allowed.") {
		t.Errorf("Parse(Quartz %q) error = %q, want it to end with %q", "* * -1 * * ?", err, "Invalid expression! Expression: -1 does not describe a range. Negative numbers are not allowed.")
	}
}

func TestQuartzParserYearIncrements(t *testing.T) {
	p := newParser(model.Quartz)
	for _, e := range []string{"0 0 0 1 * ? 2017/2", "0 0 0 1 * ? 2017/3", "0 0 0 1 * ? 2017/10", "0 0 0 1 * ? 2017-2047/2"} {
		if err := mustParse(t, p, e).Validate(); err != nil {
			t.Errorf("Validate() of Quartz %q: %v; want no error", e, err)
		}
	}
}

func TestQuartzBuilderYears(t *testing.T) {
	d := model.InstanceDefinitionFor(model.Quartz)
	_, err := builder.Cron(d).
		WithDoM(model.NewOn(1)).
		WithDoW(model.QuestionMarkExpression()).
		WithYear(model.NewEveryOf(model.NewBetween(model.IntegerFieldValue(1970), model.IntegerFieldValue(2099)), 4)).
		WithMonth(model.NewOn(1)).
		WithHour(model.NewOn(0)).
		WithMinute(model.NewOn(0)).
		WithSecond(model.NewOn(0)).
		Instance()
	if err != nil {
		t.Errorf("CronBuilder.Instance() for Quartz 0 0 0 1 1 ? 1970-2099/4: %v; want no error", err)
	}
	if _, err := builder.Cron(d).WithMonth(model.NewOn(0)).Instance(); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("CronBuilder.Instance() for Quartz with month 0: error %v, want an invalid-argument error", err)
	}
}

func TestQuartzDescribeAfterParse(t *testing.T) {
	p := newParser(model.Quartz)
	mustParse(t, p, "0 0 0 ? * MON,TUE *")
	mustParse(t, p, "0 * * ? * 1,5")
}

func TestSpringNthDayOfWeek(t *testing.T) {
	mustParse(t, newParser(model.Spring), "0 0 0 ? * FRI#1")
}

func singleFieldDefinition(t *testing.T, names ...model.CronFieldName) *model.CronDefinition {
	var fields []*model.FieldDefinition
	for _, n := range names {
		fields = append(fields, &model.FieldDefinition{FieldName: n, Constraints: model.NewFieldConstraintsBuilder().Build()})
	}
	d, err := model.NewCronDefinition(fields, nil, nil, false)
	if err != nil {
		t.Fatalf("NewCronDefinition(%v): %v", names, err)
	}
	return d
}

func TestCronParserGeneral(t *testing.T) {
	seconds := parser.New(singleFieldDefinition(t, model.Second))
	expectParseError(t, seconds, "")
	expectParseError(t, seconds, "* *")
	for expr, want := range map[string]string{"*/": "Missing steps for expression: */", "-": "Missing values for range: -",
		"1-": "Missing values for range: 1-"} {
		if err := expectParseError(t, seconds, expr); err != nil && !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) error = %q, want it to contain %q", expr, err, want)
		}
	}
	five := parser.New(singleFieldDefinition(t, model.Minute, model.Hour, model.DayOfMonth, model.Month, model.DayOfWeek))
	mustParse(t, five, "* *   * * *")
}

func TestTrailingCommasIssue368(t *testing.T) {
	cases := []struct {
		ct   model.CronType
		expr string
	}{
		{model.Cron4j, "1, * * * *"}, {model.Quartz, "1, * * * * ?"}, {model.Spring, "1,2, * * * * ?"},
		{model.Unix, "1, * * * *"}, {model.Unix, "0 0 0 ? * #"},
	}
	for _, c := range cases {
		expectParseError(t, newParser(c.ct), c.expr)
	}
}

func dayOfYearDefinition(t *testing.T) *model.CronDefinition {
	d, err := model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().And().
		WithYear().WithValidRange(1970, 2099).Optional().And().
		WithDayOfYear().SupportsQuestionMark().WithValidRange(1, 366).Optional().And().
		WithCronValidation(model.EnsureEitherDayOfYearOrMonth()).
		WithCronValidation(model.EnsureEitherDayOfWeekOrDayOfMonth()).
		Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a Quartz-like definition with day of year: %v", err)
	}
	return d
}

func TestQuartzPeriods(t *testing.T) {
	p := newParser(model.Quartz)
	mustParse(t, p, "0/59 0/59 0/23 1/30 1/11 ? 2017/3")
	expectParseError(t, p, "0/0 0 0 1 1 ? 2017/3")
	expectParseError(t, p, "0/60 0 0 1 1 ? 2017/3")
	if got := mustParse(t, p, "0 0/1 * 1/1 * ? *").AsString(); got != "0 0/1 * 1/1 * ? *" {
		t.Errorf("Parse(Quartz %q).AsString() = %q, want %q", "0 0/1 * 1/1 * ? *", got, "0 0/1 * 1/1 * ? *")
	}
	multicron := "0 0|0|30|0 9|10|11|12 * * ? *"
	if got := mustParse(t, p, multicron).AsString(); got != multicron {
		t.Errorf("Parse(Quartz %q).AsString() = %q, want %q", multicron, got, multicron)
	}
}

func TestExtendedQuartzWithDayOfYear(t *testing.T) {
	p := parser.New(dayOfYearDefinition(t))
	for _, e := range []string{"0 0 0 ? * ? 2017 1/14", "0 0 0 ? * ? 2017 *", "0 0 0 1 * ? 2017 ?"} {
		mustParse(t, p, e)
	}
	expectParseError(t, p, "0 0 0 ? * #")
}

func TestCompositeCronParsing(t *testing.T) {
	p := parser.New(dayOfYearDefinition(t))
	for _, expr := range []string{"2 0 0 1 * ? 2000 ? || 5 0 0 1 * ? 2017 ?",
		"5 3 0 ? * ? 1998 * || 1 0 0 ? * * 2023 ? || 3 3 * * * ? 2015 ?"} {
		c, ok := mustParse(t, p, expr).(*model.CompositeCron)
		if !ok {
			t.Fatalf("Parse(%q) is a %T, want a *model.CompositeCron", expr, mustParse(t, p, expr))
		}
		want := strings.Split(expr, "||")
		crons := c.Crons()
		if len(crons) != len(want) {
			t.Fatalf("Parse(%q) has %d crons, want %d", expr, len(crons), len(want))
		}
		for i, member := range crons {
			if got := member.AsString(); got != strings.TrimSpace(want[i]) {
				t.Errorf("Parse(%q): member %d = %q, want %q", expr, i, got, strings.TrimSpace(want[i]))
			}
		}
	}
}

func TestSlashIncrementValidation(t *testing.T) {
	b := model.DefineCron().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).WithStrictRange().And().
		WithMonth().WithValidRange(1, 12).WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 6).WithMondayDoWValue(1).WithStrictRange().And().
		MatchDayOfWeekAndDayOfMonth()
	for _, n := range []model.CronNickname{model.Yearly, model.Annually, model.Monthly, model.Weekly, model.Daily,
		model.Midnight, model.Hourly} {
		b.WithSupportedNickname(n)
	}
	d, err := b.Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a Unix-like definition matching day of week and day of month: %v", err)
	}
	p := parser.New(d)
	expectParseError(t, p, "1 /2 * * *")
	mustParse(t, p, "1 */5 * * *")
	mustParse(t, p, "1 0/2 * * *")
}

func TestUnixMonthPeriodIssue413(t *testing.T) {
	_, err := newParser(model.Unix).Parse("* * * */12 *")
	if err != nil && err.Error() != "Failed to parse '* * * */12 *'. Period 12 not in range [1, 12]" {
		t.Errorf("Parse(Unix %q) error = %q, want none or %q", "* * * */12 *", err, "Failed to parse '* * * */12 *'. Period 12 not in range [1, 12]")
	}
}

func TestUnixRejectsQuestionMarkIssue499(t *testing.T) {
	expectParseError(t, newParser(model.Unix), "12 1 * ? *")
}

func hashFieldParser() *parser.FieldParser {
	return parser.NewFieldParser(model.NewFieldConstraintsBuilder().AddHashSupport().Build())
}

func mustParseField(t *testing.T, p *parser.FieldParser, expression string) model.FieldExpression {
	t.Helper()
	e, err := p.Parse(expression)
	if err != nil {
		t.Fatalf("FieldParser.Parse(%q): %v", expression, err)
	}
	return e
}

func TestFieldParserShapes(t *testing.T) {
	p := hashFieldParser()
	if _, ok := mustParseField(t, p, "*").(*model.Always); !ok {
		t.Errorf("FieldParser.Parse(%q) = %#v, want *model.Always", "*", mustParseField(t, p, "*"))
	}
	if e, ok := mustParseField(t, p, "*/5").(*model.Every); !ok || e.Period != 5 {
		t.Errorf("FieldParser.Parse(%q) = %#v, want every 5", "*/5", e)
	} else if _, always := e.Expression.(*model.Always); !always {
		t.Errorf("FieldParser.Parse(%q) steps over %#v, want *", "*/5", e.Expression)
	}
	if on, ok := mustParseField(t, p, "5").(*model.On); !ok || on.Time != 5 {
		t.Errorf("FieldParser.Parse(%q) = %#v, want On 5", "5", on)
	}
	if on, ok := mustParseField(t, p, "5#3").(*model.On); !ok || on.Time != 5 || on.Nth != 3 || on.SpecialChar.Value() != model.Hash {
		t.Errorf("FieldParser.Parse(%q) = %#v, want On 5#3", "5#3", on)
	}
	if and, ok := mustParseField(t, p, "3,4").(*model.And); !ok || and.AsString() != "3,4" || len(and.Expressions()) != 2 {
		t.Errorf("FieldParser.Parse(%q) = %#v, want And of 3 and 4", "3,4", and)
	}
	if b, ok := mustParseField(t, p, "3-4").(*model.Between); !ok || b.From != model.IntegerFieldValue(3) || b.To != model.IntegerFieldValue(4) {
		t.Errorf("FieldParser.Parse(%q) = %#v, want Between 3 and 4", "3-4", b)
	}
	e, ok := mustParseField(t, p, "10-40/5").(*model.Every)
	if !ok || e.Period != 5 {
		t.Fatalf("FieldParser.Parse(%q) = %#v, want every 5", "10-40/5", e)
	}
	if b, ok := e.Expression.(*model.Between); !ok || b.From != model.IntegerFieldValue(10) || b.To != model.IntegerFieldValue(40) {
		t.Errorf("FieldParser.Parse(%q) steps over %#v, want 10-40", "10-40/5", e.Expression)
	}
}

func TestFieldParserRejectsUnsupportedHash(t *testing.T) {
	if _, err := parser.NewFieldParser(model.NewFieldConstraintsBuilder().Build()).Parse("5#3"); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("FieldParser.Parse(%q) without # support: error %v, want an invalid-argument error", "5#3", err)
	}
}

func TestFieldParserLWErrors(t *testing.T) {
	p := hashFieldParser()
	for _, expr := range []string{"abLW", "LWlw", "1LW7"} {
		_, err := p.Parse(expr)
		want := "Expected: LW, found: " + strings.ReplaceAll(expr, "LW", "")
		if err == nil || err.Error() != want {
			t.Errorf("FieldParser.Parse(%q) error = %v, want %q", expr, err, want)
		}
	}
}
