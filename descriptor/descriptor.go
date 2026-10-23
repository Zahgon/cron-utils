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
// Ported to Go from com/cronutils/descriptor/{CronDescriptor,TimeDescriptionStrategy}.java.

// Package descriptor describes crons in human-readable text, in every language of the bundled description texts.
package descriptor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/internal/javalang"
	"github.com/jmrozanec/cron-utils/model"
)

// DefaultLocale is the language tag of the default descriptor.
const DefaultLocale = "en-GB"

// CronDescriptor describes crons in one language.
type CronDescriptor struct {
	bundle *Bundle
}

// Instance returns the descriptor for the default locale.
func Instance() (*CronDescriptor, error) { return InstanceFor(DefaultLocale) }

// InstanceFor returns the descriptor for a BCP 47 language tag.
func InstanceFor(tag string) (*CronDescriptor, error) {
	b, err := BundleFor(tag)
	if err != nil {
		return nil, err
	}
	return &CronDescriptor{bundle: b}, nil
}

// Bundle returns the descriptor's texts.
func (d *CronDescriptor) Bundle() *Bundle { return d.bundle }

// Describe returns the description of a cron: time of day, day of month, month, day of week and year, in that
// order.
func (d *CronDescriptor) Describe(cron model.Cron) (string, error) {
	if _, ok := cron.(*model.RebootCron); ok {
		return d.bundle.String("on_reboot")
	}
	fields, err := cron.Fields()
	if err != nil {
		return "", err
	}
	exprs := map[model.CronFieldName]model.FieldExpression{}
	for _, f := range fields {
		exprs[f.Field] = f.Expression
	}
	parts := make([]string, 0, 5)
	steps := []func() (string, error){
		func() (string, error) { return d.describeTime(exprs) },
		func() (string, error) {
			return d.withUnits(dayOfMonthStrategy(d.bundle, exprs[model.DayOfMonth]).describeField, "day", "days")
		},
		func() (string, error) {
			return d.withUnits(monthStrategy(d.bundle, exprs[model.Month]).describeField, "month", "months")
		},
		func() (string, error) {
			s := dayOfWeekStrategy(d.bundle, exprs[model.DayOfWeek], cron.Definition().FieldDefinition(model.DayOfWeek))
			return d.withUnits(s.describeField, "day", "days")
		},
		func() (string, error) {
			return d.withUnits(newNominalStrategy(d.bundle, nil, exprs[model.Year]).describeField, "year", "years")
		},
	}
	for _, step := range steps {
		p, err := step()
		if err != nil {
			return "", err
		}
		parts = append(parts, p)
	}
	return javalang.Trim(javalang.CollapseSpaces(strings.Join(parts, " "))), nil
}

// withUnits describes a field and substitutes its unit names for the %s and replace_plural placeholders.
func (d *CronDescriptor) withUnits(describe func() (string, error), singularKey, pluralKey string) (string, error) {
	description, err := describe()
	if err != nil {
		return "", err
	}
	singular, err := d.bundle.String(singularKey)
	if err != nil {
		return "", err
	}
	plural, err := d.bundle.String(pluralKey)
	if err != nil {
		return "", err
	}
	return substituteUnits(description, singular, plural), nil
}

var (
	singularPlaceholder = regexp.MustCompile(`%s`)
	pluralPlaceholder   = regexp.MustCompile(`replace_plural`)
)

func substituteUnits(description, singular, plural string) string {
	description = singularPlaceholder.ReplaceAllLiteralString(description, singular)
	return pluralPlaceholder.ReplaceAllLiteralString(description, plural)
}

// timeFields are the hour, minute and second expressions of a cron.
type timeFields struct {
	hours, minutes, seconds model.FieldExpression
}

func (d *CronDescriptor) describeTime(exprs map[model.CronFieldName]model.FieldExpression) (string, error) {
	t := timeFields{hours: exprs[model.Hour], minutes: exprs[model.Minute], seconds: exprs[model.Second]}
	if t.hours == nil {
		t.hours = model.AlwaysExpression()
	}
	if t.minutes == nil {
		t.minutes = model.AlwaysExpression()
	}
	if t.seconds == nil {
		t.seconds = model.NewOn(0)
	}
	s := newStrategy(d.bundle)
	s.describeAlways = func() (string, error) { return s.describeEvery(model.NewEvery(1)) }
	ts := &timeStrategy{strategy: s, fields: t}
	return ts.describeTime()
}

