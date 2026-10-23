// Copyright 2019 fahmpeermoh
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
// Ported to Go from com/cronutils/converter/CronConverterTest.java.
//
// The converter takes the current day as its reference; the expectations hold in the northern summer (America/St_Johns
// is on daylight time), so the test fixes the reference day to 2024-07-15.

package converter

import (
	"testing"
	"time"
)

func TestCronConverter(t *testing.T) {
	cases := []struct{ zone, input, want string }{
		{"Pacific/Pago_Pago", "15 * * * *", "15 * * * *"},
		{"Antarctica/Casey", "? * * * *", "? * * * *"},
		{"Antarctica/Troll", "45 * * * *", "45 * * * *"},
		{"Pacific/Chatham", "15 * * * *", "30 * * * *"},
		{"Asia/Colombo", "45 * * ? *", "15 * * ? *"},
		{"Asia/Colombo", "0/45 * * ? *", "0/45 * * ? *"},
		{"Australia/Eucla", "13 * * ? *", "28 * * ? *"},
		{"America/St_Johns", "0 0/15 * * * ?", "30 0/15 * * * ?"},
		{"America/St_Johns", "0 8 * * ?", "30 10 * * ?"},
		{"America/St_Johns", "0 0/1 * * ?", "30 0/1 * * ?"},
		{"America/St_Johns", "20 0 * * ?", "50 2 * * ?"},
		{"Asia/Kolkata", "20 0 * * ?", "50 18 * * ?"},
	}
	reference := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC)
	for _, c := range cases {
		from, err := time.LoadLocation(c.zone)
		if err != nil {
			t.Fatalf("LoadLocation(%q): %v", c.zone, err)
		}
		conv := New()
		conv.now = func() time.Time { return reference }
		if got, err := conv.Using(c.input).From(from).To(time.UTC).Convert(); err != nil || got != c.want {
			t.Errorf("convert %q from %s to UTC = %q, %v; want %q", c.input, c.zone, got, err, c.want)
		}
	}
}

func TestCronConverterReportsMisuse(t *testing.T) {
	if _, err := New().Using("0 8 * * ?").To(time.UTC).Convert(); err == nil {
		t.Errorf("Convert of %q after To(UTC) without From: no error, want an error", "0 8 * * ?")
	}
	if _, err := New().Using("99999999999 8 * * ?").From(time.UTC).To(time.UTC).Convert(); err == nil {
		t.Errorf("Convert of %q from UTC to UTC: no error, want an error", "99999999999 8 * * ?")
	}
}
