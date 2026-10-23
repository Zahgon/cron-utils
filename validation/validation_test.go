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
// Ported to Go from com/cronutils/validation/CronValidatorTest.java and
// com/cronutils/validator/CronValidatorQuartzIntegrationTest.java.
//
// The original validated a bean through Jakarta Bean Validation; here the validator is called directly.

package validation

import (
	"strings"
	"testing"

	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func TestCronValidatorSpring(t *testing.T) {
	cases := []struct {
		expr  string
		valid bool
	}{
		{"0 0 * * * *", true},
		{"*/10 * * * * *", true},
		{"0 0 8-10 * * *", true},
		{"0 0 6,19 * * *", true},
		{"0 0/30 8-10 * * *", true},
		{"0 0 9-17 * * MON-FRI", true},
		{"0 0 0 25 12 ?", true},
		{"0 0 0 L 12 ?", false},
		{"1,2, * * * * *", false},
		{"1- * * * * *", false},
		{"java.lang.Runtime.getRuntime().exec('touch /tmp/pwned'); // 4 5 [${''.getClass().forName('javax.script.ScriptEngineManager').newInstance().getEngineByName('js').eval(validatedValue)}]", false},
	}
	v := CronValidator{Type: model.Spring}
	for _, c := range cases {
		expr := c.expr
		valid, reason, err := v.IsValid(&expr)
		if err != nil || valid != c.valid || (!valid && reason == "") {
			t.Errorf("IsValid(%q) = %v, %q, %v; want %v", c.expr, valid, reason, err, c.valid)
		}
	}
	if valid, _, err := v.IsValid(nil); !valid || err != nil {
		t.Errorf("IsValid(nil) = %v, %v; want valid", valid, err)
	}
}

func TestCronValidatorQuartzIntegration(t *testing.T) {
	p := parser.New(model.InstanceDefinitionFor(model.Quartz))
	parse := func(e string) model.Cron {
		c, err := p.Parse(e)
		if err != nil {
			t.Fatalf("Parse(%q): %v", e, err)
		}
		return c
	}
	if err := parse("0 0 0 * JUL-AUG ? *").Validate(); err != nil {
		t.Errorf("Validate() of Quartz %q: %v; want no error", "0 0 0 * JUL-AUG ? *", err)
	}
	for _, m := range []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"} {
		parse("0 0 0 * " + strings.ToUpper(m) + " ? *")
	}
	parse("0 0 0 ? * MON-FRI *")
	for _, dow := range []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"} {
		parse("0 0 0 ? * " + dow + " *")
	}
	for _, e := range []string{"0 10,44 14 ? 3 WED", "0 0 12 ? * FRI-SAT", "0 0 12 ? * SAT-SUN", "20-10 0 0 ? * 3",
		"0 40-20 0 ? * 3", "0 0 12-2 ? * 3", "0 0 0 24-7 * ?", "0 0 0 ? 10-3 3", "0 0 0 ? * 5-1"} {
		parse(e)
	}
}