type timeStrategy struct {
	*strategy
	fields timeFields
}

func isAlways(e model.FieldExpression) bool {
	_, ok := e.(*model.Always)
	return ok
}

func asOn(e model.FieldExpression) (*model.On, bool) {
	o, ok := e.(*model.On)
	return o, ok
}

func isDefaultOn(e model.FieldExpression) bool {
	o, ok := asOn(e)
	return ok && o.Time.Value() == 0
}

// texts returns the bundle texts of the given keys.
func (t *timeStrategy) texts(keys ...string) ([]any, error) {
	out := make([]any, len(keys))
	for i, k := range keys {
		v, err := t.text(k)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// format fills a format with bundle texts (string keys) and values (anything else), reading texts in order.
func (t *timeStrategy) format(format string, items ...any) (string, error) {
	args := make([]any, len(items))
	for i, item := range items {
		if key, ok := item.(textKey); ok {
			v, err := t.text(string(key))
			if err != nil {
				return "", err
			}
			args[i] = v
		} else {
			args[i] = item
		}
	}
	return fmt.Sprintf(format, args...), nil
}

type textKey string

// special returns the description of the special time-of-day shapes, or "" when none applies. The shapes do not
// overlap, so at most one applies.
func (t *timeStrategy) special() (string, error) {
	h, m, s := t.fields.hours, t.fields.minutes, t.fields.seconds
	hOn, hIsOn := asOn(h)
	mOn, mIsOn := asOn(m)
	sOn, sIsOn := asOn(s)
	switch {
	case isAlways(h) && isAlways(m):
		// every second; every minute; every minute at second x
		if isAlways(s) {
			return t.format("%s %s ", textKey("every"), textKey("second"))
		}
		if sIsOn {
			if sOn.Time.Value() == 0 {
				return t.format("%s %s ", textKey("every"), textKey("minute"))
			}
			return t.format("%s %s %s %s %02d", textKey("every"), textKey("minute"), textKey("at"), textKey("second"),
				sOn.Time.Value())
		}
	case isAlways(h) && mIsOn && sIsOn:
		// at minute x of every hour, optionally with a second
		if sOn.Time.Value() == 0 {
			if mOn.Time.Value() == 0 {
				return t.format("%s %s ", textKey("every"), textKey("hour"))
			}
			return t.format("%s %s %s %s %d", textKey("every"), textKey("hour"), textKey("at"), textKey("minute"),
				mOn.Time.Value())
		}
		return t.format("%s %s %s %s %d %s %s %d", textKey("every"), textKey("hour"), textKey("at"), textKey("minute"),
			mOn.Time.Value(), textKey("and"), textKey("second"), sOn.Time.Value())
	case hIsOn && mIsOn && isAlways(s):
		return t.format("%s %s %s %02d:%02d", textKey("every"), textKey("second"), textKey("at"), hOn.Time.Value(),
			mOn.Time.Value())
	case hIsOn && mIsOn && sIsOn:
		if sOn.Time.Value() == 0 {
			return t.format("%s %02d:%02d", textKey("at"), hOn.Time.Value(), mOn.Time.Value())
		}
		return t.format("%s %02d:%02d:%02d", textKey("at"), hOn.Time.Value(), mOn.Time.Value(), sOn.Time.Value())
	case hIsOn && isAlways(m) && isAlways(s):
		return t.format("%s %02d:00", textKey("at"), hOn.Time.Value())
	case hIsOn:
		b, ok := m.(*model.Between)
		if !ok || !(sIsOn || isAlways(s)) {
			return "", nil
		}
		unit := textKey("minute")
		if isAlways(s) {
			unit = textKey("second")
		}
		from, err := zeroPadded(b.From)
		if err != nil {
			return "", err
		}
		to, err := zeroPadded(b.To)
		if err != nil {
			return "", err
		}
		return t.format("%s %s %s %02d:%s %s %02d:%s", textKey("every"), unit, textKey("between"), hOn.Time.Value(),
			from, textKey("and"), hOn.Time.Value(), to)
	case isAlways(h) && sIsOn:
		every, ok := m.(*model.Every)
		if !ok {
			return "", nil
		}
		var desc string
		var err error
		if every.Period.Value() == 1 && sOn.Time.Value() == 0 {
			desc, err = t.format("%s %s ", textKey("every"), textKey("minute"))
		} else {
			desc, err = t.format("%s %d %s ", textKey("every"), every.Period.Value(), textKey("minutes"))
		}
		if err != nil {
			return "", err
		}
		if _, between := every.Expression.(*model.Between); between {
			return "", nil
		}
		return desc, nil
	}
	if every, ok := h.(*model.Every); ok && mIsOn && sIsOn {
		// every x hours at minute y
		if mOn.Time.Value() == 0 && sOn.Time.Value() == 0 && every.Period.Value() == 1 {
			return t.format("%s %s ", textKey("every"), textKey("hour"))
		}
		result, err := t.format("%s %d %s %s %s %d ", textKey("every"), every.Period.Value(), textKey("hours"),
			textKey("at"), textKey("minute"), mOn.Time.Value())
		if err != nil {
			return "", err
		}
		if sOn.Time.Value() == 0 {
			return result, nil
		}
		return t.format("%s %s %d", textKey("and"), textKey("second"), sOn.Time.Value())
	}
	if isAlways(h) && isAlways(s) {
		if mIsOn {
			return t.format("%s %s %s %s %02d %s", textKey("every"), textKey("second"), textKey("at"), textKey("minute"),
				mOn.Time.Value(), textKey("of_every_hour"))
		}
		if _, ok := m.(*model.And); ok {
			d, err := t.describe(m, false)
			if err != nil {
				return "", err
			}
			units, err := t.texts("minute", "minutes")
			if err != nil {
				return "", err
			}
			minutes := substituteUnits(d, units[0].(string), units[1].(string))
			return t.format("%s %s %s", textKey("every"), textKey("second"), minutes)
		}
	}
	if _, ok := h.(*model.And); ok && isAlways(m) && isAlways(s) {
		d, err := t.describe(h, false)
		if err != nil {
			return "", err
		}
		units, err := t.texts("hour", "hours")
		if err != nil {
			return "", err
		}
		hours := substituteUnits(d, units[0].(string), units[1].(string))
		return t.format("%s %s %s", textKey("every"), textKey("second"), hours)
	}
	return "", nil
}

// zeroPadded formats a range bound as a two-digit number.
func zeroPadded(v model.FieldValue) (string, error) {
	i, ok := v.(model.IntegerFieldValue)
	if !ok {
		return "", cronerr.Argument("d != com.cronutils.model.field.value.SpecialChar")
	}
	return fmt.Sprintf("%02d", int(i)), nil
}

func (t *timeStrategy) describeTime() (string, error) {
	d, err := t.special()
	if err != nil || d != "" {
		return d, err
	}
	h, m, s := t.fields.hours, t.fields.minutes, t.fields.seconds
	var secondsDesc, minutesDesc, hoursDesc string
	if !isAlways(h) {
		if hoursDesc, err = t.unitDescription(h, "hour", "hours"); err != nil {
			return "", err
		}
	}
	if !isDefaultOn(m) && !(isAlways(m) && isAlways(h)) {
		if minutesDesc, err = t.unitDescription(m, "minute", "minutes"); err != nil {
			return "", err
		}
	}
	if !isDefaultOn(s) {
		if secondsDesc, err = t.unitDescription(s, "second", "seconds"); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s %s %s", secondsDesc, minutesDesc, hoursDesc), nil
}

func (t *timeStrategy) unitDescription(e model.FieldExpression, singularKey, pluralKey string) (string, error) {
	d, err := t.describe(e, false)
	if err != nil {
		return "", err
	}
	units, err := t.texts(singularKey, pluralKey)
	if err != nil {
		return "", err
	}
	return substituteUnits(d, units[0].(string), units[1].(string)), nil
}
