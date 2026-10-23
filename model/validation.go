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
// Ported to Go from com/cronutils/model/field/expression/visitor/ValidationFieldExpressionVisitor.java and
// com/cronutils/StringValidations.java.

package model

import (
	"regexp"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/internal/javalang"
)

var numsAndChars = regexp.MustCompile(`[#?/*0-9]`)

// StringValidations finds the characters of an expression a field does not accept.
type StringValidations struct {
	stringToIntKeys *regexp.Regexp
	lw              *regexp.Regexp
}

// NewStringValidations returns the validations of a field.
func NewStringValidations(constraints *FieldConstraints) *StringValidations {
	var lwWords []string
	for _, sc := range []SpecialChar{L, LW, W} {
		if constraints.SupportsSpecialChar(sc) {
			lwWords = append(lwWords, sc.String())
		}
	}
	return &StringValidations{
		stringToIntKeys: wordsPattern(constraints.StringMappingKeys()),
		lw:              wordsPattern(lwWords),
	}
}

func wordsPattern(words []string) *regexp.Regexp {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`\b(` + strings.Join(quoted, "|") + `)\b`)
}

// RemoveValidChars returns what is left of exp, upper-cased, once digits, the characters # ? / * , - and white
// space, the field's names and its supported letter markers are removed.
func (s *StringValidations) RemoveValidChars(exp string) string {
	rest := numsAndChars.ReplaceAllString(javalang.ToUpper(exp), "")
	rest = s.stringToIntKeys.ReplaceAllString(rest, "")
	rest = s.lw.ReplaceAllString(rest, "")
	rest = javalang.RemoveSpaces(rest)
	rest = strings.ReplaceAll(rest, ",", "")
	return strings.ReplaceAll(rest, "-", "")
}

// ValidationVisitor checks an expression against a field's constraints.
type ValidationVisitor struct {
	constraints *FieldConstraints
	validations *StringValidations
}

// NewValidationVisitor returns a validator for a field.
func NewValidationVisitor(constraints *FieldConstraints) *ValidationVisitor {
	return &ValidationVisitor{constraints: constraints, validations: NewStringValidations(constraints)}
}

func (v *ValidationVisitor) checkUnsupportedChars(e FieldExpression) error {
	unsupported := v.validations.RemoveValidChars(e.AsString())
	if unsupported == "" {
		return nil
	}
	return cronerr.Argument("Invalid chars in expression! Expression: %s Invalid chars: %s", e.AsString(), unsupported)
}

// VisitAlways accepts "*".
func (v *ValidationVisitor) VisitAlways(a *Always) (FieldExpression, error) {
	return a, v.checkUnsupportedChars(a)
}

// VisitAnd validates every listed expression.
func (v *ValidationVisitor) VisitAnd(a *And) (FieldExpression, error) {
	if err := v.checkUnsupportedChars(a); err != nil {
		return nil, err
	}
	for _, e := range a.expressions {
		if _, err := e.Accept(v); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// VisitBetween validates both ends of a range.
func (v *ValidationVisitor) VisitBetween(b *Between) (FieldExpression, error) {
	if err := v.checkUnsupportedChars(b); err != nil {
		return nil, err
	}
	if err := v.isInRange(b.From); err != nil {
		return nil, err
	}
	if err := v.isInRange(b.To); err != nil {
		return nil, err
	}
	if isSpecialCharNotL(b.From) || isSpecialCharNotL(b.To) {
		return nil, cronerr.Argument("No special characters allowed in range, except for 'L'")
	}
	from, fromInt := b.From.(IntegerFieldValue)
	to, toInt := b.To.(IntegerFieldValue)
	if v.constraints.IsStrictRange() && fromInt && toInt && from > to {
		return nil, cronerr.Argument("Invalid range! [%d,%d]", from, to)
	}
	return b, nil
}

// VisitEvery validates the stepped expression and the period.
func (v *ValidationVisitor) VisitEvery(e *Every) (FieldExpression, error) {
	if err := v.checkUnsupportedChars(e); err != nil {
		return nil, err
	}
	if e.Expression != nil {
		if _, err := e.Expression.Accept(v); err != nil {
			return nil, err
		}
	}
	if !v.constraints.IsPeriodInRange(e.Period.Value()) {
		return nil, cronerr.Argument("Period %d not in range [%d, %d]", e.Period.Value(), v.constraints.StartRange(),
			v.constraints.EndRange())
	}
	return e, nil
}

// VisitOn validates the value and the nth value.
func (v *ValidationVisitor) VisitOn(o *On) (FieldExpression, error) {
	if err := v.checkUnsupportedChars(o); err != nil {
		return nil, err
	}
	if o.Time != DefaultNth {
		if err := v.isInRange(o.Time); err != nil {
			return nil, err
		}
	}
	if o.Nth != DefaultNth {
		if err := v.isInRange(o.Nth); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// VisitQuestionMark accepts "?" when the field supports it.
func (v *ValidationVisitor) VisitQuestionMark(q *QuestionMark) (FieldExpression, error) {
	if !v.constraints.SupportsSpecialChar(QuestionMarkChar) {
		return nil, cronerr.Argument("Question mark (?) is not supported!")
	}
	return q, nil
}

// VisitRandom validates the bounds and the step of a random value.
func (v *ValidationVisitor) VisitRandom(r *RandomExpression) (FieldExpression, error) {
	for _, p := range []*int{r.From, r.To, r.Step} {
		if p != nil {
			if err := v.isInRange(IntegerFieldValue(*p)); err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (v *ValidationVisitor) isInRange(value FieldValue) error {
	if i, ok := value.(IntegerFieldValue); ok && !v.constraints.IsInRange(int(i)) {
		return cronerr.Argument("Value %d not in range [%d, %d]", int(i), v.constraints.StartRange(), v.constraints.EndRange())
	}
	return nil
}

func isSpecialCharNotL(value FieldValue) bool {
	s, ok := value.(SpecialCharFieldValue)
	return ok && s.Value() != L
}
