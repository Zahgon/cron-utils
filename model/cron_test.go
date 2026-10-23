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
// Ported to Go from com/cronutils/model/{CompositeCronTest,CronTest}.java and
// com/cronutils/Issue58UnixCronAsStringIntegrationTest.java.
//
// Not ported: the null-argument cases (Go values cannot be null) and CronTest.testSerialization (Go has no Java
// serialization; the canonical form it round-trips is covered by the parser tests).

package model_test

import (
	"testing"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/mapper"
	"github.com/jmrozanec/cron-utils/model"
)

func quartz() *model.CronDefinition { return model.InstanceDefinitionFor(model.Quartz) }

func composite(t *testing.T, crons ...model.Cron) *model.CompositeCron {
	t.Helper()
	c, err := model.NewCompositeCron(crons)
	if err != nil {
		t.Fatalf("NewCompositeCron(%v): %v", expressions(crons), err)
	}
	return c
}

// expressions returns each cron's expression, for failure messages.
func expressions(crons []model.Cron) []string {
	var s []string
	for _, c := range crons {
		s = append(s, c.AsString())
	}
	return s
}

func compositeFixtures(t *testing.T) (*model.CompositeCron, *model.CompositeCron) {
	d := quartz()
	c1 := mustParse(t, d, "0 0 0 15 8 ? 2015/2")
	c2 := mustParse(t, d, "0 0 0 16 9 ? 2015/2")
	c3 := mustParse(t, d, "0 0 0 17 10 ? 2015/2")
	return composite(t, c1, c2), composite(t, c2, c3)
}

func TestCompositeCronRejectsDifferentDefinitions(t *testing.T) {
	c1 := mustParse(t, quartz(), "0 0 0 15 8 ? 2015/2")
	c2 := mustParse(t, model.InstanceDefinitionFor(model.Unix), "0 0 1 * *")
	if _, err := model.NewCompositeCron([]model.Cron{c1, c2}); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("NewCompositeCron([Quartz %q, Unix %q]): error %v, want an invalid-argument error", c1.AsString(), c2.AsString(), err)
	}
}

func TestCompositeCronRejectsEmptyList(t *testing.T) {
	if _, err := model.NewCompositeCron(nil); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("NewCompositeCron(nil): error %v, want an invalid-argument error", err)
	}
}

func TestCompositeCronFieldAccessUnsupported(t *testing.T) {
	c, _ := compositeFixtures(t)
	if _, err := c.Retrieve(model.DayOfWeek); !cronerr.Is(err, cronerr.UnsupportedOperation) {
		t.Errorf("Retrieve(DAY_OF_WEEK) of composite %q: error %v, want unsupported operation", c.AsString(), err)
	}
	if _, err := c.Fields(); !cronerr.Is(err, cronerr.UnsupportedOperation) {
		t.Errorf("Fields() of composite %q: error %v, want unsupported operation", c.AsString(), err)
	}
}

func TestCompositeCronAsStringAndDefinition(t *testing.T) {
	c, _ := compositeFixtures(t)
	if got := c.AsString(); got != "0 0 0 15|16 8|9 ? 2015/2" {
		t.Errorf("AsString() of composite [%q %q] = %q, want %q", "0 0 0 15 8 ? 2015/2", "0 0 0 16 9 ? 2015/2", got, "0 0 0 15|16 8|9 ? 2015/2")
	}
	if !c.Definition().Equal(quartz()) {
		t.Errorf("Definition() of composite %q is not the Quartz definition; want the Quartz definition", c.AsString())
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() of composite %q: %v; want no error", c.AsString(), err)
	}
}

func TestCompositeCronEquivalent(t *testing.T) {
	c1, c2 := compositeFixtures(t)
	if !c1.Equivalent(c1) {
		t.Errorf("Equivalent(%q, itself) = false, want true", c1.AsString())
	}
	if c1.Equivalent(c2) {
		t.Errorf("Equivalent(%q, %q) = true, want false", c1.AsString(), c2.AsString())
	}
}

