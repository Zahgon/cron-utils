// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		args           []string
		stdout, stderr string
		status         int
	}{
		{[]string{"parse", "--type", "quartz", "--", "0 0 12 ? * mon-fri"}, "0 0 12 ? * 2-6\n", "", 0},
		{[]string{"validate", "--type", "unix", "--", "61 * * * *"},
			"invalid: Failed to parse cron expression. Value 61 not in range [0, 59]\n", "", 0},
		{[]string{"describe", "--type", "unix", "--locale", "de", "--", "0 12 * * MON-FRI"},
			"um 12:00 jede Tag zwischen Montag und Freitag\n", "", 0},
		{[]string{"next", "--type", "quartz", "--zone", "Europe/Paris", "--at", "2024-03-30T12:00", "--count", "2",
			"--", "0 30 2 * * ?"}, "2024-04-01T02:30+02:00[Europe/Paris]\n2024-04-02T02:30+02:00[Europe/Paris]\n", "", 0},
		{[]string{"time-to-next", "--type", "unix", "--zone", "UTC", "--at", "2024-01-01T00:00:30", "--", "* * * * *"},
			"PT30S\n", "", 0},
		{[]string{"map", "--from", "quartz", "--to", "unix", "--", "0 23 * ? * 1-5 *"}, "23 * * * 0-4\n", "", 0},
		{[]string{"parse", "--type", "unix", "--", "* * * *"}, "",
			"error: Cron expression contains 4 parts but we expect one of [5]\n", 1},
		{[]string{"describe", "--type", "quartz", "--", "0 0 12 * * ?||0 0 13 * * ?"}, "",
			"error: Currently not supported for CompositeCron\n", 1},
		{[]string{"next", "--type", "unix", "--zone", "Mars/Olympus", "--at", "2024-01-01T00:00", "--", "* * * * *"},
			"", "usage: invalid zone 'Mars/Olympus'\n", 2},
		{[]string{"map", "--from", "unix", "--to", "cron4j", "--", "* * * * *"}, "", "usage: no mapping from unix to cron4j\n", 2},
		{nil, "", "usage: missing command\n", 2},
	}
	for _, tc := range tests {
		var stdout, stderr bytes.Buffer
		status := run(tc.args, &stdout, &stderr)
		if status != tc.status || stdout.String() != tc.stdout || stderr.String() != tc.stderr {
			t.Errorf("run(%q) = %d, %q, %q; want %d, %q, %q", tc.args, status, stdout.String(), stderr.String(),
				tc.status, tc.stdout, tc.stderr)
		}
	}
}
