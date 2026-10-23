// Copyright 2014 jmrozanec
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
// Ported to Go from com/cronutils/utils/{CronFrequencyComparatorTest,DateUtilsTest}.java.
//
// Not ported: StringUtilsTest (the Java string helpers it covers have no Go counterpart; Go's strings package does
// that work).

package utils

import (
	"testing"

	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

func TestCronFrequencyComparator(t *testing.T) {
	start, err := javatime.OfFields(2018, 11, 5, 0, 0, 0, 0, javatime.UTC)
	if err != nil {
		t.Fatalf("javatime.OfFields(2018-11-05T00:00, UTC): %v", err)
	}
	end, err := javatime.OfFields(2018, 11, 11, 0, 0, 0, 0, javatime.UTC)
	if err != nil {
		t.Fatalf("javatime.OfFields(2018-11-11T00:00, UTC): %v", err)
	}
	c := CronFrequencyComparator{Start: start, End: end}
	p := parser.New(model.InstanceDefinitionFor(model.Spring53))
	parse := func(e string) model.Cron {
		cron, err := p.Parse(e)
		if err != nil {
			t.Fatalf("Parse(%q): %v", e, err)
		}
		return cron
	}
	weekdays := parse("0 0 9-17 * * MON-FRI")
	mornings := parse("0 0/30 8-10 * * *")
	if d, err := c.Compare(weekdays, mornings); err != nil || d <= 0 {
		t.Errorf("Compare(%q, %q) between %s and %s = %d, %v; want > 0", weekdays.AsString(), mornings.AsString(), start, end, d, err)
	}
	if d, err := c.Compare(mornings, weekdays); err != nil || d >= 0 {
		t.Errorf("Compare(%q, %q) between %s and %s = %d, %v; want < 0", mornings.AsString(), weekdays.AsString(), start, end, d, err)
	}
	if d, err := c.Compare(weekdays, parse("0 0 9-17 * * MON-FRI")); err != nil || d != 0 {
		t.Errorf("Compare(%q, %q) between %s and %s = %d, %v; want 0", weekdays.AsString(), "0 0 9-17 * * MON-FRI", start, end, d, err)
	}
}

func buenosAires(t *testing.T, year, month, day int) javatime.ZonedDateTime {
	t.Helper()
	z, err := javatime.ZoneOf("America/Argentina/Buenos_Aires")
	if err != nil {
		t.Fatalf("javatime.ZoneOf(%q): %v", "America/Argentina/Buenos_Aires", err)
	}
	d, err := javatime.OfFields(year, month, day, 1, 0, 0, 0, z)
	if err != nil {
		t.Fatalf("javatime.OfFields(%04d-%02d-%02dT01:00, %s): %v", year, month, day, z, err)
	}
	return d
}

func TestWorkdaysCountPolicies(t *testing.T) {
	saturday := buenosAires(t, 2018, 1, 6)
	for policy, want := range map[WeekendPolicy]int{ThursdayFriday: 2, FridaySaturday: 1, SaturdaySunday: 0} {
		got, err := WorkdaysCountFor(saturday, 1, nil, policy)
		if err != nil || got != want {
			t.Errorf("WorkdaysCountFor(%s (Saturday), 1 day, no holidays, policy %d) = %d, %v; want %d", saturday, policy, got, err, want)
		}
	}
}

func TestWorkdaysMay2018Argentina(t *testing.T) {
	holidays := []javatime.ZonedDateTime{buenosAires(t, 2018, 5, 1), buenosAires(t, 2018, 5, 25)}
	got, err := WorkdaysCount(buenosAires(t, 2018, 5, 1), buenosAires(t, 2018, 5, 31), holidays, SaturdaySunday)
	if err != nil || got != 21 {
		t.Errorf("WorkdaysCount(2018-05-01 to 2018-05-31 Buenos Aires, holidays %v, SaturdaySunday) = %d, %v; want 21", holidays, got, err)
	}
}
