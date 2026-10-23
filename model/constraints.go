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
// Ported to Go from com/cronutils/model/field/constraint/{FieldConstraints,FieldConstraintsBuilder}.java and
// com/cronutils/model/field/{CronField,CronFieldName}.java.

package model

import (
	"maps"
	"math"
)

// CronFieldName names a cron field; the order of the constants is the order of the fields in an expression.
type CronFieldName int

// The cron fields.
const (
	Second CronFieldName = iota
	Minute
	Hour
	DayOfMonth
	Month
	DayOfWeek
	Year
	DayOfYear
)

var fieldNames = [...]string{"SECOND", "MINUTE", "HOUR", "DAY_OF_MONTH", "MONTH", "DAY_OF_WEEK", "YEAR", "DAY_OF_YEAR"}

// FieldNames lists every field in order.
var FieldNames = []CronFieldName{Second, Minute, Hour, DayOfMonth, Month, DayOfWeek, Year, DayOfYear}

// String returns the field's name, e.g. DAY_OF_MONTH.
func (n CronFieldName) String() string { return fieldNames[n] }

// Order returns the field's position among the fields.
func (n CronFieldName) Order() int { return int(n) }

// FieldConstraints are the values a field accepts: its range, the names it maps to numbers, the numbers it maps to
// other numbers and the special characters it supports.
type FieldConstraints struct {
	stringMapping map[string]int
	intMapping    map[int]int
	specialChars  map[SpecialChar]bool
	startRange    int
	endRange      int
	strictRange   bool
}

// StartRange returns the lowest valid value.
func (c *FieldConstraints) StartRange() int { return c.startRange }

// EndRange returns the highest valid value.
func (c *FieldConstraints) EndRange() int { return c.endRange }

// IsStrictRange reports whether a range must not run from a higher to a lower value.
func (c *FieldConstraints) IsStrictRange() bool { return c.strictRange }

// SupportsSpecialChar reports whether the field accepts the special character.
func (c *FieldConstraints) SupportsSpecialChar(s SpecialChar) bool { return c.specialChars[s] }

// SpecialChars returns the supported special characters.
func (c *FieldConstraints) SpecialChars() map[SpecialChar]bool { return maps.Clone(c.specialChars) }

// IsInRange reports whether value lies in the field's range.
func (c *FieldConstraints) IsInRange(value int) bool {
	return value >= c.startRange && value <= c.endRange
}

// IsPeriodInRange reports whether period is a valid step for the field.
func (c *FieldConstraints) IsPeriodInRange(period int) bool {
	return period > 0 && period <= c.endRange-c.startRange+1 && period <= c.endRange
}

// StringMappingKeys returns the names the field maps to numbers.
func (c *FieldConstraints) StringMappingKeys() []string {
	keys := make([]string, 0, len(c.stringMapping))
	for k := range c.stringMapping {
		keys = append(keys, k)
	}
	return keys
}

// StringMappingValue returns the number a name maps to.
func (c *FieldConstraints) StringMappingValue(exp string) (int, bool) {
	v, ok := c.stringMapping[exp]
	return v, ok
}

// IntMappingValue returns the number a number maps to.
func (c *FieldConstraints) IntMappingValue(exp int) (int, bool) {
	v, ok := c.intMapping[exp]
	return v, ok
}

// Equal reports whether both constraints accept the same values.
func (c *FieldConstraints) Equal(o *FieldConstraints) bool {
	return c.strictRange == o.strictRange && c.startRange == o.startRange && c.endRange == o.endRange &&
		maps.Equal(c.stringMapping, o.stringMapping) && maps.Equal(c.intMapping, o.intMapping) &&
		maps.Equal(c.specialChars, o.specialChars)
}

// FieldConstraintsBuilder builds FieldConstraints.
type FieldConstraintsBuilder struct {
	stringMapping map[string]int
	intMapping    map[int]int
	startRange    int
	endRange      int
	strictRange   bool
	specialChars  map[SpecialChar]bool
}

