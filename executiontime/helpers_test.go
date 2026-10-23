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
// Test helpers shared by the tests ported from com/cronutils/model/time/*Test.java and com/cronutils/Issue*Test.java.

package executiontime_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

// zone returns the zone with the given ID.
func zone(t *testing.T, id string) javatime.Zone {
	t.Helper()
	z, err := javatime.ZoneOf(id)
	if err != nil {
		t.Fatalf("javatime.ZoneOf(%q): %v", id, err)
	}
	return z
}

// utc is the region UTC, the time-zone the original tests ran in.
func utc(t *testing.T) javatime.Zone { return zone(t, "UTC") }

// at returns the zoned date-time of the given fields.
func at(t *testing.T, z javatime.Zone, year, month, day, hour, minute, second int) javatime.ZonedDateTime {
	t.Helper()
	d, err := javatime.OfFields(year, month, day, hour, minute, second, 0, z)
	if err != nil {
		t.Fatalf("javatime.OfFields(%d-%d-%d %d:%d:%d, %s): %v", year, month, day, hour, minute, second, z, err)
	}
	return d
}

// atNano returns the zoned date-time of the given fields with nanoseconds.
func atNano(t *testing.T, z javatime.Zone, year, month, day, hour, minute, second, nano int) javatime.ZonedDateTime {
	t.Helper()
	d, err := javatime.OfFields(year, month, day, hour, minute, second, nano, z)
	if err != nil {
		t.Fatalf("javatime.OfFields(%d-%d-%d %d:%d:%d.%09d, %s): %v", year, month, day, hour, minute, second, nano, z, err)
	}
	return d
}

// parseZoned reads an ISO-8601 zoned date-time: 2017-09-05T11:31:55.407-05:00, 2017-09-05T11:31Z or
// 2017-09-05T11:31+02:00[Europe/Paris]. Without a bracketed zone the zone is the offset.
func parseZoned(t *testing.T, text string) javatime.ZonedDateTime {
	t.Helper()
	var z javatime.Zone
	var err error
	if i := strings.IndexByte(text, '['); i >= 0 {
		z, err = javatime.ZoneOf(strings.TrimSuffix(text[i+1:], "]"))
		text = text[:i]
	} else if strings.HasSuffix(text, "Z") {
		z = javatime.UTC
	} else {
		cut := strings.LastIndexAny(text, "+-")
		z, err = javatime.OffsetOf(text[cut:])
	}
	if err != nil {
		t.Fatalf("zone of %q: %v", text, err)
	}
	d, err := javatime.ParseZoned(text, z)
	if err != nil {
		t.Fatalf("javatime.ParseZoned(%q): %v", text, err)
	}
	return d
}

// definition returns a predefined cron definition.
func definition(t model.CronType) *model.CronDefinition { return model.InstanceDefinitionFor(t) }

// mustParse parses an expression of a predefined dialect.
func mustParse(t *testing.T, ct model.CronType, expression string) model.Cron {
	t.Helper()
	return mustParseWith(t, definition(ct), expression)
}

// mustParseWith parses an expression of a definition.
func mustParseWith(t *testing.T, d *model.CronDefinition, expression string) model.Cron {
	t.Helper()
	c, err := parser.New(d).Parse(expression)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expression, err)
	}
	return c
}

// mustDefinition builds a custom definition.
func mustDefinition(t *testing.T, b *model.CronDefinitionBuilder) *model.CronDefinition {
	t.Helper()
	d, err := b.Instance()
	if err != nil {
		t.Fatalf("CronDefinitionBuilder.Instance() for a custom definition: %v; want a definition", err)
	}
	return d
}

// expressions remembers the expression each execution time was built from, for failure messages.
var expressions sync.Map

// forCron returns the execution times of a cron.
func forCron(t *testing.T, c model.Cron) executiontime.ExecutionTime {
	t.Helper()
	et, err := executiontime.ForCron(c)
	if err != nil {
		t.Fatalf("ForCron(%s): %v", c.AsString(), err)
	}
	expressions.Store(et, c.AsString())
	return et
}

// exprOf returns the expression an execution time was built from.
func exprOf(et executiontime.ExecutionTime) string {
	if e, ok := expressions.Load(et); ok {
		return e.(string)
	}
	return "<unknown cron>"
}

// next returns the next execution, failing the test when there is none.
func next(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) javatime.ZonedDateTime {
	t.Helper()
	n, ok, err := et.NextExecution(d)
	if err != nil || !ok {
		t.Fatalf("NextExecution(%s) of %q = present %v, error %v; want an execution", d, exprOf(et), ok, err)
	}
	return n
}

// last returns the last execution, failing the test when there is none.
func last(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) javatime.ZonedDateTime {
	t.Helper()
	l, ok, err := et.LastExecution(d)
	if err != nil || !ok {
		t.Fatalf("LastExecution(%s) of %q = present %v, error %v; want an execution", d, exprOf(et), ok, err)
	}
	return l
}

// noNext fails the test when there is a next execution.
func noNext(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) {
	t.Helper()
	n, ok, err := et.NextExecution(d)
	if err != nil || ok {
		t.Fatalf("NextExecution(%s) of %q = %s, present %v, error %v; want none", d, exprOf(et), n, ok, err)
	}
}

// noLast fails the test when there is a last execution.
func noLast(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) {
	t.Helper()
	l, ok, err := et.LastExecution(d)
	if err != nil || ok {
		t.Fatalf("LastExecution(%s) of %q = %s, present %v, error %v; want none", d, exprOf(et), l, ok, err)
	}
}

// isMatch reports whether d is an execution.
func isMatch(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) bool {
	t.Helper()
	ok, err := et.IsMatch(d)
	if err != nil {
		t.Fatalf("IsMatch(%s) of %q: %v; want no error", d, exprOf(et), err)
	}
	return ok
}

// timeToNext returns the time to the next execution.
func timeToNext(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) javatime.Duration {
	t.Helper()
	dur, ok, err := et.TimeToNextExecution(d)
	if err != nil || !ok {
		t.Fatalf("TimeToNextExecution(%s) of %q = present %v, error %v; want a duration", d, exprOf(et), ok, err)
	}
	return dur
}

// timeFromLast returns the time since the last execution.
func timeFromLast(t *testing.T, et executiontime.ExecutionTime, d javatime.ZonedDateTime) javatime.Duration {
	t.Helper()
	dur, ok, err := et.TimeFromLastExecution(d)
	if err != nil || !ok {
		t.Fatalf("TimeFromLastExecution(%s) of %q = present %v, error %v; want a duration", d, exprOf(et), ok, err)
	}
	return dur
}

// assertZoned fails the test when got is not want (same local date-time, offset and zone).
func assertZoned(t *testing.T, call string, want, got javatime.ZonedDateTime) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s = %s, want %s", call, got, want)
	}
}

// assertInstant fails the test when got is not the same instant as want.
func assertInstant(t *testing.T, call string, want, got javatime.ZonedDateTime) {
	t.Helper()
	if got.Compare(want) != 0 && (got.EpochSecond() != want.EpochSecond() || got.Nano() != want.Nano()) {
		t.Errorf("%s = %s, want the instant %s", call, got, want)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func contains(s, sub string) bool { return strings.Contains(s, sub) }
