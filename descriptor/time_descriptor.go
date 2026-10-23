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
// Ported to Go from com/cronutils/descriptor/refactor/TimeDescriptor.java.

package descriptor

import (
	"strings"

	"github.com/jmrozanec/cron-utils/model"
)

// TimeDescriptor describes the seconds field of a cron with the bundle's pattern texts (pattern_every_seconds and
// pattern_at_second); it is the start of a pattern-based descriptor.
type TimeDescriptor struct {
	bundle *Bundle
}

// NewTimeDescriptor returns a seconds describer for a bundle.
func NewTimeDescriptor(bundle *Bundle) *TimeDescriptor { return &TimeDescriptor{bundle: bundle} }

// Describe describes the seconds field: every second, every n seconds or at second n; "" when the cron has no seconds
// field or the field has another shape.
func (d *TimeDescriptor) Describe(cron model.Cron) (string, error) {
	f, err := cron.Retrieve(model.Second)
	if err != nil || f == nil {
		return "", err
	}
	switch e := f.Expression.(type) {
	case *model.Always:
		return d.everySecond(1)
	case *model.On:
		return d.atSecond(e.Time.Value())
	case *model.Every:
		return d.everySecond(e.Period.Value())
	}
	return "", nil
}

// choice picks the text of the largest limit at or below n (the first text below every limit).
func choice(n int, limits []int, texts []string) string {
	chosen := texts[0]
	for i, limit := range limits {
		if n >= limit {
			chosen = texts[i]
		}
	}
	return chosen
}

func (d *TimeDescriptor) texts(keys ...string) ([]string, error) {
	out := make([]string, len(keys))
	for i, k := range keys {
		v, err := d.bundle.String(k)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// everySecond fills pattern_every_seconds: {0} is every_one or every_multi, {1} is oneSecond or multipleSeconds (a
// pattern itself, in which {2} is the number).
func (d *TimeDescriptor) everySecond(n int) (string, error) {
	t, err := d.texts("oneSecond", "multipleSeconds", "every_one", "every_multi", "pattern_every_seconds")
	if err != nil {
		return "", err
	}
	every := choice(n, []int{1, 2}, []string{t[2], t[3]})
	seconds := choice(n, []int{1, 2}, []string{t[0], t[1]})
	if strings.Contains(every, "{") {
		every = messageFormat(every, n, n, n)
	}
	if strings.Contains(seconds, "{") {
		seconds = messageFormat(seconds, n, n, n)
	}
	return messageFormat(t[4], every, seconds, n), nil
}

func (d *TimeDescriptor) atSecond(n int) (string, error) {
	pattern, err := d.bundle.String("pattern_at_second")
	if err != nil {
		return "", err
	}
	return messageFormat(pattern, n), nil
}
