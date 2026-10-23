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
// Ported to Go from com/cronutils/model/definition/{CronDefinition,CronDefinitionBuilder,CronConstraint,
// CronNicknames}.java, com/cronutils/model/field/definition/*.java and com/cronutils/model/CronType.java.

package model

import (
	"slices"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// CronType names a predefined cron dialect.
type CronType int

// The predefined cron dialects.
const (
	Cron4j CronType = iota
	Quartz
	Unix
	Spring
	Spring53
)

var cronTypeNames = [...]string{"CRON4J", "QUARTZ", "UNIX", "SPRING", "SPRING53"}

// String returns the dialect's name, e.g. QUARTZ.
func (t CronType) String() string { return cronTypeNames[t] }

// CronNickname is an @-prefixed shorthand such as @daily.
type CronNickname int

// The cron nicknames.
const (
	Yearly CronNickname = iota
	Annually
	Monthly
	Weekly
	Daily
	Midnight
	Hourly
	Reboot
)

// FieldDefinition describes one field a cron definition accepts.
type FieldDefinition struct {
	FieldName   CronFieldName
	Constraints *FieldConstraints
	Optional    bool
	// MondayDoW is set for a day-of-week field: how the dialect numbers the days of the week.
	MondayDoW *WeekDay
}

// Equal reports whether both definitions describe the same field.
func (d *FieldDefinition) Equal(o *FieldDefinition) bool {
	same := d.FieldName == o.FieldName && d.Optional == o.Optional && d.Constraints.Equal(o.Constraints)
	return same && (d.MondayDoW == nil) == (o.MondayDoW == nil)
}

// CronConstraint is a rule a whole cron must satisfy.
type CronConstraint struct {
	Description string
	Validate    func(Cron) (bool, error)
}

// CronDefinition describes a cron dialect: its fields, whole-cron rules and nicknames.
type CronDefinition struct {
	fieldDefinitions            map[CronFieldName]*FieldDefinition
	constraints                 []*CronConstraint
	nicknames                   map[CronNickname]bool
	matchDayOfWeekAndDayOfMonth bool
}

// NewCronDefinition returns a definition; unless it supports @reboot it needs at least one field, and the first
// field must not be optional.
func NewCronDefinition(fields []*FieldDefinition, constraints []*CronConstraint, nicknames map[CronNickname]bool,
	matchDayOfWeekAndDayOfMonth bool) (*CronDefinition, error) {
	if !nicknames[Reboot] {
		if len(fields) == 0 {
			return nil, cronerr.Argument("Field definitions must not be empty")
		}
		if fields[0].Optional {
			return nil, cronerr.Argument("The first field must not be optional")
		}
	}
	d := &CronDefinition{
		fieldDefinitions:            map[CronFieldName]*FieldDefinition{},
		constraints:                 slices.Clone(constraints),
		nicknames:                   map[CronNickname]bool{},
		matchDayOfWeekAndDayOfMonth: matchDayOfWeekAndDayOfMonth,
	}
	for _, f := range fields {
		d.fieldDefinitions[f.FieldName] = f
	}
	for n, ok := range nicknames {
		if ok {
			d.nicknames[n] = true
		}
	}
	return d, nil
}

// IsMatchDayOfWeekAndDayOfMonth reports whether a day must match both the day-of-week and the day-of-month fields
// (rather than either of them).
func (d *CronDefinition) IsMatchDayOfWeekAndDayOfMonth() bool { return d.matchDayOfWeekAndDayOfMonth }

// FieldDefinitions returns the field definitions in field order.
func (d *CronDefinition) FieldDefinitions() []*FieldDefinition {
	var out []*FieldDefinition
	for _, name := range FieldNames {
		if f, ok := d.fieldDefinitions[name]; ok {
			out = append(out, f)
		}
	}
	return out
}

// FieldDefinition returns the definition of a field, or nil when the dialect lacks the field.
func (d *CronDefinition) FieldDefinition(name CronFieldName) *FieldDefinition {
	return d.fieldDefinitions[name]
}

// ContainsFieldDefinition reports whether the dialect has the field.
func (d *CronDefinition) ContainsFieldDefinition(name CronFieldName) bool {
	_, ok := d.fieldDefinitions[name]
	return ok
}

// Constraints returns the whole-cron rules.
func (d *CronDefinition) Constraints() []*CronConstraint { return slices.Clone(d.constraints) }

// SupportsNickname reports whether the dialect accepts the nickname.
func (d *CronDefinition) SupportsNickname(n CronNickname) bool { return d.nicknames[n] }

// HasNicknames reports whether the dialect accepts any nickname.
func (d *CronDefinition) HasNicknames() bool { return len(d.nicknames) > 0 }

// Equal reports whether both definitions describe the same dialect.
func (d *CronDefinition) Equal(o *CronDefinition) bool {
	if d == o {
		return true
	}
	if d.matchDayOfWeekAndDayOfMonth != o.matchDayOfWeekAndDayOfMonth || len(d.fieldDefinitions) != len(o.fieldDefinitions) ||
		len(d.constraints) != len(o.constraints) || len(d.nicknames) != len(o.nicknames) {
		return false
	}
	for name, f := range d.fieldDefinitions {
		g, ok := o.fieldDefinitions[name]
		if !ok || !f.Equal(g) {
			return false
		}
	}
	for n := range d.nicknames {
		if !o.nicknames[n] {
			return false
		}
	}
	for _, c := range d.constraints {
		if !slices.ContainsFunc(o.constraints, func(x *CronConstraint) bool { return x.Description == c.Description }) {
			return false
		}
	}
	return true
}

// CronDefinitionBuilder builds a CronDefinition field by field.
type CronDefinitionBuilder struct {
	fields                      map[CronFieldName]*FieldDefinition
	constraints                 []*CronConstraint
	nicknames                   map[CronNickname]bool
	matchDayOfWeekAndDayOfMonth bool
	err                         error
}

// DefineCron starts a new definition.
func DefineCron() *CronDefinitionBuilder {
	return &CronDefinitionBuilder{fields: map[CronFieldName]*FieldDefinition{}, nicknames: map[CronNickname]bool{}}
}

// WithSeconds starts the definition of the seconds field.
func (b *CronDefinitionBuilder) WithSeconds() *FieldDefinitionBuilder { return b.field(Second) }

// WithMinutes starts the definition of the minutes field.
func (b *CronDefinitionBuilder) WithMinutes() *FieldDefinitionBuilder { return b.field(Minute) }

// WithHours starts the definition of the hours field.
func (b *CronDefinitionBuilder) WithHours() *FieldDefinitionBuilder { return b.field(Hour) }

// WithDayOfMonth starts the definition of the day-of-month field.
func (b *CronDefinitionBuilder) WithDayOfMonth() *FieldDefinitionBuilder { return b.field(DayOfMonth) }

// WithMonth starts the definition of the month field.
func (b *CronDefinitionBuilder) WithMonth() *FieldDefinitionBuilder { return b.field(Month) }

// WithDayOfWeek starts the definition of the day-of-week field (Monday is 1 unless changed).
func (b *CronDefinitionBuilder) WithDayOfWeek() *FieldDefinitionBuilder {
	f := b.field(DayOfWeek)
	f.dayOfWeek = true
	f.mondayDoWValue = 1
	return f
}

// WithYear starts the definition of the year field.
func (b *CronDefinitionBuilder) WithYear() *FieldDefinitionBuilder { return b.field(Year) }

// WithDayOfYear starts the definition of the day-of-year field.
func (b *CronDefinitionBuilder) WithDayOfYear() *FieldDefinitionBuilder { return b.field(DayOfYear) }

func (b *CronDefinitionBuilder) field(name CronFieldName) *FieldDefinitionBuilder {
	return &FieldDefinitionBuilder{parent: b, name: name, constraints: NewFieldConstraintsBuilder().ForField(name)}
}

// MatchDayOfWeekAndDayOfMonth requires a day to match both day fields.
func (b *CronDefinitionBuilder) MatchDayOfWeekAndDayOfMonth() *CronDefinitionBuilder {
	b.matchDayOfWeekAndDayOfMonth = true
	return b
}

// WithSupportedNickname accepts a nickname.
func (b *CronDefinitionBuilder) WithSupportedNickname(n CronNickname) *CronDefinitionBuilder {
	b.nicknames[n] = true
	return b
}

// WithCronValidation adds a whole-cron rule.
func (b *CronDefinitionBuilder) WithCronValidation(c *CronConstraint) *CronDefinitionBuilder {
	for _, existing := range b.constraints {
		if existing.Description == c.Description {
			return b
		}
	}
	b.constraints = append(b.constraints, c)
	return b
}

// Register adds a field definition; a mandatory field cannot follow an optional one.
func (b *CronDefinitionBuilder) Register(definition *FieldDefinition) error {
	for _, f := range b.fields {
		if f.Optional && !definition.Optional {
			return cronerr.Argument("Can't register mandatory definition after a optional definition.")
		}
	}
	b.fields[definition.FieldName] = definition
	return nil
}

// Instance returns the definition, or the first error met while building it.
func (b *CronDefinitionBuilder) Instance() (*CronDefinition, error) {
	if b.err != nil {
		return nil, b.err
	}
	values := make([]*FieldDefinition, 0, len(b.fields))
	for _, name := range FieldNames {
		if f, ok := b.fields[name]; ok {
			values = append(values, f)
		}
	}
	return NewCronDefinition(values, b.constraints, b.nicknames, b.matchDayOfWeekAndDayOfMonth)
}

// FieldDefinitionBuilder defines one field of a CronDefinitionBuilder.
type FieldDefinitionBuilder struct {
	parent         *CronDefinitionBuilder
	name           CronFieldName
	constraints    *FieldConstraintsBuilder
	optional       bool
	dayOfWeek      bool
	mondayDoWValue int
}

// WithIntMapping maps the number source to dest.
func (f *FieldDefinitionBuilder) WithIntMapping(source, dest int) *FieldDefinitionBuilder {
	f.constraints.WithIntValueMapping(source, dest)
	return f
}

// WithValidRange sets the field's range.
func (f *FieldDefinitionBuilder) WithValidRange(start, end int) *FieldDefinitionBuilder {
	f.constraints.WithValidRange(start, end)
	return f
}

// WithStrictRange rejects ranges that run from a higher to a lower value.
func (f *FieldDefinitionBuilder) WithStrictRange() *FieldDefinitionBuilder {
	f.constraints.WithStrictRange()
	return f
}

// Optional marks the field as optional: expressions may leave it out.
func (f *FieldDefinitionBuilder) Optional() *FieldDefinitionBuilder {
	f.optional = true
	return f
}

// SupportsHash accepts the # special character.
func (f *FieldDefinitionBuilder) SupportsHash() *FieldDefinitionBuilder {
	f.constraints.AddHashSupport()
	return f
}

// SupportsL accepts the L special character.
func (f *FieldDefinitionBuilder) SupportsL() *FieldDefinitionBuilder {
	f.constraints.AddLSupport()
	return f
}

// SupportsW accepts the W special character.
func (f *FieldDefinitionBuilder) SupportsW() *FieldDefinitionBuilder {
	f.constraints.AddWSupport()
	return f
}

// SupportsLW accepts the LW special character.
func (f *FieldDefinitionBuilder) SupportsLW() *FieldDefinitionBuilder {
	f.constraints.AddLWSupport()
	return f
}

// SupportsQuestionMark accepts the ? special character.
func (f *FieldDefinitionBuilder) SupportsQuestionMark() *FieldDefinitionBuilder {
	f.constraints.AddQuestionMarkSupport()
	return f
}

// WithMondayDoWValue sets the number of Monday in a day-of-week field and shifts the day names accordingly.
func (f *FieldDefinitionBuilder) WithMondayDoWValue(mondayDoW int) *FieldDefinitionBuilder {
	f.constraints.WithShiftedStringMapping(mondayDoW - f.mondayDoWValue)
	f.mondayDoWValue = mondayDoW
	return f
}

// And registers the field and returns to the definition builder.
func (f *FieldDefinitionBuilder) And() *CronDefinitionBuilder {
	constraints := f.constraints.Build()
	definition := &FieldDefinition{FieldName: f.name, Constraints: constraints, Optional: f.optional}
	if f.dayOfWeek {
		weekDay, err := NewWeekDay(f.mondayDoWValue, constraints.IsInRange(0))
		if err != nil && f.parent.err == nil {
			f.parent.err = err
		}
		definition.MondayDoW = &weekDay
	}
	if err := f.parent.Register(definition); err != nil && f.parent.err == nil {
		f.parent.err = err
	}
	return f.parent
}

func mustInstance(b *CronDefinitionBuilder) *CronDefinition {
	d, err := b.Instance()
	if err != nil {
		panic(err)
	}
	return d
}

func cron4jDefinition() *CronDefinition {
	return mustInstance(DefineCron().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(0, 31).SupportsL().WithStrictRange().And().
		WithMonth().WithValidRange(1, 12).WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 6).WithMondayDoWValue(1).WithStrictRange().And().
		MatchDayOfWeekAndDayOfMonth())
}

