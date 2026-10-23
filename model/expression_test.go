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
// Ported to Go from com/cronutils/model/field/expression/{AlwaysTest,AndTest,BetweenTest,EveryTest,
// FieldExpressionTest,OnTest,RandomExpressionTest}.java and
// com/cronutils/model/field/expression/visitor/ValueMappingFieldExpressionVisitorTest.java.
//
// Not ported: the null-argument cases (Go values cannot be null) and RandomExpressionTest's random-value cases (the
// Go port has no random-value generator: randomised fields only keep their text form, which is tested here).

package model_test

import (
	"strconv"
	"testing"

	"github.com/jmrozanec/cron-utils/model"
)

// textExpression is a field expression with a fixed text form.
type textExpression string

func (e textExpression) AsString() string { return string(e) }

func (e textExpression) Accept(model.Visitor) (model.FieldExpression, error) { return e, nil }

func TestAlwaysAsString(t *testing.T) {
	if got := model.AlwaysExpression().AsString(); got != "*" {
		t.Errorf("AlwaysExpression().AsString() = %q, want %q", got, "*")
	}
}

func TestAndKeepsExpressionsInOrder(t *testing.T) {
	e1, e2 := textExpression("expression1"), textExpression("expression2")
	and := model.NewAnd().And(e1).And(e2)
	got := and.Expressions()
	if len(got) != 2 || got[0] != e1 || got[1] != e2 {
		t.Errorf("NewAnd().And(expression1).And(expression2).Expressions() = %v, want [expression1 expression2]", got)
	}
	if s := and.AsString(); s != "expression1,expression2" {
		t.Errorf("NewAnd().And(expression1).And(expression2).AsString() = %q, want %q", s, "expression1,expression2")
	}
}

func TestAndExpressionsAreACopy(t *testing.T) {
	and := model.NewAnd().And(textExpression("a"))
	list := and.Expressions()
	list[0] = textExpression("b")
	_ = append(list, textExpression("c"))
	if got := and.Expressions(); len(got) != 1 || got[0] != textExpression("a") {
		t.Errorf("NewAnd().And(a).Expressions() after changing the returned slice = %v, want [a]", got)
	}
}

func TestNewAndContainsBothExpressions(t *testing.T) {
	first, second := textExpression("test"), textExpression("other")
	got := model.NewAnd(first, second).Expressions()
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Errorf("NewAnd(test, other).Expressions() = %v, want [test other]", got)
	}
}

func TestBetweenValues(t *testing.T) {
	b := model.NewBetween(model.IntegerFieldValue(1), model.IntegerFieldValue(5))
	if b.From != model.IntegerFieldValue(1) || b.To != model.IntegerFieldValue(5) {
		t.Errorf("NewBetween(1, 5) = %v-%v, want 1-5", b.From, b.To)
	}
}

func TestBetweenNonNumericRangeSupported(t *testing.T) {
	b := model.NewBetween(model.SpecialCharFieldValue(model.L), model.IntegerFieldValue(5))
	if v, ok := b.From.(model.SpecialCharFieldValue); !ok || v.Value() != model.L {
		t.Errorf("NewBetween(L, 5).From = %v, want L", b.From)
	}
	if got := b.AsString(); got != "L-5" {
		t.Errorf("NewBetween(L, 5).AsString() = %q, want %q", got, "L-5")
	}
}

func TestEveryPeriod(t *testing.T) {
	if got := model.NewEvery(5).Period.Value(); got != 5 {
		t.Errorf("NewEvery(5).Period = %d, want 5", got)
	}
}

func TestEveryAsStringKeepsOneStepFromValue(t *testing.T) {
	if got := model.NewEveryOf(model.NewOn(0), 1).AsString(); got != "0/1" {
		t.Errorf("NewEveryOf(0, 1).AsString() = %q, want %q", got, "0/1")
	}
}

func TestOnValues(t *testing.T) {
	if got := model.NewOn(5).Time.Value(); got != 5 {
		t.Errorf("NewOn(5).Time = %d, want 5", got)
	}
	if got := model.NewOnSpecial(5, model.Hash, 3).Nth.Value(); got != 3 {
		t.Errorf("NewOnSpecial(5, HASH, 3).Nth = %d, want 3", got)
	}
}

func TestOnAsString(t *testing.T) {
	cases := []struct {
		on   *model.On
		want string
	}{
		{model.NewOn(3), "3"},
		{model.NewOnSpecial(1, model.W, model.DefaultNth), "1W"},
		{model.NewOnSpecial(model.DefaultNth, model.L, model.DefaultNth), "L"},
		{model.NewOnSpecial(model.DefaultNth, model.L, 3), "L-3"},
		{model.NewOnSpecial(3, model.Hash, 4), "3#4"},
	}
	for _, c := range cases {
		if got := c.on.AsString(); got != c.want {
			t.Errorf("AsString() of %+v = %q, want %q", *c.on, got, c.want)
		}
	}
}

// optInt prints an optional int, for failure messages.
func optInt(p *int) string {
	if p == nil {
		return "nil"
	}
	return strconv.Itoa(*p)
}

func TestRandomExpressionAsString(t *testing.T) {
	ptr := func(v int) *int { return &v }
	cases := []struct {
		r    model.RandomExpression
		want string
	}{
		{model.RandomExpression{}, "~"},
		{model.RandomExpression{From: ptr(10), To: ptr(20)}, "10~20"},
		{model.RandomExpression{To: ptr(20)}, "~20"},
		{model.RandomExpression{From: ptr(10)}, "10~"},
		{model.RandomExpression{Step: ptr(30)}, "~/30"},
		{model.RandomExpression{From: ptr(10), To: ptr(20), Step: ptr(30)}, "10~20/30"},
	}
	for _, c := range cases {
		if got := c.r.AsString(); got != c.want {
			t.Errorf("RandomExpression{From: %s, To: %s, Step: %s}.AsString() = %q, want %q", optInt(c.r.From), optInt(c.r.To), optInt(c.r.Step), got, c.want)
		}
	}
}

func TestValueMappingVisitorKeepsQuestionMarkSingleton(t *testing.T) {
	v := model.ValueMappingVisitor{Transform: func(f model.FieldValue) model.FieldValue { return f }}
	got, err := model.QuestionMarkExpression().Accept(v)
	if err != nil || got != model.FieldExpression(model.QuestionMarkExpression()) {
		t.Errorf("Accept(ValueMappingVisitor) of ? = %v, %v; want the ? singleton", got, err)
	}
}
