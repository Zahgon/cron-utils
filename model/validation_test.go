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
// Ported to Go from com/cronutils/model/field/expression/visitor/ValidationFieldExpressionVisitorTest.java.
//
// The original replaced the character validation with mocks (accepting everything, or rejecting everything) and
// counted visits with spies. Here the accepting validation is a StringValidations that removes every character, the
// rejecting case uses a marker the field does not support, and the visit counts are not checked.

package model

import (
	"regexp"
	"testing"

	"github.com/jmrozanec/cron-utils/cronerr"
)

const (
	lowOutOfRange  = -21
	highOutOfRange = 999
	low            = 1
	middle         = 25
	high           = 50
)

var removeEverything = regexp.MustCompile(`(?s).+`)

// acceptingVisitor validates ranges against [0, 59] with every character accepted.
func acceptingVisitor(strict bool, specialChars ...SpecialChar) *ValidationVisitor {
	b := NewFieldConstraintsBuilder().WithValidRange(0, 59)
	if strict {
		b.WithStrictRange()
	}
	for _, s := range specialChars {
		b.add(s)
	}
	return &ValidationVisitor{
		constraints: b.Build(),
		validations: &StringValidations{stringToIntKeys: removeEverything, lw: removeEverything},
	}
}

func visitors() map[string]*ValidationVisitor {
	return map[string]*ValidationVisitor{
		"strict":  acceptingVisitor(true, QuestionMarkChar),
		"default": acceptingVisitor(true, QuestionMarkChar),
	}
}

func between(from, to FieldValue) *Between { return NewBetween(from, to) }

func iv(v int) IntegerFieldValue { return IntegerFieldValue(v) }

func expectValid(t *testing.T, name string, v *ValidationVisitor, e FieldExpression) {
	t.Helper()
	got, err := e.Accept(v)
	if err != nil || got != e {
		t.Errorf("%s visitor: Accept(%s) = %v, %v; want the expression back", name, e.AsString(), got, err)
	}
}

func expectInvalid(t *testing.T, name string, v *ValidationVisitor, e FieldExpression) {
	t.Helper()
	if _, err := e.Accept(v); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("%s visitor: Accept(%s) error = %v, want an invalid-argument error", name, e.AsString(), err)
	}
}

func TestValidationVisitWithInvalidChars(t *testing.T) {
	v := NewValidationVisitor(NewFieldConstraintsBuilder().WithValidRange(0, 59).Build())
	expectInvalid(t, "real", v, NewOnSpecial(low, LW, DefaultNth))
}

func TestValidationVisitValidExpressions(t *testing.T) {
	for name, v := range visitors() {
		b := between(iv(low), iv(middle))
		for _, e := range []FieldExpression{
			AlwaysExpression(), QuestionMarkExpression(), b, NewEvery(low), NewAnd().And(b), NewOn(middle),
			between(iv(low), SpecialCharFieldValue(L)), between(SpecialCharFieldValue(L), iv(middle)),
			NewEvery(middle), NewEveryOf(between(iv(low), iv(middle)), high), NewEveryOf(NewOn(low), high),
			NewOn(low), NewOn(DefaultNth), NewOnSpecial(DefaultNth, L, DefaultNth), NewOnSpecial(low, L, high),
			NewOnSpecial(low, LW, DefaultNth), NewAnd(),
			NewAnd().And(between(iv(low), iv(middle))).And(between(iv(middle), iv(high))).
				And(between(iv(middle), iv(high))).And(NewOn(low)),
		} {
			expectValid(t, name, v, e)
		}
	}
}

func TestValidationVisitInvalidExpressions(t *testing.T) {
	for name, v := range visitors() {
		for _, e := range []FieldExpression{
			between(iv(high), iv(low)),
			between(iv(lowOutOfRange), iv(high)),
			between(iv(low), SpecialCharFieldValue(LW)),
			between(iv(low), iv(highOutOfRange)),
			between(iv(lowOutOfRange), iv(highOutOfRange)),
			NewEvery(highOutOfRange),
			NewEveryOf(between(iv(low), iv(middle)), highOutOfRange),
			NewEveryOf(NewOn(low), highOutOfRange),
			NewEveryOf(between(iv(lowOutOfRange), iv(middle)), high),
			NewEveryOf(NewOn(highOutOfRange), high),
			NewOn(lowOutOfRange),
			NewOnSpecial(low, LW, highOutOfRange),
			NewAnd().And(between(iv(low), iv(middle))).And(between(iv(middle), iv(highOutOfRange))).And(NewOn(low)),
		} {
			expectInvalid(t, name, v, e)
		}
	}
}

func TestValidationVisitBetweenOutOfOrder(t *testing.T) {
	b := between(iv(high), iv(low))
	expectInvalid(t, "strict", acceptingVisitor(true), b)
	expectValid(t, "non-strict", acceptingVisitor(false), b)
}

func TestValidationIsSpecialCharNotL(t *testing.T) {
	for _, sc := range SpecialChars {
		if got := isSpecialCharNotL(SpecialCharFieldValue(sc)); got != (sc != L) {
			t.Errorf("isSpecialCharNotL(%s) = %v, want %v", sc, got, sc != L)
		}
	}
	if isSpecialCharNotL(iv(81)) {
		t.Errorf("isSpecialCharNotL(81) = true, want false")
	}
}

func TestValidationIsInRange(t *testing.T) {
	v := acceptingVisitor(true)
	for _, value := range []FieldValue{SpecialCharFieldValue(LW), iv(5)} {
		if err := v.isInRange(value); err != nil {
			t.Errorf("isInRange(%s) = %v, want nil", value, err)
		}
	}
	if err := v.isInRange(iv(highOutOfRange)); !cronerr.Is(err, cronerr.IllegalArgument) {
		t.Errorf("isInRange(%d) = %v, want an invalid-argument error", highOutOfRange, err)
	}
}