func quartzDefinition() *CronDefinition {
	return mustInstance(DefineCron().
		WithSeconds().WithValidRange(0, 59).And().
		WithMinutes().WithValidRange(0, 59).And().
		WithHours().WithValidRange(0, 23).And().
		WithDayOfMonth().WithValidRange(1, 31).SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().WithValidRange(1, 12).And().
		WithDayOfWeek().WithValidRange(1, 7).WithMondayDoWValue(2).SupportsHash().SupportsL().SupportsQuestionMark().And().
		WithYear().WithValidRange(1970, 2099).WithStrictRange().Optional().And().
		WithCronValidation(EnsureQuartzDayOfMonthAndDayOfWeekValidation()))
}

func springDefinition() *CronDefinition {
	return mustInstance(DefineCron().
		WithSeconds().WithValidRange(0, 59).WithStrictRange().And().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).SupportsQuestionMark().And().
		WithMonth().WithValidRange(1, 12).And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).
		SupportsHash().SupportsQuestionMark().And())
}

func spring53Definition() *CronDefinition {
	return mustInstance(DefineCron().
		WithSeconds().WithValidRange(0, 59).WithStrictRange().And().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).SupportsL().SupportsW().SupportsLW().SupportsQuestionMark().And().
		WithMonth().WithValidRange(1, 12).And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).
		SupportsHash().SupportsL().SupportsQuestionMark().And().
		WithSupportedNickname(Yearly).WithSupportedNickname(Annually).
		WithSupportedNickname(Monthly).
		WithSupportedNickname(Weekly).
		WithSupportedNickname(Daily).WithSupportedNickname(Midnight).
		WithSupportedNickname(Hourly))
}

func unixDefinition() *CronDefinition {
	return mustInstance(DefineCron().
		WithMinutes().WithValidRange(0, 59).WithStrictRange().And().
		WithHours().WithValidRange(0, 23).WithStrictRange().And().
		WithDayOfMonth().WithValidRange(1, 31).WithStrictRange().And().
		WithMonth().WithValidRange(1, 12).WithStrictRange().And().
		WithDayOfWeek().WithValidRange(0, 7).WithMondayDoWValue(1).WithIntMapping(7, 0).WithStrictRange().And())
}

// InstanceDefinitionFor returns the definition of a predefined dialect.
func InstanceDefinitionFor(t CronType) *CronDefinition {
	switch t {
	case Cron4j:
		return cron4jDefinition()
	case Quartz:
		return quartzDefinition()
	case Unix:
		return unixDefinition()
	case Spring:
		return springDefinition()
	}
	return spring53Definition()
}
