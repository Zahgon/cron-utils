// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package descriptor

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jmrozanec/cron-utils/cronerr"
)

//go:embed i18n/*.properties
var bundleFiles embed.FS

// Bundle holds the description texts of one language, falling back to the base texts for keys it lacks.
type Bundle struct {
	language string
	texts    map[string]string
	parent   *Bundle
}

// Language returns the bundle's language code; empty for the base (root) texts.
func (b *Bundle) Language() string { return b.language }

// String returns the text of key, or an error when neither the bundle nor its parents define it.
func (b *Bundle) String(key string) (string, error) {
	for c := b; c != nil; c = c.parent {
		if v, ok := c.texts[key]; ok {
			return v, nil
		}
	}
	return "", cronerr.New(cronerr.MissingResource,
		"Can't find resource for bundle java.util.PropertyResourceBundle, key %s", key)
}

var (
	loadOnce sync.Once
	bundles  map[string]*Bundle
	loadErr  error
)

func loadBundles() (map[string]*Bundle, error) {
	loadOnce.Do(func() {
		entries, err := bundleFiles.ReadDir("i18n")
		if err != nil {
			loadErr = err
			return
		}
		bundles = map[string]*Bundle{}
		for _, e := range entries {
			data, err := bundleFiles.ReadFile("i18n/" + e.Name())
			if err != nil {
				loadErr = err
				return
			}
			language := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "CronUtilsI18N"), ".properties")
			language = strings.TrimPrefix(language, "_")
			bundles[language] = &Bundle{language: language, texts: parseProperties(data)}
		}
		root := bundles[""]
		if root == nil {
			loadErr = fmt.Errorf("descriptor: base bundle missing")
			return
		}
		for language, b := range bundles {
			if language != "" {
				b.parent = root
			}
		}
	})
	return bundles, loadErr
}

// BundleFor returns the texts for a BCP 47 language tag. A language without texts uses English; the tag "und" (no
// language, region or script) uses the base texts.
func BundleFor(tag string) (*Bundle, error) {
	all, err := loadBundles()
	if err != nil {
		return nil, err
	}
	l := parseLanguageTag(tag)
	if l.language == "" {
		if l.script == "" && l.region == "" && l.variant == "" {
			return all[""], nil
		}
		return all[defaultLanguage], nil
	}
	if b, ok := all[l.language]; ok {
		return b, nil
	}
	return all[defaultLanguage], nil
}

// defaultLanguage is the language used when a tag names one without texts.
const defaultLanguage = "en"

// parseProperties reads a .properties file: key=value or key:value lines, # and ! comments, backslash escapes
// (\uXXXX, \t, \n, \r, \f) and backslash line continuations.
func parseProperties(data []byte) map[string]string {
	out := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimLeft(lines[i], " \t\f")
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		for endsWithOddBackslashes(line) && i+1 < len(lines) {
			i++
			line = line[:len(line)-1] + strings.TrimLeft(lines[i], " \t\f")
		}
		key, value := splitProperty(line)
		out[unescape(key)] = unescape(value)
	}
	return out
}

func endsWithOddBackslashes(s string) bool {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

func splitProperty(line string) (string, string) {
	i := 0
	for i < len(line) {
		c := line[i]
		if c == '\\' {
			i += 2
			continue
		}
		if c == '=' || c == ':' || c == ' ' || c == '\t' || c == '\f' {
			break
		}
		i++
	}
	key := line[:min(i, len(line))]
	rest := line[min(i, len(line)):]
	rest = strings.TrimLeft(rest, " \t\f")
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], " \t\f")
	}
	return key, rest
}

