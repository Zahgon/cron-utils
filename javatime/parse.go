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

import (
	"regexp"
	"strconv"

	"github.com/jmrozanec/cron-utils/cronerr"
)

var isoDateTime = regexp.MustCompile(`^([+-]?\d{4,10})-(\d{2})-(\d{2})[Tt](\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,9}))?)?([Zz]|[+-]\d{2}(?::?\d{2}(?::?\d{2})?)?)?$`)

// ParseZoned reads an ISO-8601 local date-time (2024-03-10T02:30, seconds and a fraction optional) and places it in
// zone, or reads an ISO-8601 date-time with an offset (2024-03-10T02:30Z, 2024-03-10T02:30+05:30) and converts the
// instant to zone.
func ParseZoned(text string, zone Zone) (ZonedDateTime, error) {
	m := isoDateTime.FindStringSubmatch(text)
	if m == nil {
		return ZonedDateTime{}, cronerr.New(cronerr.DateTime, "Text '%s' could not be parsed", text)
	}
	year, err := strconv.Atoi(m[1])
	if err != nil || (len(m[1]) > 4 && m[1][0] != '+' && m[1][0] != '-') || (m[1][0] == '+' && len(m[1]) <= 5) {
		return ZonedDateTime{}, cronerr.New(cronerr.DateTime, "Text '%s' could not be parsed", text)
	}
	month := digits(m[2])
	day := digits(m[3])
	hour := digits(m[4])
	minute := digits(m[5])
	second := 0
	if m[6] != "" {
		second = digits(m[6])
	}
	nano := 0
	if m[7] != "" {
		frac := m[7]
		for len(frac) < 9 {
			frac += "0"
		}
		nano = digits(frac)
	}
	date, err := DateOf(year, month, day)
	if err != nil {
		return ZonedDateTime{}, cronerr.Wrap(cronerr.DateTime, err, "Text '%s' could not be parsed: %s", text, err)
	}
	t, err := TimeOfNano(hour, minute, second, nano)
	if err != nil {
		return ZonedDateTime{}, cronerr.Wrap(cronerr.DateTime, err, "Text '%s' could not be parsed: %s", text, err)
	}
	dt := LocalDateTime{date, t}
	if m[8] == "" {
		return Of(dt, zone), nil
	}
	offsetText := m[8]
	if offsetText == "z" {
		offsetText = "Z"
	}
	offset, err := OffsetOf(offsetText)
	if err != nil {
		return ZonedDateTime{}, cronerr.Wrap(cronerr.DateTime, err, "Text '%s' could not be parsed: %s", text, err)
	}
	return OfInstant(dt.toEpochSecond(offset.offset), nano, zone), nil
}

// digits reads a run of ASCII digits the pattern has already matched.
func digits(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}
