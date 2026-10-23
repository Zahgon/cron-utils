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
// Ported to Go from com/cronutils/builder/CronBuilder.java.

// Package builder builds crons field by field for a cron definition, and builds the crons the @-nicknames stand
// for.
package builder

import (
	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/model"
)

// CronBuilder builds a cron field by field; each field is validated when it is added.
type CronBuilder struct {
	definition *model.CronDefinition
	fields     map[model.CronFieldName]*model.CronField
	err        error
}

// Cron starts a cron for the definition.
func Cron(definition *model.CronDefinition) *CronBuilder {
	return &CronBuilder{definition: definition, fields: map[model.CronFieldName]*model.CronField{}}
}

// WithDoY sets the day-of-year field.
func (b *CronBuilder) WithDoY(e model.FieldExpression) *CronBuilder { return b.add(model.DayOfYear, e) }

// WithYear sets the year field.
func (b *CronBuilder) WithYear(e model.FieldExpression) *CronBuilder { return b.add(model.Year, e) }

// WithDoM sets the day-of-month field.
func (b *CronBuilder) WithDoM(e model.FieldExpression) *CronBuilder {
	return b.add(model.DayOfMonth, e)
}

// WithMonth sets the month field.
func (b *CronBuilder) WithMonth(e model.FieldExpression) *CronBuilder { return b.add(model.Month, e) }

// WithDoW sets the day-of-week field.
func (b *CronBuilder) WithDoW(e model.FieldExpression) *CronBuilder { return b.add(model.DayOfWeek, e) }

// WithHour sets the hour field.
func (b *CronBuilder) WithHour(e model.FieldExpression) *CronBuilder { return b.add(model.Hour, e) }

// WithMinute sets the minute field.
func (b *CronBuilder) WithMinute(e model.FieldExpression) *CronBuilder { return b.add(model.Minute, e) }

// WithSecond sets the second field.
func (b *CronBuilder) WithSecond(e model.FieldExpression) *CronBuilder { return b.add(model.Second, e) }

func (b *CronBuilder) add(name model.CronFieldName, e model.FieldExpression) *CronBuilder {
	if b.err != nil {
		return b
	}
	if b.definition == nil {
		b.err = cronerr.New(cronerr.IllegalState, "CronBuilder not initialized.")
		return b
	}
	fd := b.definition.FieldDefinition(name)
	if fd == nil {
		b.err = cronerr.New(cronerr.IllegalState, "Cron field definition does not exist: %s", name)
		return b
	}
	if _, err := e.Accept(model.NewValidationVisitor(fd.Constraints)); err != nil {
		b.err = err
		return b
	}
	b.fields[name] = model.NewCronField(name, e, fd.Constraints)
	return b
}

// Instance returns the validated cron, or the first error met while building it.
func (b *CronBuilder) Instance() (model.Cron, error) {
	if b.err != nil {
		return nil, b.err
	}
	fields := make([]*model.CronField, 0, len(b.fields))
	for _, name := range model.FieldNames {
		if f, ok := b.fields[name]; ok {
			fields = append(fields, f)
		}
	}
	cron := model.NewSingleCron(b.definition, fields)
	if err := cron.Validate(); err != nil {
		return nil, err
	}
	return cron, nil
}

// nickname builds a cron from per-field expressions, setting only the fields the definition has.
func nickname(d *model.CronDefinition, second, minute, hour, dom, month, dow model.FieldExpression) (model.Cron, error) {
	b := Cron(d)
	steps := []struct {
		name model.CronFieldName
		e    model.FieldExpression
	}{
		{model.Second, second}, {model.Minute, minute}, {model.Hour, hour}, {model.DayOfMonth, dom},
		{model.Month, month}, {model.DayOfWeek, dow},
	}
	for _, s := range steps {
		if d.ContainsFieldDefinition(s.name) {
			b.add(s.name, s.e)
		}
	}
	return b.Instance()
}

func on(v int) model.FieldExpression { return model.NewOn(v) }

func always() model.FieldExpression { return model.AlwaysExpression() }

// Yearly returns the cron of @yearly: midnight on January 1st.
func Yearly(d *model.CronDefinition) (model.Cron, error) {
	return nickname(d, on(0), on(0), on(0), on(1), on(1), always())
}

// Annually returns the cron of @annually, the same as @yearly.
func Annually(d *model.CronDefinition) (model.Cron, error) { return Yearly(d) }

// Monthly returns the cron of @monthly: midnight on the first day of every month.
func Monthly(d *model.CronDefinition) (model.Cron, error) {
	return nickname(d, on(0), on(0), on(0), on(1), always(), always())
}

// Weekly returns the cron of @weekly: midnight on day 0 of the week.
func Weekly(d *model.CronDefinition) (model.Cron, error) {
	return nickname(d, on(0), on(0), on(0), always(), always(), on(0))
}

// Daily returns the cron of @daily: every day at midnight.
func Daily(d *model.CronDefinition) (model.Cron, error) {
	return nickname(d, on(0), on(0), on(0), always(), always(), always())
}

// Midnight returns the cron of @midnight, the same as @daily.
func Midnight(d *model.CronDefinition) (model.Cron, error) { return Daily(d) }

// Hourly returns the cron of @hourly: at the start of every hour.
func Hourly(d *model.CronDefinition) (model.Cron, error) {
	return nickname(d, on(0), on(0), always(), always(), always(), always())
}

// Reboot returns the cron of @reboot.
func Reboot(d *model.CronDefinition) model.Cron { return model.NewRebootCron(d) }
