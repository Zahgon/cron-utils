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
// Ported to Go from com/cronutils/model/field/expression/{FieldExpression,Always,And,Between,Every,On,QuestionMark,
// RandomExpression,FieldExpressionFactory}.java.

package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// FieldExpression is the parsed form of one cron field.
type FieldExpression interface {
	// AsString returns the expression in cron syntax.
	AsString() string
	// Accept passes the expression to the matching method of the visitor.
	Accept(v Visitor) (FieldExpression, error)
}

// Always is the expression "*": every value of the field.
type Always struct{}

// QuestionMark is the expression "?": no specific value.
type QuestionMark struct{}

var (
	alwaysInstance       = &Always{}
	questionMarkInstance = &QuestionMark{}
)

// AlwaysExpression returns the "*" expression.
func AlwaysExpression() *Always { return alwaysInstance }

// QuestionMarkExpression returns the "?" expression.
func QuestionMarkExpression() *QuestionMark { return questionMarkInstance }

// AsString returns "*".
func (*Always) AsString() string { return "*" }

// Accept calls v.VisitAlways.
func (a *Always) Accept(v Visitor) (FieldExpression, error) { return v.VisitAlways(a) }

// AsString returns "?".
func (*QuestionMark) AsString() string { return "?" }

// Accept calls v.VisitQuestionMark.
func (q *QuestionMark) Accept(v Visitor) (FieldExpression, error) { return v.VisitQuestionMark(q) }

// And is a comma-separated list of expressions.
type And struct {
	expressions []FieldExpression
}

// NewAnd returns a list of the given expressions.
func NewAnd(expressions ...FieldExpression) *And {
	return &And{expressions: append([]FieldExpression(nil), expressions...)}
}

// And appends an expression to the list.
func (a *And) And(exp FieldExpression) *And {
	a.expressions = append(a.expressions, exp)
	return a
}

// Expressions returns the listed expressions.
func (a *And) Expressions() []FieldExpression {
	return append([]FieldExpression(nil), a.expressions...)
}

// AsString joins the listed expressions with commas.
func (a *And) AsString() string {
	parts := make([]string, len(a.expressions))
	for i, e := range a.expressions {
		parts[i] = e.AsString()
	}
	return strings.Join(parts, ",")
}

// Accept calls v.VisitAnd.
func (a *And) Accept(v Visitor) (FieldExpression, error) { return v.VisitAnd(a) }

// Between is a range from-to.
type Between struct {
	From, To FieldValue
}

// NewBetween returns the range from-to.
func NewBetween(from, to FieldValue) *Between {
	return &Between{From: from, To: to}
}

// AsString returns from-to.
func (b *Between) AsString() string {
	return fmt.Sprintf("%s-%s", b.From, b.To)
}

// Accept calls v.VisitBetween.
func (b *Between) Accept(v Visitor) (FieldExpression, error) { return v.VisitBetween(b) }

// Every is a stepped expression: expression/period.
type Every struct {
	Expression FieldExpression
	Period     IntegerFieldValue
}

// NewEvery returns */period.
func NewEvery(period IntegerFieldValue) *Every {
	return &Every{Expression: alwaysInstance, Period: period}
}

// NewEveryOf returns expression/period.
func NewEveryOf(expression FieldExpression, period IntegerFieldValue) *Every {
	return &Every{Expression: expression, Period: period}
}

// AsString returns expression/period, or "*" for */1.
func (e *Every) AsString() string {
	s := e.Expression.AsString()
	if s == "*" && e.Period == 1 {
		return s
	}
	return s + "/" + e.Period.String()
}

// Accept calls v.VisitEvery.
func (e *Every) Accept(v Visitor) (FieldExpression, error) { return v.VisitEvery(e) }

// DefaultNth is the time or nth value of an On expression that carries no number.
const DefaultNth = -1

// On is a single value, optionally with a special character: 5, 5L, L, L-3, 15W, LW, 6#3, ?.
type On struct {
	Time        IntegerFieldValue
	SpecialChar SpecialCharFieldValue
	Nth         IntegerFieldValue
}

// NewOn returns the plain value time.
func NewOn(time int) *On {
	return &On{Time: IntegerFieldValue(time), SpecialChar: SpecialCharFieldValue(None), Nth: DefaultNth}
}

// OnTime returns the plain value time, rejecting negative values (FieldExpressionFactory.on).
func OnTime(time int) (*On, error) {
	if time < 0 {
		return nil, cronerr.Argument("Time must not be negative")
	}
	return NewOn(time), nil
}

// NewOnSpecial returns time with a special character and an nth value.
func NewOnSpecial(time int, special SpecialChar, nth int) *On {
	return &On{Time: IntegerFieldValue(time), SpecialChar: SpecialCharFieldValue(special), Nth: IntegerFieldValue(nth)}
}

// AsString returns the value in cron syntax.
func (o *On) AsString() string {
	switch o.SpecialChar.Value() {
	case None:
		return o.Time.String()
	case Hash:
		return fmt.Sprintf("%s#%s", o.Time, o.Nth)
	case W:
		if o.Time == DefaultNth {
			return "W"
		}
		return fmt.Sprintf("%sW", o.Time)
	case L:
		if o.Time == DefaultNth {
			if o.Nth == DefaultNth {
				return "L"
			}
			return "L-" + o.Nth.String()
		}
		return fmt.Sprintf("%sL", o.Time)
	case QuestionMarkChar:
		return "?"
	}
	return o.SpecialChar.String()
}

