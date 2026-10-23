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
// Ported to Go from com/cronutils/descriptor/{DescriptionStrategy,NominalDescriptionStrategy,
// DescriptionStrategyFactory}.java.

package descriptor

import (
	"fmt"
	"strconv"

	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

// strategy builds the description of field expressions. Descriptions contain the placeholders %s (the field's unit,
// singular) and replace_plural (the unit, plural) that the caller substitutes.
type strategy struct {
	bundle  *Bundle
	nominal func(int) (string, error)
	// describeAlways replaces the description of "*" when set.
	describeAlways func() (string, error)
}

func newStrategy(bundle *Bundle) *strategy {
	return &strategy{bundle: bundle, nominal: func(v int) (string, error) { return " " + strconv.Itoa(v), nil }}
}

func (s *strategy) text(key string) (string, error) { return s.bundle.String(key) }

func (s *strategy) describe(e model.FieldExpression, and bool) (string, error) {
	switch x := e.(type) {
	case *model.Always:
		if s.describeAlways != nil {
			return s.describeAlways()
		}
		return "", nil
	case *model.And:
		return s.describeAnd(x)
	case *model.Between:
		return s.describeBetween(x)
	case *model.Every:
		return s.describeEvery(x)
	case *model.On:
		return s.describeOn(x, and)
	}
	return "", nil
}

func (s *strategy) describeAnd(a *model.And) (string, error) {
	var expressions, ons []model.FieldExpression
	for _, e := range a.Expressions() {
		if _, ok := e.(*model.On); ok {
			ons = append(ons, e)
		} else {
			expressions = append(expressions, e)
		}
	}
	var out string
	if len(ons) > 0 {
		at, err := s.text("at")
		if err != nil {
			return "", err
		}
		out = at
		if len(ons) == 1 {
			out += " %s "
		}
		list, err := s.andList(ons)
		if err != nil {
			return "", err
		}
		out += list
		if len(ons) > 1 {
			out += " replace_plural "
		}
	}
	if len(expressions) > 0 {
		if len(ons) > 0 {
			and, err := s.text("and")
			if err != nil {
				return "", err
			}
			out += fmt.Sprintf(" %s ", and)
		}
		list, err := s.andList(expressions)
		if err != nil {
			return "", err
		}
		out += list
	}
	return out, nil
}

// andList describes a list as " a,  b  and  c ".
func (s *strategy) andList(expressions []model.FieldExpression) (string, error) {
	var out string
	n := len(expressions)
	if n > 1 {
		for j := 0; j < n-2; j++ {
			d, err := s.describe(expressions[j], true)
			if err != nil {
				return "", err
			}
			out += fmt.Sprintf(" %s, ", d)
		}
		d, err := s.describe(expressions[n-2], true)
		if err != nil {
			return "", err
		}
		out += fmt.Sprintf(" %s ", d)
		and, err := s.text("and")
		if err != nil {
			return "", err
		}
		out += fmt.Sprintf(" %s ", and)
	}
	d, err := s.describe(expressions[n-1], true)
	if err != nil {
		return "", err
	}
	return out + fmt.Sprintf(" %s ", d), nil
}

func (s *strategy) betweenText(b *model.Between) (string, error) {
	pattern, err := s.text("between_x_and_y")
	if err != nil {
		return "", err
	}
	from, err := s.nominalValue(b.From)
	if err != nil {
		return "", err
	}
	to, err := s.nominalValue(b.To)
	if err != nil {
		return "", err
	}
	return messageFormat(pattern, from, to), nil
}

func (s *strategy) describeBetween(b *model.Between) (string, error) {
	every, err := s.text("every")
	if err != nil {
		return "", err
	}
	between, err := s.betweenText(b)
	if err != nil {
		return "", err
	}
	return every + " %s " + between + " ", nil
}

func (s *strategy) describeEvery(e *model.Every) (string, error) {
	every, err := s.text("every")
	if err != nil {
		return "", err
	}
	var out string
	if e.Period.Value() > 1 {
		out = fmt.Sprintf("%s %s ", every, e.Period) + " replace_plural "
	} else {
		out = every + " %s "
	}
	switch inner := e.Expression.(type) {
	case *model.Between:
		between, err := s.betweenText(inner)
		if err != nil {
			return "", err
		}
		out += between + " "
	case *model.On:
		from, err := s.text("from")
		if err != nil {
			return "", err
		}
		value, err := s.nominalValue(inner.Time)
		if err != nil {
			return "", err
		}
		out += from + " %s " + value
	}
	return out, nil
}

func (s *strategy) describeOn(o *model.On, and bool) (string, error) {
	value, err := s.nominalValue(o.Time)
	if err != nil {
		return "", err
	}
	if and {
		return value, nil
	}
	at, err := s.text("at")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s ", at, value) + "%s", nil
}

func (s *strategy) nominalValue(v model.FieldValue) (string, error) {
	if i, ok := v.(model.IntegerFieldValue); ok {
		return s.nominal(int(i))
	}
	return v.String(), nil
}

// nominalStrategy describes a date field: a field-specific description first, then the generic one.
type nominalStrategy struct {
	*strategy
	expression model.FieldExpression
	special    func(model.FieldExpression) (string, error)
}

func newNominalStrategy(bundle *Bundle, nominal func(int) (string, error), expression model.FieldExpression) *nominalStrategy {
	s := newStrategy(bundle)
	if nominal != nil {
		s.nominal = nominal
	}
	if expression == nil {
		expression = model.AlwaysExpression()
	}
	return &nominalStrategy{strategy: s, expression: expression}
}

func (n *nominalStrategy) describeField() (string, error) {
	if n.special != nil {
		d, err := n.special(n.expression)
		if err != nil {
			return "", err
		}
		if d != "" {
			return d, nil
		}
	}
	return n.describe(n.expression, false)
}

func dayOfWeekStrategy(bundle *Bundle, expression model.FieldExpression, definition *model.FieldDefinition) *nominalStrategy {
	names := namesFor(bundle)
	nominal := func(v int) (string, error) {
		diff := 0
		if definition != nil && definition.MondayDoW != nil {
			diff = 1 - definition.MondayDoW.MondayDoWValue()
		}
		day := v + diff
		if day < 1 {
			day = 7
		}
		if err := javatime.CheckDayOfWeek(day); err != nil {
			return "", err
		}
		return names.days[day-1], nil
	}
	s := newNominalStrategy(bundle, nominal, expression)
	s.special = func(e model.FieldExpression) (string, error) {
		on, ok := e.(*model.On)
		if !ok {
			return "", nil
		}
		switch on.SpecialChar.Value() {
		case model.Hash:
			day, err := nominal(on.Time.Value())
			if err != nil {
				return "", err
			}
			ofEveryMonth, err := bundle.String("of_every_month")
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %s %s ", day, on.Nth, ofEveryMonth), nil
		case model.L:
			last, err := bundle.String("last")
			if err != nil {
				return "", err
			}
			day, err := nominal(on.Time.Value())
			if err != nil {
				return "", err
			}
			ofEveryMonth, err := bundle.String("of_every_month")
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %s %s ", last, day, ofEveryMonth), nil
		}
		return "", nil
	}
	return s
}

func dayOfMonthStrategy(bundle *Bundle, expression model.FieldExpression) *nominalStrategy {
	s := newNominalStrategy(bundle, nil, expression)
	s.special = func(e model.FieldExpression) (string, error) {
		on, ok := e.(*model.On)
		if !ok {
			return "", nil
		}
		switch on.SpecialChar.Value() {
		case model.W:
			nearest, err := bundle.String("the_nearest_weekday_to_the")
			if err != nil {
				return "", err
			}
			ofTheMonth, err := bundle.String("of_the_month")
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %d %s ", nearest, on.Time.Value(), ofTheMonth), nil
		case model.L:
			switch daysBefore := on.Nth.Value(); {
			case daysBefore > 1:
				pattern, err := bundle.String("days_before_last_day_of_month")
				if err != nil {
					return "", err
				}
				return messageFormat(pattern, daysBefore), nil
			case daysBefore == 1:
				return bundle.String("day_before_last_day_of_month")
			default:
				return bundle.String("last_day_of_month")
			}
		case model.LW:
			return bundle.String("last_weekday_of_month")
		}
		return "", nil
	}
	return s
}

func monthStrategy(bundle *Bundle, expression model.FieldExpression) *nominalStrategy {
	var nominal func(int) (string, error)
	if _, ok := expression.(*model.Every); ok {
		nominal = func(v int) (string, error) { return strconv.Itoa(v), nil }
	} else {
		names := namesFor(bundle)
		nominal = func(v int) (string, error) {
			if err := javatime.CheckMonth(v); err != nil {
				return "", err
			}
			return names.months[v-1], nil
		}
	}
	return newNominalStrategy(bundle, nominal, expression)
}

func namesFor(bundle *Bundle) displayNames {
	if n, ok := localeNames[bundle.Language()]; ok {
		return n
	}
	return localeNames[defaultLanguage]
}
