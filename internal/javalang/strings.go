// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package javalang holds the text rules the cron-utils packages share: how expressions are split into tokens,
// trimmed, upper-cased and read as numbers. The rules follow the Java platform conventions the library was defined
// with (Unicode full case mapping, ASCII-only regular expression whitespace, decimal digits of every script).
package javalang

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// Split splits s around each occurrence of the literal separator sep. Trailing empty strings are dropped; a string
// that does not contain sep is returned as the only element, even when it is empty.
func Split(s, sep string) []string {
	if !strings.Contains(s, sep) {
		return []string{s}
	}
	parts := strings.Split(s, sep)
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// SplitKeepTrailing splits s around each occurrence of sep and keeps trailing empty strings.
func SplitKeepTrailing(s, sep string) []string {
	return strings.Split(s, sep)
}

// IsRegexSpace reports whether b is one of the six characters of the regular-expression class \s: space, tab, line
// feed, vertical tab, form feed and carriage return.
func IsRegexSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// SplitSpaces splits s around runs of \s characters. A leading run yields a leading empty string; trailing empty
// strings are dropped.
func SplitSpaces(s string) []string {
	var parts []string
	start := 0
	matched := false
	for i := 0; i < len(s); {
		if !IsRegexSpace(s[i]) {
			i++
			continue
		}
		j := i
		for j < len(s) && IsRegexSpace(s[j]) {
			j++
		}
		parts = append(parts, s[start:i])
		start, i, matched = j, j, true
	}
	if !matched {
		return []string{s}
	}
	parts = append(parts, s[start:])
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// CollapseSpaces replaces every run of \s characters with a single space.
func CollapseSpaces(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if IsRegexSpace(s[i]) {
			b.WriteByte(' ')
			for i < len(s) && IsRegexSpace(s[i]) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// RemoveSpaces deletes every \s character.
func RemoveSpaces(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if !IsRegexSpace(s[i]) {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// Trim removes the leading and trailing characters at or below U+0020 (spaces and control characters).
func Trim(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] <= ' ' {
		start++
	}
	for end > start && s[end-1] <= ' ' {
		end--
	}
	return s[start:end]
}

// ToUpper upper-cases s with the full Unicode case mapping, in which some characters (ß, ligatures, some Greek
// letters) turn into two or three characters.
func ToUpper(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if special, ok := upperSpecial[r]; ok {
			b.WriteString(special)
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// digitZeros are the code points of the digit zero of every decimal digit set in the Basic Multilingual Plane.
var digitZeros = []rune{
	0x0030, 0x0660, 0x06F0, 0x07C0, 0x0966, 0x09E6, 0x0A66, 0x0AE6, 0x0B66, 0x0BE6, 0x0C66, 0x0CE6, 0x0D66,
	0x0DE6, 0x0E50, 0x0ED0, 0x0F20, 0x1040, 0x1090, 0x17E0, 0x1810, 0x1946, 0x19D0, 0x1A80, 0x1A90, 0x1B50,
	0x1BB0, 0x1C40, 0x1C50, 0xA620, 0xA8D0, 0xA900, 0xA9D0, 0xA9F0, 0xAA50, 0xABF0, 0xFF10,
}

// Digit returns the decimal value of r, or -1 when r is not a decimal digit of the Basic Multilingual Plane.
func Digit(r rune) int {
	for _, zero := range digitZeros {
		if r >= zero && r < zero+10 {
			return int(r - zero)
		}
	}
	return -1
}

// IsDigit reports whether r is a decimal digit of the Basic Multilingual Plane.
func IsDigit(r rune) bool {
	return Digit(r) >= 0
}

// IsNumeric reports whether s is non-empty and made of decimal digits only.
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !IsDigit(r) {
			return false
		}
	}
	return true
}

// NumberFormatError returns the error ParseInt reports for s.
func NumberFormatError(s string) *cronerr.Error {
	return cronerr.Argument("For input string: \"%s\"", s)
}

// ParseInt reads s as a signed 32-bit decimal integer. Digits of any script are accepted; an optional leading sign,
// nothing else. The error message quotes the whole input.
func ParseInt(s string) (int, error) {
	if s == "" {
		return 0, NumberFormatError(s)
	}
	const minInt32, maxInt32 = -1 << 31, 1<<31 - 1
	negative := false
	i := 0
	limit := -maxInt32
	if s[0] < '0' {
		switch s[0] {
		case '-':
			negative = true
			limit = minInt32
		case '+':
		default:
			return 0, NumberFormatError(s)
		}
		if len(s) == 1 {
			return 0, NumberFormatError(s)
		}
		i++
	}
	multmin := limit / 10
	result := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		digit := -1
		if r != utf8.RuneError && r <= 0xFFFF {
			digit = Digit(r)
		}
		if digit < 0 || result < multmin {
			return 0, NumberFormatError(s)
		}
		result *= 10
		if result < limit+digit {
			return 0, NumberFormatError(s)
		}
		result -= digit
	}
	if negative {
		return result, nil
	}
	return -result, nil
}

// ContainsAny reports whether s contains any of the given ASCII characters.
func ContainsAny(s string, chars ...byte) bool {
	for i := 0; i < len(s); i++ {
		for _, c := range chars {
			if s[i] == c {
				return true
			}
		}
	}
	return false
}

// Itoa formats an integer the way string concatenation does.
func Itoa(n int) string {
	return strconv.Itoa(n)
}
