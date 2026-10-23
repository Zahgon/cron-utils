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
// Ported to Go from com/cronutils/utils/descriptor/{CronDescriptorCron4jIntegrationTest,
// CronDescriptorCron4jIntegrationZhTest,CronDescriptorQuartzIntegrationTest,CronDescriptorTest,Issue227Test,
// Issue281Test,Issue343Test,TestDescriptor}.java and com/cronutils/{Issue338Test,Issue439Test,Issue440Test,
// Issue462Test,Issue477Test}.java.
//
// CronDescriptorTest described crons of a mocked definition; here the crons belong to a definition without a
// day-of-week field, which describes days of the week the same way. Not ported: the disabled cases of TestDescriptor
// and Issue403Test, and SecondsDescriptorTest (the package-private refactoring sketch it covers has no Go counterpart).

package descriptor_test

import (
	"testing"

	"github.com/jmrozanec/cron-utils/descriptor"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func describer(t *testing.T, tag string) *descriptor.CronDescriptor {
	t.Helper()
	d, err := descriptor.InstanceFor(tag)
	if err != nil {
		t.Fatalf("InstanceFor(%q): %v", tag, err)
	}
	return d
}

func mustParse(t *testing.T, d *model.CronDefinition, expression string) model.Cron {
	t.Helper()
	c, err := parser.New(d).Parse(expression)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expression, err)
	}
	return c
}

func expectDescription(t *testing.T, d *descriptor.CronDescriptor, c model.Cron, want string) {
	t.Helper()
	got, err := d.Describe(c)
	if err != nil || got != want {
		t.Errorf("Describe(%s) in language %q = %q, %v; want %q", c.AsString(), d.Bundle().Language(), got, err, want)
	}
}

type describeCase struct{ expr, want string }

func checkDescriptions(t *testing.T, tag string, ct model.CronType, cases []describeCase) {
	t.Helper()
	d := describer(t, tag)
	def := model.InstanceDefinitionFor(ct)
	for _, c := range cases {
		expectDescription(t, d, mustParse(t, def, c.expr), c.want)
	}
}

func TestDescribeCron4j(t *testing.T) {
	checkDescriptions(t, "en-GB", model.Cron4j, []describeCase{
		{"0-10 11 * * *", "every minute between 11:00 and 11:10"},
		{"* * * * *", "every minute"},
		{"*/1 * * * *", "every minute"},
		{"0/1 * * * *", "every minute"},
		{"*/5 * * * *", "every 5 minutes"},
		{"0/5 * * * *", "every 5 minutes"},
		{"30 11 * * *", "at 11:30"},
		{"0 23 * * MON-FRI", "at 23:00 every day between Monday and Friday"},
		{"0 23 * * 1-5", "at 23:00 every day between Monday and Friday"},
	})
	d := describer(t, "en-GB")
	for _, e := range []string{"* 1,2,3,4,5,6 * 1,2,3 *", "* 1 1,2 * 4"} {
		if _, err := d.Describe(mustParse(t, model.InstanceDefinitionFor(model.Cron4j), e)); err != nil {
			t.Errorf("Describe(Cron4j %q) in en-GB: %v; want a description", e, err)
		}
	}
}

func TestDescribeCron4jChinese(t *testing.T) {
	checkDescriptions(t, "zh", model.Cron4j, []describeCase{
		{"0-10 11 * * *", "每 分钟 在 11:00 和 11:10"},
		{"* * * * *", "每 分钟"},
		{"*/1 * * * *", "每 分钟"},
		{"0/1 * * * *", "每 分钟"},
		{"*/5 * * * *", "每 5 分钟"},
		{"0/5 * * * *", "每 5 分钟"},
		{"30 11 * * *", "在 11:30"},
	})
}