// Accept calls v.VisitOn.
func (o *On) Accept(v Visitor) (FieldExpression, error) { return v.VisitOn(o) }

// RandomExpression is a randomised value: ~, a~b, ~/s, a~b/s.
type RandomExpression struct {
	From, To, Step *int
}

// AsString returns the expression in cron syntax.
func (r *RandomExpression) AsString() string {
	var b strings.Builder
	if r.From != nil {
		b.WriteString(strconv.Itoa(*r.From))
	}
	b.WriteString("~")
	if r.To != nil {
		b.WriteString(strconv.Itoa(*r.To))
	}
	if r.Step != nil {
		b.WriteString("/" + strconv.Itoa(*r.Step))
	}
	return b.String()
}

// Accept calls v.VisitRandom.
func (r *RandomExpression) Accept(v Visitor) (FieldExpression, error) { return v.VisitRandom(r) }

// TypeName returns the qualified class name the original library reports for an expression type.
func TypeName(e FieldExpression) string {
	const pkg = "class com.cronutils.model.field.expression."
	switch e.(type) {
	case *Always:
		return pkg + "Always"
	case *And:
		return pkg + "And"
	case *Between:
		return pkg + "Between"
	case *Every:
		return pkg + "Every"
	case *On:
		return pkg + "On"
	case *QuestionMark:
		return pkg + "QuestionMark"
	case *RandomExpression:
		return pkg + "RandomExpression"
	}
	return fmt.Sprintf("%T", e)
}

// Visitor receives the concrete type of a field expression.
type Visitor interface {
	VisitAlways(*Always) (FieldExpression, error)
	VisitAnd(*And) (FieldExpression, error)
	VisitBetween(*Between) (FieldExpression, error)
	VisitEvery(*Every) (FieldExpression, error)
	VisitOn(*On) (FieldExpression, error)
	VisitQuestionMark(*QuestionMark) (FieldExpression, error)
	VisitRandom(*RandomExpression) (FieldExpression, error)
}

// VisitorAdaptor returns every expression unchanged; embed it to override only some kinds.
type VisitorAdaptor struct{}

// VisitAlways returns the expression.
func (VisitorAdaptor) VisitAlways(a *Always) (FieldExpression, error) { return a, nil }

// VisitAnd returns the expression.
func (VisitorAdaptor) VisitAnd(a *And) (FieldExpression, error) { return a, nil }

// VisitBetween returns the expression.
func (VisitorAdaptor) VisitBetween(b *Between) (FieldExpression, error) { return b, nil }

// VisitEvery returns the expression.
func (VisitorAdaptor) VisitEvery(e *Every) (FieldExpression, error) { return e, nil }

// VisitOn returns the expression.
func (VisitorAdaptor) VisitOn(o *On) (FieldExpression, error) { return o, nil }

// VisitQuestionMark returns the expression.
func (VisitorAdaptor) VisitQuestionMark(q *QuestionMark) (FieldExpression, error) { return q, nil }

// VisitRandom returns the expression.
func (VisitorAdaptor) VisitRandom(r *RandomExpression) (FieldExpression, error) { return r, nil }

// ValueMappingVisitor rebuilds an expression with every value passed through a transform.
type ValueMappingVisitor struct {
	Transform func(FieldValue) FieldValue
}

// VisitAlways returns the expression.
func (m ValueMappingVisitor) VisitAlways(a *Always) (FieldExpression, error) { return a, nil }

// VisitAnd maps every listed expression.
func (m ValueMappingVisitor) VisitAnd(a *And) (FieldExpression, error) {
	clone := NewAnd()
	for _, e := range a.expressions {
		mapped, err := e.Accept(m)
		if err != nil {
			return nil, err
		}
		clone.And(mapped)
	}
	return clone, nil
}

// VisitBetween maps both ends of the range.
func (m ValueMappingVisitor) VisitBetween(b *Between) (FieldExpression, error) {
	return NewBetween(m.Transform(b.From), m.Transform(b.To)), nil
}

// VisitEvery maps the period.
func (m ValueMappingVisitor) VisitEvery(e *Every) (FieldExpression, error) {
	return NewEvery(m.Transform(e.Period).(IntegerFieldValue)), nil
}

// VisitOn maps the value.
func (m ValueMappingVisitor) VisitOn(o *On) (FieldExpression, error) {
	return &On{Time: m.Transform(o.Time).(IntegerFieldValue), SpecialChar: o.SpecialChar, Nth: o.Nth}, nil
}

// VisitQuestionMark returns the "?" expression.
func (m ValueMappingVisitor) VisitQuestionMark(*QuestionMark) (FieldExpression, error) {
	return questionMarkInstance, nil
}

// VisitRandom returns the expression.
func (m ValueMappingVisitor) VisitRandom(r *RandomExpression) (FieldExpression, error) { return r, nil }
