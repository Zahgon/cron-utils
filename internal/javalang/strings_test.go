// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package javalang

import (
	"slices"
	"testing"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		in, sep string
		want    []string
	}{
		{"1,2,", ",", []string{"1", "2"}},
		{",", ",", []string{}},
		{"", ",", []string{""}},
		{",5", ",", []string{"", "5"}},
		{"1,,2", ",", []string{"1", "", "2"}},
	}
	for _, tc := range tests {
		if got := Split(tc.in, tc.sep); !slices.Equal(got, tc.want) {
			t.Errorf("Split(%q, %q) = %q, want %q", tc.in, tc.sep, got, tc.want)
		}
	}
}

func TestSplitSpaces(t *testing.T) {
	if got, want := SplitSpaces(" a\v b\t"), []string{"", "a", "b"}; !slices.Equal(got, want) {
		t.Errorf("SplitSpaces(%q) = %q, want %q", " a\v b\t", got, want)
	}
	if got, want := SplitSpaces("a b"), []string{"a b"}; !slices.Equal(got, want) {
		t.Errorf("SplitSpaces(%q) = %q, want %q", "a\u00a0b", got, want)
	}
}

func TestParseInt(t *testing.T) {
	ok := map[string]int{"12": 12, "+5": 5, "-7": -7, "０９": 9, "١٢": 12, "2147483647": 2147483647}
	for in, want := range ok {
		if got, err := ParseInt(in); err != nil || got != want {
			t.Errorf("ParseInt(%q) = %d, %v, want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "+", "-", "1a", " 1", "2147483648", "\U0001D7CE"} {
		_, err := ParseInt(in)
		if err == nil || err.Error() != `For input string: "`+in+`"` {
			t.Errorf("ParseInt(%q) error = %v, want %q", in, err, `For input string: "`+in+`"`)
		}
	}
}

func TestToUpperAndTrim(t *testing.T) {
	if got := ToUpper("straße ﬁ mon"); got != "STRASSE FI MON" {
		t.Errorf("ToUpper(%q) = %q, want %q", "straße ﬁ mon", got, "STRASSE FI MON")
	}
	if got := Trim("\u001f a b \t"); got != "a b" {
		t.Errorf("Trim(%q) = %q, want %q", "\u001f a b \t", got, "a b")
	}
	if got := CollapseSpaces("a \t\v b"); got != "a b" {
		t.Errorf("CollapseSpaces(%q) = %q, want %q", "a \t\v b", got, "a b")
	}
}
