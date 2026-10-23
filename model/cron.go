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
// Ported to Go from com/cronutils/model/{Cron,SingleCron,CompositeCron,RebootCron}.java and
// com/cronutils/model/definition/CronConstraintsFactory.java.

package model

import (
	"fmt"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// Cron is a parsed cron expression.
type Cron interface {
	// Retrieve returns the parsed field, or nil when the cron has no such field.
	Retrieve(name CronFieldName) (*CronField, error)
	// Fields returns the parsed fields in field order.
	Fields() ([]*CronField, error)
	// AsString returns the expression in canonical cron syntax.
	AsString() string
	// Definition returns the dialect the cron was parsed with.
	Definition() *CronDefinition
	// Validate checks every field against its constraints and the cron against the dialect's rules.
	Validate() error
	// Equivalent reports whether both crons have the same canonical form.
	Equivalent(other Cron) bool
}

// SingleCron is a cron with one value per field.
type SingleCron struct {
	definition *CronDefinition
	fields     map[CronFieldName]*CronField
}

// NewSingleCron returns a cron of the given fields; a later field replaces an earlier one with the same name.
func NewSingleCron(definition *CronDefinition, fields []*CronField) *SingleCron {
	c := &SingleCron{definition: definition, fields: map[CronFieldName]*CronField{}}
	for _, f := range fields {
		c.fields[f.Field] = f
	}
	return c
}

// Retrieve returns the parsed field, or nil.
func (c *SingleCron) Retrieve(name CronFieldName) (*CronField, error) {
	return c.fields[name], nil
}

// Fields returns the parsed fields in field order.
func (c *SingleCron) Fields() ([]*CronField, error) {
	return c.orderedFields(), nil
}

// FieldsInOrder returns the parsed fields in field order.
func (c *SingleCron) FieldsInOrder() []*CronField {
	return c.orderedFields()
}

// Field returns the parsed field, or nil when the cron has no such field.
func (c *SingleCron) Field(name CronFieldName) *CronField {
	return c.fields[name]
}

func (c *SingleCron) orderedFields() []*CronField {
	out := make([]*CronField, 0, len(c.fields))
	for _, name := range FieldNames {
		if f, ok := c.fields[name]; ok {
			out = append(out, f)
		}
	}
	return out
}

// AsString joins the fields' expressions with spaces.
func (c *SingleCron) AsString() string {
	parts := make([]string, 0, len(c.fields))
	for _, f := range c.orderedFields() {
		parts = append(parts, f.Expression.AsString())
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// Definition returns the dialect.
func (c *SingleCron) Definition() *CronDefinition { return c.definition }

// Validate checks every field against its constraints, then the dialect's whole-cron rules.
func (c *SingleCron) Validate() error {
	for _, f := range c.orderedFields() {
		fd := c.definition.FieldDefinition(f.Field)
		if fd == nil {
			return cronerr.New(cronerr.NullValue, "")
		}
		if _, err := f.Expression.Accept(NewValidationVisitor(fd.Constraints)); err != nil {
			return err
		}
	}
	return validateConstraints(c)
}

func validateConstraints(c Cron) error {
	for _, constraint := range c.Definition().constraints {
		ok, err := constraint.Validate(c)
		if err != nil {
			return err
		}
		if !ok {
			return cronerr.Argument("Invalid cron expression: %s. %s", c.AsString(), constraint.Description)
		}
	}
	return nil
}

// Equivalent compares canonical forms.
func (c *SingleCron) Equivalent(other Cron) bool { return c.AsString() == other.AsString() }

// RebootCron is the @reboot nickname: it never runs on a schedule.
type RebootCron struct {
	definition *CronDefinition
}

// NewRebootCron returns the @reboot cron of a dialect.
func NewRebootCron(definition *CronDefinition) *RebootCron {
	return &RebootCron{definition: definition}
}

// Retrieve returns nil: @reboot has no fields.
func (c *RebootCron) Retrieve(CronFieldName) (*CronField, error) { return nil, nil }

// Fields returns no fields.
func (c *RebootCron) Fields() ([]*CronField, error) { return nil, nil }

// AsString returns "@reboot".
func (c *RebootCron) AsString() string { return "@reboot" }

// Definition returns the dialect.
func (c *RebootCron) Definition() *CronDefinition { return c.definition }

// Validate checks the dialect's whole-cron rules.
func (c *RebootCron) Validate() error { return validateConstraints(c) }

// Equivalent compares canonical forms.
func (c *RebootCron) Equivalent(other Cron) bool { return c.AsString() == other.AsString() }

// CompositeCron is a set of crons of one dialect that run on the union of their schedules.
type CompositeCron struct {
	crons      []Cron
	definition *CronDefinition
}

// NewCompositeCron returns the union of crons, which must be non-empty and share one dialect.
func NewCompositeCron(crons []Cron) (*CompositeCron, error) {
	if len(crons) == 0 {
		return nil, cronerr.Argument("List of Cron cannot be null or empty")
	}
	definition := crons[0].Definition()
	for _, c := range crons {
		if !c.Definition().Equal(definition) {
			return nil, cronerr.Argument("All Cron objects must have same definition for CompositeCron")
		}
	}
	return &CompositeCron{crons: append([]Cron(nil), crons...), definition: definition}, nil
}

// Crons returns the member crons.
func (c *CompositeCron) Crons() []Cron { return append([]Cron(nil), c.crons...) }

func unsupportedForComposite() error {
	return cronerr.New(cronerr.UnsupportedOperation, "Currently not supported for CompositeCron")
}

// Retrieve is not supported for a composite cron.
func (c *CompositeCron) Retrieve(CronFieldName) (*CronField, error) {
	return nil, unsupportedForComposite()
}

// Fields is not supported for a composite cron.
func (c *CompositeCron) Fields() ([]*CronField, error) { return nil, unsupportedForComposite() }

// AsString merges the members field by field (see CheckedAsString); members with fewer fields than the first one
// contribute nothing to the fields they lack.
func (c *CompositeCron) AsString() string {
	return c.merge(c.patterns())
}

// CheckedAsString merges the members field by field: a field whose members all agree is written once, otherwise the
// members' values are joined with "|". It reports an error when a member has fewer fields than the first member.
func (c *CompositeCron) CheckedAsString() (string, error) {
	patterns := c.patterns()
	for _, p := range patterns {
		if len(p) < len(patterns[0]) {
			return "", cronerr.New(cronerr.IndexOutOfBounds, "Index %d out of bounds for length %d", len(p), len(p))
		}
	}
	return c.merge(patterns), nil
}

// patterns returns the fields of each member's canonical form.
func (c *CompositeCron) patterns() [][]string {
	patterns := make([][]string, len(c.crons))
	for i, cron := range c.crons {
		patterns[i] = strings.Split(cron.AsString(), " ")
	}
	return patterns
}

func (c *CompositeCron) merge(patterns [][]string) string {
	var b strings.Builder
	for j := range patterns[0] {
		values := make([]string, 0, len(patterns))
		for _, p := range patterns {
			if j < len(p) {
				values = append(values, p[j])
			}
		}
		field := strings.Join(values, "|")
		if distinct(values) <= 1 {
			field = values[0]
		}
		fmt.Fprintf(&b, "%s ", field)
	}
	return strings.TrimSpace(b.String())
}

func distinct(values []string) int {
	seen := map[string]bool{}
	for _, v := range values {
		seen[v] = true
	}
	return len(seen)
}

// CanonicalString returns the canonical form of any cron, reporting an error when a composite cron's members cannot
// be merged.
func CanonicalString(c Cron) (string, error) {
	if composite, ok := c.(*CompositeCron); ok {
		return composite.CheckedAsString()
	}
	return c.AsString(), nil
}

// Definition returns the dialect.
func (c *CompositeCron) Definition() *CronDefinition { return c.definition }

// Validate validates every member.
func (c *CompositeCron) Validate() error {
	for _, cron := range c.crons {
		if err := cron.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Equivalent compares canonical forms.
func (c *CompositeCron) Equivalent(other Cron) bool { return c.AsString() == other.AsString() }

func isQuestionMark(f *CronField) bool {
	if f == nil {
		return false
	}
	_, ok := f.Expression.(*QuestionMark)
	return ok
}

// EnsureEitherDayOfYearOrMonth rejects a day-of-year together with a day-of-month or day-of-week.
func EnsureEitherDayOfYearOrMonth() *CronConstraint {
	return &CronConstraint{
		Description: "Both, a day-of-year AND a day-of-month or day-of-week, are not supported.",
		Validate: func(c Cron) (bool, error) {
			doy, err := c.Retrieve(DayOfYear)
			if err != nil {
				return false, err
			}
			if doy != nil && !isQuestionMark(doy) {
				dow, err := c.Retrieve(DayOfWeek)
				if err != nil {
					return false, err
				}
				dom, err := c.Retrieve(DayOfMonth)
				if err != nil {
					return false, err
				}
				if dow == nil || dom == nil {
					return false, cronerr.New(cronerr.NullValue, "")
				}
				return isQuestionMark(dow) && isQuestionMark(dom), nil
			}
			return true, nil
		},
	}
}

// EnsureEitherDayOfWeekOrDayOfMonth requires exactly one of the day-of-week and day-of-month fields to be "?".
func EnsureEitherDayOfWeekOrDayOfMonth() *CronConstraint {
	return &CronConstraint{
		Description: "Both, a day-of-week AND a day-of-month parameter, are not supported.",
		Validate: func(c Cron) (bool, error) {
			doy, err := c.Retrieve(DayOfYear)
			if err != nil {
				return false, err
			}
			dom, err := c.Retrieve(DayOfMonth)
			if err != nil {
				return false, err
			}
			dow, err := c.Retrieve(DayOfWeek)
			if err != nil {
				return false, err
			}
			if doy == nil || isQuestionMark(doy) {
				if dom != nil && !isQuestionMark(dom) {
					return dow != nil && isQuestionMark(dow), nil
				}
				return dow != nil && !isQuestionMark(dow), nil
			}
			return true, nil
		},
	}
}

// EnsureQuartzDayOfMonthAndDayOfWeekValidation requires at least one of the day-of-week and day-of-month fields to
// be "?".
func EnsureQuartzDayOfMonthAndDayOfWeekValidation() *CronConstraint {
	return &CronConstraint{
		Description: "Invalid cron expression: Both, a day-of-week AND a day-of-month parameter, must have at least one with a '?' character.",
		Validate: func(c Cron) (bool, error) {
			dom, err := c.Retrieve(DayOfMonth)
			if err != nil {
				return false, err
			}
			dow, err := c.Retrieve(DayOfWeek)
			if err != nil {
				return false, err
			}
			return isQuestionMark(dom) || isQuestionMark(dow), nil
		},
	}
}