func TestDescribeQuartz(t *testing.T) {
	checkDescriptions(t, "en-GB", model.Quartz, []describeCase{
		{"* * 1,2,3,4 * * ? *", "every second at 1, 2, 3 and 4 hours"},
		{"* * 1,2,3,4,6-9 * * ? *", "every second at 1, 2, 3 and 4 hours and every hour between 6 and 9"},
		{"0 2-59/3 1,9,22 11-26 1-6 ?", "every 3 minutes between 2 and 59 at 1, 9 and 22 hours every day between 11 and 26 every month between January and June"},
		{"* * * * * ?", "every second"},
		{"*/45 * * * * ?", "every 45 seconds"},
		{"0 0 * * * ?", "every hour"},
		{"0 0 0/1 * * ?", "every hour"},
		{"0 0 12 * * ?", "at 12:00"},
		{"0 15 10 ? * *", "at 10:15"},
		{"0 15 10 * * ?", "at 10:15"},
		{"0 15 10 * * ? *", "at 10:15"},
		{"0 15 10 * * ? 2005", "at 10:15 at 2005 year"},
		{"0 * 14 * * ?", "every minute at 14 hour"},
		{"0 0/5 14 * * ?", "every 5 minutes from minute 0 at 14 hour"},
		{"0 0/5 14,18 * * ?", "every 5 minutes from minute 0 at 14 and 18 hours"},
		{"0 0 0 3/5 * ?", "at 00:00 every 5 days from day 3"},
		{"0 0 0 ? * 3/5", "at 00:00 every 5 days from day Tuesday"},
		{"0 0 2/4 * * ?", "every 4 hours at minute 0"},
		{"* 0/1 * ? * TUE", "every second every minute from minute 0 at Tuesday day"},
		{"0 0 0 ? * MON-FRI *", "at 00:00 every day between Monday and Friday"},
		{"*/4 */6 */8 */2 * ? */2", "every 4 seconds every 6 minutes every 8 hours every 2 days every 2 years"},
		{"0 0 0 3/5 * ? *", "at 00:00 every 5 days from day 3"},
		{"* 0 * * * ?", "every second at minute 00 of every hour"},
		{"* 2,1/31 * * * ?", "every second at minute 2 and every 31 minutes from minute 1"},
	})
	checkDescriptions(t, "en", model.Quartz, []describeCase{
		{"0 0 11 L-2 * ?", "at 11:00 2 days before the last day of the month"},
		{"0 0 11 L-1 * ?", "at 11:00 one day before the last day of the month"},
	})
	checkDescriptions(t, "en-US", model.Quartz, []describeCase{
		{"0 5-35/30 * * * ?", "every 30 minutes between 5 and 35"},
		{"5-35/30 * * * * ?", "every 30 seconds between 5 and 35"},
	})
	checkDescriptions(t, "fr-FR", model.Quartz, []describeCase{{"* * * * * ? *", "chaque seconde"}})
}

func TestDescribeDefaultLocale(t *testing.T) {
	d, err := descriptor.Instance()
	if err != nil {
		t.Fatalf("descriptor.Instance() for the default locale: %v", err)
	}
	quartz := model.InstanceDefinitionFor(model.Quartz)
	for _, e := range []string{"* * * * * ?", "* * * ? * * *", "* * * * * ? *"} {
		expectDescription(t, d, mustParse(t, quartz, e), "every second")
	}
}

func TestDescribeUnixAndSpring(t *testing.T) {
	checkDescriptions(t, "en-GB", model.Unix, []describeCase{
		{"2,1/31 * * * *", "at minute 2 and every 31 minutes from minute 1"},
		{"*/2 */2 * * *", "every 2 minutes every 2 hours"},
		{"* */2 * * *", "every minute every 2 hours"},
		{"*/1 */2 * * *", "every minute every 2 hours"},
	})
	checkDescriptions(t, "en", model.Spring, []describeCase{
		{"* */30 * * * *", "every second every 30 minutes"},
		{"0 */30 * * * *", "every 30 minutes"},
	})
}

func TestDescribeSpringIssue343(t *testing.T) {
	cases := []describeCase{
		{"0 0 * * * *", "every hour"},
		{"*/10 * * * * *", "every 10 seconds"},
		{"0 0 8-10 * * *", "every hour between 8 and 10"},
		{"0 0 6,19 * * *", "at 6 and 19 hours"},
		{"0 0/30 8-10 * * *", "every 30 minutes from minute 0 every hour between 8 and 10"},
		{"0 0 9-17 * * MON-FRI", "every hour between 9 and 17 every day between Monday and Friday"},
	}
	checkDescriptions(t, "en", model.Spring, cases)
	workaround, err := model.DefineCron().
		WithSeconds().And().WithMinutes().And().WithHours().And().
		WithDayOfMonth().SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().And().
		Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a Spring-like definition with Monday = 2: %v", err)
	}
	d := describer(t, "en")
	for _, c := range cases {
		expectDescription(t, d, mustParse(t, workaround, c.expr), c.want)
	}
}

func TestDescribeMonthStepsIssue281(t *testing.T) {
	quartz := model.InstanceDefinitionFor(model.Quartz)
	for _, e := range []string{"0 0 0 24 1/12 ?", "0 0 0 24/1 1/12 ?", "0 0 0 1/31 7 ?"} {
		mustParse(t, quartz, e)
	}
	if _, err := parser.New(quartz).Parse("0 0 0 24 1/13 ?"); err == nil {
		t.Errorf("Parse(%q) with the Quartz definition: no error, want an error", "0 0 0 24 1/13 ?")
	}
}

// fieldsDefinition is a definition without a day-of-week field, for crons assembled field by field.
func fieldsDefinition(t *testing.T) *model.CronDefinition {
	d, err := model.DefineCron().WithMinutes().And().Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a minutes-only definition: %v", err)
	}
	return d
}