func TestCompositeCronIssue318(t *testing.T) {
	d := quartz()
	c := composite(t, mustParse(t, d, "0 0 9 * * ? *"), mustParse(t, d, "0 0 10 * * ? *"),
		mustParse(t, d, "0 30 11 * * ? *"), mustParse(t, d, "0 0 12 * * ? *"))
	if got := c.AsString(); got != "0 0|0|30|0 9|10|11|12 * * ? *" {
		t.Errorf("AsString() of composite [%q %q %q %q] = %q, want %q", "0 0 9 * * ? *", "0 0 10 * * ? *",
			"0 30 11 * * ? *", "0 0 12 * * ? *", got, "0 0|0|30|0 9|10|11|12 * * ? *")
	}
	et, err := executiontime.ForCron(c)
	if err != nil {
		t.Fatalf("executiontime.ForCron(composite %q): %v", c.AsString(), err)
	}
	at := func(y, mo, d, h, mi int) javatime.ZonedDateTime {
		z, err := javatime.OfFields(y, mo, d, h, mi, 0, 0, javatime.UTC)
		if err != nil {
			t.Fatalf("javatime.OfFields(%d-%02d-%02d %02d:%02d, UTC): %v", y, mo, d, h, mi, err)
		}
		return z
	}
	cases := [][2]javatime.ZonedDateTime{
		{at(2015, 4, 15, 0, 0), at(2015, 4, 15, 9, 0)},
		{at(2015, 4, 15, 9, 30), at(2015, 4, 15, 10, 0)},
		{at(2015, 4, 15, 11, 0), at(2015, 4, 15, 11, 30)},
		{at(2015, 4, 15, 11, 30), at(2015, 4, 15, 12, 0)},
		{at(2015, 4, 15, 12, 30), at(2015, 4, 16, 9, 0)},
	}
	for _, c := range cases {
		got, ok, err := et.NextExecution(c[0])
		if err != nil || !ok || !got.Equal(c[1]) {
			t.Errorf("NextExecution(%s) of composite %q = %s, %v, %v; want %s", c[0], "0 0|0|30|0 9|10|11|12 * * ? *", got, ok, err, c[1])
		}
	}
}

func TestCompositeCronIssue263(t *testing.T) {
	expr := "0 1 0 ? 1/1 MON#2|MON#3|MON#4|MON#5 *"
	want := "0 1 0 ? 1/1 2#2|2#3|2#4|2#5 *"
	if got := mustParse(t, quartz(), expr).AsString(); got != want {
		t.Errorf("Parse(%q).AsString() = %q, want %q", expr, got, want)
	}
}

func TestSingleCronFieldAccess(t *testing.T) {
	f := model.NewCronField(model.Second, textExpression("somestring"), model.NewFieldConstraintsBuilder().Build())
	c := model.NewSingleCron(quartz(), []*model.CronField{f})
	got, err := c.Retrieve(model.Second)
	if err != nil || got != f {
		t.Errorf("Retrieve(SECOND) of single cron %q = %v, %v; want the field %v", "somestring", got, err, f)
	}
	fields, err := c.Fields()
	if err != nil || len(fields) != 1 || fields[0] != f {
		t.Errorf("Fields() of single cron %q = %v, %v; want [%v]", "somestring", fields, err, f)
	}
	if s := c.AsString(); s != "somestring" {
		t.Errorf("AsString() of a single cron with SECOND field %q = %q, want %q", "somestring", s, "somestring")
	}
}

func TestCronEquivalent(t *testing.T) {
	unix := model.InstanceDefinitionFor(model.Unix)
	cron1 := mustParse(t, unix, "* * * * MON")
	cron2 := mustParse(t, unix, "*/1 * * * 1")
	cron3 := mustParse(t, unix, "0 * * * *")
	cron4 := mustParse(t, quartz(), "0 * * ? * MON *")
	cases := []struct {
		m     *mapper.CronMapper
		other model.Cron
		want  bool
	}{
		{mapper.SameCronMapper(unix), cron2, true},
		{mapper.SameCronMapper(unix), cron3, false},
		{mapper.FromQuartzToCron4j(), cron4, true},
	}
	for _, c := range cases {
		got, err := c.m.Equivalent(cron1, c.other)
		if err != nil || got != c.want {
			t.Errorf("Equivalent(%s, %s) = %v, %v; want %v", cron1.AsString(), c.other.AsString(), got, err, c.want)
		}
	}
}

func TestUnixCronAsStringIssue58(t *testing.T) {
	unix := model.InstanceDefinitionFor(model.Unix)
	cases := map[string][]string{
		"0 0/1 * * *": {"0 0/1 * * *", "0 /1 * * *", "0 0 * * *"},
		"0 1/2 * * *": {"0 1/2 * * *"},
		"0/1 * * * *": {"0/1 * * * *", "/1 * * * *", "0 * * * *"},
		"1/2 * * * *": {"1/2 * * * *"},
	}
	for expr, accepted := range cases {
		got := mustParse(t, unix, expr).AsString()
		ok := false
		for _, a := range accepted {
			ok = ok || got == a
		}
		if !ok {
			t.Errorf("Parse(%q).AsString() = %q, want one of %v", expr, got, accepted)
		}
	}
}