// NewFieldConstraintsBuilder returns a builder for a field with range 0 to the largest 32-bit integer.
func NewFieldConstraintsBuilder() *FieldConstraintsBuilder {
	return &FieldConstraintsBuilder{
		stringMapping: map[string]int{},
		intMapping:    map[int]int{},
		endRange:      math.MaxInt32,
		specialChars:  map[SpecialChar]bool{None: true},
	}
}

// ForField sets the default range and names of a field.
func (b *FieldConstraintsBuilder) ForField(field CronFieldName) *FieldConstraintsBuilder {
	switch field {
	case Second, Minute:
		b.endRange = 59
	case Hour:
		b.endRange = 23
	case DayOfWeek:
		b.stringMapping = daysOfWeekMapping()
		b.endRange = 6
	case DayOfMonth:
		b.startRange, b.endRange = 1, 31
	case Month:
		b.stringMapping = monthsMapping()
		b.startRange, b.endRange = 1, 12
	case DayOfYear:
		b.startRange, b.endRange = 1, 366
	}
	return b
}

// AddHashSupport accepts the # special character.
func (b *FieldConstraintsBuilder) AddHashSupport() *FieldConstraintsBuilder { return b.add(Hash) }

// AddLSupport accepts the L special character.
func (b *FieldConstraintsBuilder) AddLSupport() *FieldConstraintsBuilder { return b.add(L) }

// AddWSupport accepts the W special character.
func (b *FieldConstraintsBuilder) AddWSupport() *FieldConstraintsBuilder { return b.add(W) }

// AddLWSupport accepts the LW special character.
func (b *FieldConstraintsBuilder) AddLWSupport() *FieldConstraintsBuilder { return b.add(LW) }

// AddQuestionMarkSupport accepts the ? special character.
func (b *FieldConstraintsBuilder) AddQuestionMarkSupport() *FieldConstraintsBuilder {
	return b.add(QuestionMarkChar)
}

func (b *FieldConstraintsBuilder) add(s SpecialChar) *FieldConstraintsBuilder {
	b.specialChars[s] = true
	return b
}

// WithIntValueMapping maps the number source to dest.
func (b *FieldConstraintsBuilder) WithIntValueMapping(source, dest int) *FieldConstraintsBuilder {
	b.intMapping[source] = dest
	return b
}

// WithValidRange sets the range.
func (b *FieldConstraintsBuilder) WithValidRange(start, end int) *FieldConstraintsBuilder {
	b.startRange, b.endRange = start, end
	return b
}

// WithStrictRange rejects ranges that run from a higher to a lower value.
func (b *FieldConstraintsBuilder) WithStrictRange() *FieldConstraintsBuilder {
	b.strictRange = true
	return b
}

// WithShiftedStringMapping shifts the numbers the names map to, wrapping around the range.
func (b *FieldConstraintsBuilder) WithShiftedStringMapping(shift int) *FieldConstraintsBuilder {
	if shift > 0 || b.endRange < len(b.stringMapping) {
		size := len(b.stringMapping)
		for k, value := range b.stringMapping {
			value += shift
			if value > b.endRange {
				value -= size
			} else if value < b.startRange {
				value += b.startRange - b.endRange
			}
			b.stringMapping[k] = value
		}
	}
	return b
}

// Build returns the constraints.
func (b *FieldConstraintsBuilder) Build() *FieldConstraints {
	return &FieldConstraints{
		stringMapping: maps.Clone(b.stringMapping),
		intMapping:    maps.Clone(b.intMapping),
		specialChars:  maps.Clone(b.specialChars),
		startRange:    b.startRange,
		endRange:      b.endRange,
		strictRange:   b.strictRange,
	}
}

func daysOfWeekMapping() map[string]int {
	return map[string]int{"MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6, "SUN": 7}
}

func monthsMapping() map[string]int {
	return map[string]int{"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8, "SEP": 9,
		"OCT": 10, "NOV": 11, "DEC": 12}
}

// CronField is one parsed field of a cron: its name, expression and constraints.
type CronField struct {
	Field       CronFieldName
	Expression  FieldExpression
	Constraints *FieldConstraints
}

// NewCronField returns a parsed field.
func NewCronField(field CronFieldName, expression FieldExpression, constraints *FieldConstraints) *CronField {
	return &CronField{Field: field, Expression: expression, Constraints: constraints}
}