func unescape(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 'f':
			b.WriteByte('\f')
		case 'u':
			if i+5 <= len(s) {
				if v, err := strconv.ParseUint(s[i+1:i+5], 16, 32); err == nil {
					r := rune(v)
					if !utf8.ValidRune(r) {
						r = utf8.RuneError
					}
					b.WriteRune(r)
					i += 4
					continue
				}
			}
			b.WriteByte('u')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// languageTag is the part of a BCP 47 language tag that selects description texts.
type languageTag struct {
	language, script, region, variant string
}

var grandfathered = map[string]string{
	"art-lojban": "jbo", "cel-gaulish": "xtg", "en-gb-oed": "en", "i-ami": "ami", "i-bnn": "bnn",
	"i-default": "en", "i-enochian": "", "i-hak": "hak", "i-klingon": "tlh", "i-lux": "lb", "i-mingo": "",
	"i-navajo": "nv", "i-pwn": "pwn", "i-tao": "tao", "i-tay": "tay", "i-tsu": "tsu", "no-bok": "nb",
	"no-nyn": "nn", "sgn-be-fr": "sfb", "sgn-be-nl": "vgt", "sgn-ch-de": "sgg", "zh-guoyu": "cmn",
	"zh-hakka": "hak", "zh-min": "nan", "zh-min-nan": "nan", "zh-xiang": "hsn",
}

var legacyLanguages = map[string]string{"iw": "he", "ji": "yi", "in": "id"}

func isAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return s != ""
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func isAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if (c < 'a' || c > 'z') && (s[i] < '0' || s[i] > '9') {
			return false
		}
	}
	return s != ""
}

// parseLanguageTag reads a BCP 47 language tag the lenient way: parsing stops at the first ill-formed subtag.
func parseLanguageTag(tag string) languageTag {
	if g, ok := grandfathered[strings.ToLower(tag)]; ok {
		return languageTag{language: g}
	}
	subtags := strings.Split(tag, "-")
	var l languageTag
	i := 0
	if i < len(subtags) && len(subtags[i]) >= 2 && len(subtags[i]) <= 8 && isAlpha(subtags[i]) {
		l.language = strings.ToLower(subtags[i])
		i++
		if len(l.language) <= 3 {
			for n := 0; n < 3 && i < len(subtags) && len(subtags[i]) == 3 && isAlpha(subtags[i]); n++ {
				if n == 0 {
					l.language = strings.ToLower(subtags[i])
				}
				i++
			}
		}
	} else {
		return l
	}
	if i < len(subtags) && len(subtags[i]) == 4 && isAlpha(subtags[i]) {
		l.script = subtags[i]
		i++
	}
	if i < len(subtags) && ((len(subtags[i]) == 2 && isAlpha(subtags[i])) || (len(subtags[i]) == 3 && isDigits(subtags[i]))) {
		l.region = subtags[i]
		i++
	}
	if i < len(subtags) && isAlnum(subtags[i]) &&
		((len(subtags[i]) >= 5 && len(subtags[i]) <= 8) || (len(subtags[i]) == 4 && subtags[i][0] >= '0' && subtags[i][0] <= '9')) {
		l.variant = subtags[i]
	}
	if l.language == "und" {
		l.language = ""
	}
	if m, ok := legacyLanguages[l.language]; ok {
		l.language = m
	}
	return l
}

// messageFormat substitutes {n} placeholders. Text between single quotes is literal (a quote that is never closed
// quotes the rest of the pattern) and ” stands for one quote.
func messageFormat(pattern string, args ...any) string {
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}
			inQuote = !inQuote
			continue
		}
		if c == '{' && !inQuote {
			end := strings.IndexByte(pattern[i:], '}')
			if end < 0 {
				b.WriteString(pattern[i:])
				break
			}
			index, err := strconv.Atoi(strings.TrimSpace(pattern[i+1 : i+end]))
			if err == nil && index >= 0 && index < len(args) {
				b.WriteString(formatArgument(args[index]))
			} else {
				b.WriteString(pattern[i : i+end+1])
			}
			i += end
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// formatArgument formats a placeholder value: integers with thousands separators, anything else as text.
func formatArgument(arg any) string {
	n, ok := arg.(int)
	if !ok {
		return fmt.Sprint(arg)
	}
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