func constraints() *model.FieldConstraints {
	return model.NewFieldConstraintsBuilder().AddHashSupport().AddLSupport().AddWSupport().Build()
}

func cronOf(t *testing.T, fields ...*model.CronField) model.Cron {
	return model.NewSingleCron(fieldsDefinition(t), fields)
}

func cf(name model.CronFieldName, e model.FieldExpression) *model.CronField {
	return model.NewCronField(name, e, constraints())
}

func TestDescribeAssembledCrons(t *testing.T) {
	d := describer(t, "en-GB")
	every3 := model.NewEvery(3)
	between := func(a, b int) model.FieldExpression {
		return model.NewBetween(model.IntegerFieldValue(a), model.IntegerFieldValue(b))
	}
	always := model.AlwaysExpression()
	cases := []struct {
		fields []*model.CronField
		want   string
	}{
		{[]*model.CronField{cf(model.Second, every3)}, "every 3 seconds"},
		{[]*model.CronField{cf(model.Minute, every3)}, "every 3 minutes"},
		{[]*model.CronField{cf(model.Hour, every3), cf(model.Minute, model.NewOn(3))}, "every 3 hours at minute 3"},
		{[]*model.CronField{cf(model.Minute, between(0, 10)), cf(model.Hour, model.NewOn(11))}, "every minute between 11:00 and 11:10"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(11)), cf(model.Minute, model.NewOn(30)), cf(model.DayOfWeek, between(2, 6))},
			"at 11:30 every day between Tuesday and Saturday"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(11)), cf(model.Minute, always), cf(model.Second, always)}, "at 11:00"},
		{[]*model.CronField{cf(model.Hour, always), cf(model.Minute, always), cf(model.Second, always), cf(model.Month, model.NewOn(2))},
			"every second at February month"},
		{[]*model.CronField{cf(model.Hour, always), cf(model.Minute, always), cf(model.Month, between(2, 3))},
			"every minute every month between February and March"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(10)), cf(model.Minute, model.NewOn(15)),
			cf(model.DayOfWeek, model.NewOnSpecial(2, model.L, model.DefaultNth))}, "at 10:15 last Tuesday of every month"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(10)), cf(model.Minute, model.NewOn(15)),
			cf(model.DayOfWeek, model.NewOnSpecial(2, model.Hash, 2))}, "at 10:15 Tuesday 2 of every month"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(10)), cf(model.Minute, model.NewOn(15)),
			cf(model.DayOfMonth, model.NewOnSpecial(model.DefaultNth, model.L, model.DefaultNth))}, "at 10:15 last day of month"},
		{[]*model.CronField{cf(model.Hour, model.NewOn(10)), cf(model.Minute, model.NewOn(15)),
			cf(model.DayOfMonth, model.NewOnSpecial(22, model.W, model.DefaultNth))}, "at 10:15 the nearest weekday to the 22 of the month"},
	}
	for _, c := range cases {
		expectDescription(t, d, cronOf(t, c.fields...), c.want)
	}
}

func TestDescriptorBundle(t *testing.T) {
	b := describer(t, "en-GB").Bundle()
	for _, key := range []string{"hours", "minutes", "seconds"} {
		if _, err := b.String(key); err != nil {
			t.Errorf("Bundle().String(%q) for en-GB: %v; want a translation", key, err)
		}
	}
}

func TestTimeDescriptor(t *testing.T) {
	quartz := model.InstanceDefinitionFor(model.Quartz)
	cases := []struct{ tag, expr, want string }{
		{"en-GB", "* * * * * ? *", "every second"},
		{"en-GB", "*/1 * * * * ? *", "every second"},
		{"en-GB", "*/2 * * * * ? *", "every 2 seconds"},
		{"de", "* * * * * ? *", "jede sekunde"},
		{"it", "* * * * * ? *", "ogni secondo"},
		{"de", "*/1 * * * * ? *", "jede sekunde"},
		{"it", "*/1 * * * * ? *", "ogni secondo"},
		{"de", "*/2 * * * * ? *", "alle 2 sekunden"},
		{"it", "*/2 * * * * ? *", "ogni 2 secondi"},
	}
	for _, c := range cases {
		b, err := descriptor.BundleFor(c.tag)
		if err != nil {
			t.Fatalf("descriptor.BundleFor(%q): %v", c.tag, err)
		}
		got, err := descriptor.NewTimeDescriptor(b).Describe(mustParse(t, quartz, c.expr))
		if err != nil || got != c.want {
			t.Errorf("TimeDescriptor(%s).Describe(Quartz %q) = %q, %v; want %q", c.tag, c.expr, got, err, c.want)
		}
	}
}
