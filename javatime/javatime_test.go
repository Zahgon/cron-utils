// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package javatime

import "testing"

func mustZone(t *testing.T, id string) Zone {
	t.Helper()
	z, err := ZoneOf(id)
	if err != nil {
		t.Fatalf("ZoneOf(%q): %v", id, err)
	}
	return z
}

func TestZonedDateTimeResolvesGapsAndOverlaps(t *testing.T) {
	ny := mustZone(t, "America/New_York")
	tests := []struct {
		in, want string
	}{
		{"2024-03-10T02:30", "2024-03-10T03:30-04:00[America/New_York]"},
		{"2024-11-03T01:30", "2024-11-03T01:30-04:00[America/New_York]"},
		{"2024-11-03T01:30-05:00", "2024-11-03T01:30-05:00[America/New_York]"},
		{"2024-07-01T12:00:00.5", "2024-07-01T12:00:00.500-04:00[America/New_York]"},
	}
	for _, tc := range tests {
		got, err := ParseZoned(tc.in, ny)
		if err != nil {
			t.Fatalf("ParseZoned(%q): %v", tc.in, err)
		}
		if got.String() != tc.want {
			t.Errorf("ParseZoned(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestOverlapKeepsOffsetOnLocalChanges(t *testing.T) {
	ny := mustZone(t, "America/New_York")
	later, err := ParseZoned("2024-11-03T01:10-05:00", ny)
	if err != nil {
		t.Fatalf("ParseZoned(%q, America/New_York): %v", "2024-11-03T01:10-05:00", err)
	}
	if got, want := later.TruncatedTo(Hours).String(), "2024-11-03T01:00-05:00[America/New_York]"; got != want {
		t.Errorf("%s.TruncatedTo(Hours) = %s, want %s", later, got, want)
	}
	if got, want := later.PlusHours(1).String(), "2024-11-03T02:10-05:00[America/New_York]"; got != want {
		t.Errorf("%s.PlusHours(1) = %s, want %s", later, got, want)
	}
}

func TestZoneIDs(t *testing.T) {
	tests := map[string]string{"Z": "Z", "+5": "+05:00", "-0330": "-03:30", "UTC": "UTC", "UTC+0": "UTC",
		"GMT+5": "GMT+05:00", "UT-01:30": "UT-01:30", "+01:00:30": "+01:00:30"}
	for in, want := range tests {
		if got := mustZone(t, in).ID(); got != want {
			t.Errorf("ZoneOf(%q).ID() = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "Mars/Olympus", "+19", "EST", "Local", "12:00"} {
		if _, err := ZoneOf(bad); err == nil {
			t.Errorf("ZoneOf(%q): no error, want an error", bad)
		}
	}
}

func TestDurationString(t *testing.T) {
	tests := []struct {
		d    Duration
		want string
	}{
		{Duration{0, 0}, "PT0S"},
		{Duration{5400, 0}, "PT1H30M"},
		{Duration{29172, 345_000_000}, "PT8H6M12.345S"},
		{Duration{-1, 500_000_000}, "PT-0.5S"},
		{Duration{-3600, 0}, "PT-1H"},
		{Duration{59, 1}, "PT59.000000001S"},
	}
	for _, tc := range tests {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("Duration{%d, %d}.String() = %q, want %q", tc.d.seconds, tc.d.nanos, got, tc.want)
		}
	}
}

func TestDateErrors(t *testing.T) {
	tests := map[[3]int]string{
		{2023, 2, 29}: "Invalid date 'February 29' as '2023' is not a leap year",
		{2024, 4, 31}: "Invalid date 'APRIL 31'",
		{2024, 13, 1}: "Invalid value for MonthOfYear (valid values 1 - 12): 13",
		{2024, 1, 32}: "Invalid value for DayOfMonth (valid values 1 - 28/31): 32",
	}
	for in, want := range tests {
		_, err := DateOf(in[0], in[1], in[2])
		if err == nil || err.Error() != want {
			t.Errorf("DateOf(%v) error = %v, want %q", in, err, want)
		}
	}
}

func TestDateStringYears(t *testing.T) {
	tests := map[LocalDate]string{
		{12, 3, 4}:    "0012-03-04",
		{-5, 1, 1}:    "-0005-01-01",
		{10000, 1, 1}: "+10000-01-01",
		{2024, 2, 29}: "2024-02-29",
	}
	for d, want := range tests {
		if got := d.String(); got != want {
			t.Errorf("LocalDate{%d, %d, %d}.String() = %q, want %q", d.year, d.month, d.day, got, want)
		}
	}
}
