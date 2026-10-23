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
// Ported to Go from com/cronutils/mapper/{CronMapper,ConstantsMapper}.java.

// Package mapper converts crons from one cron definition to another.
package mapper

import (
	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/model"
)

// Predefined day-of-week numberings.
var (
	QuartzWeekDay  = model.QuartzWeekDay
	Java8WeekDay   = model.Java8WeekDay
	CrontabWeekDay = model.CrontabWeekDay
)

// WeekDayMapping converts a day number from the source numbering to the target numbering.
func WeekDayMapping(source, target model.WeekDay, weekday int) int {
	return model.WeekDayMapping(source, target, weekday)
}

type fieldMapping func(*model.CronField) (*model.CronField, error)

// Rules adjusts a mapped cron to the target dialect.
type Rules func(model.Cron) (model.Cron, error)

// CronMapper converts crons of one definition into another.
type CronMapper struct {
	mappings map[model.CronFieldName]fieldMapping
	rules    Rules
	to       *model.CronDefinition
}

// New returns a mapper between two definitions that applies rules to every mapped cron.
func New(from, to *model.CronDefinition, rules Rules) *CronMapper {
	m := &CronMapper{mappings: map[model.CronFieldName]fieldMapping{}, rules: rules, to: to}
	m.buildMappings(from, to)
	return m
}

// Map converts a cron and validates the result against the target definition.
func (m *CronMapper) Map(cron model.Cron) (model.Cron, error) {
	if _, ok := cron.(*model.RebootCron); ok {
		if m.to.SupportsNickname(model.Reboot) {
			return model.NewRebootCron(m.to), nil
		}
		return nil, cronerr.Argument("The target cron definition does not support @reboot nickname")
	}
	var fields []*model.CronField
	for _, name := range model.FieldNames {
		mapping, ok := m.mappings[name]
		if !ok {
			continue
		}
		source, err := cron.Retrieve(name)
		if err != nil {
			return nil, err
		}
		f, err := mapping(source)
		if err != nil {
			return nil, err
		}
		if f != nil {
			fields = append(fields, f)
		}
	}
	mapped, err := m.rules(model.NewSingleCron(m.to, fields))
	if err != nil {
		return nil, err
	}
	if err := mapped.Validate(); err != nil {
		return nil, err
	}
	return mapped, nil
}

// Equivalent reports whether a cron of this mapper's source definition maps to the canonical form of target.
func (m *CronMapper) Equivalent(target, cron model.Cron) (bool, error) {
	mapped, err := m.Map(cron)
	if err != nil {
		return false, err
	}
	return target.AsString() == mapped.AsString(), nil
}

// FromCron4jToQuartz maps cron4j crons to Quartz.
func FromCron4jToQuartz() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Cron4j), model.InstanceDefinitionFor(model.Quartz), SetQuestionMark)
}

// FromQuartzToCron4j maps Quartz crons to cron4j.
func FromQuartzToCron4j() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Quartz), model.InstanceDefinitionFor(model.Cron4j), SameCron)
}

// FromQuartzToUnix maps Quartz crons to Unix.
func FromQuartzToUnix() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Quartz), model.InstanceDefinitionFor(model.Unix), SameCron)
}

// FromUnixToQuartz maps Unix crons to Quartz.
func FromUnixToQuartz() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Unix), model.InstanceDefinitionFor(model.Quartz), SetQuestionMark)
}

// FromQuartzToSpring maps Quartz crons to Spring.
func FromQuartzToSpring() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Quartz), model.InstanceDefinitionFor(model.Spring), SetQuestionMark)
}

// FromSpringToQuartz maps Spring crons to Quartz.
func FromSpringToQuartz() *CronMapper {
	return New(model.InstanceDefinitionFor(model.Spring), model.InstanceDefinitionFor(model.Quartz), SetQuestionMark)
}

// SameCronMapper maps crons of a definition onto the same definition.
func SameCronMapper(definition *model.CronDefinition) *CronMapper {
	return New(definition, definition, SameCron)
}

// SameCron leaves a mapped cron unchanged.
func SameCron(c model.Cron) (model.Cron, error) { return c, nil }

// SetQuestionMark puts "?" in the day-of-week field (or, when that field has a value, in the day-of-month field)
// unless one of them already holds "?".
func SetQuestionMark(c model.Cron) (model.Cron, error) {
	dow, err := c.Retrieve(model.DayOfWeek)
	if err != nil {
		return nil, err
	}
	dom, err := c.Retrieve(model.DayOfMonth)
	if err != nil {
		return nil, err
	}
	if dow == nil && dom == nil {
		return c, nil
	}
	if dow == nil || dom == nil {
		return nil, cronerr.New(cronerr.NullValue, "")
	}
	_, dowQuestion := dow.Expression.(*model.QuestionMark)
	_, domQuestion := dom.Expression.(*model.QuestionMark)
	if dowQuestion || domQuestion {
		return c, nil
	}
	fields, err := c.Fields()
	if err != nil {
		return nil, err
	}
	byName := map[model.CronFieldName]*model.CronField{}
	for _, f := range fields {
		byName[f.Field] = f
	}
	if _, always := dow.Expression.(*model.Always); always {
		byName[model.DayOfWeek] = model.NewCronField(model.DayOfWeek, model.QuestionMarkExpression(), dow.Constraints)
	} else if _, always := dom.Expression.(*model.Always); always {
		byName[model.DayOfMonth] = model.NewCronField(model.DayOfMonth, model.QuestionMarkExpression(), dom.Constraints)
	} else if err := c.Validate(); err != nil {
		return nil, err
	}
	ordered := make([]*model.CronField, 0, len(byName))
	for _, name := range model.FieldNames {
		if f, ok := byName[name]; ok {
			ordered = append(ordered, f)
		}
	}
	return model.NewSingleCron(c.Definition(), ordered), nil
}

func (m *CronMapper) buildMappings(from, to *model.CronDefinition) {
	startedDest, startedSource := false, false
	for _, name := range model.FieldNames {
		dest := to.FieldDefinition(name)
		source := from.FieldDefinition(name)
		if dest != nil {
			startedDest = true
		}
		if source != nil {
			startedSource = true
		}
		if startedDest && dest == nil {
			break
		}
		if !startedSource && dest != nil {
			m.mappings[name] = onZero(name)
		}
		if startedSource && source == nil && dest != nil {
			m.mappings[name] = alwaysField(name)
		}
		if source == nil || dest == nil {
			continue
		}
		switch name {
		case model.DayOfWeek:
			m.mappings[name] = dayOfWeekMapping(source, dest)
		case model.DayOfMonth:
			m.mappings[name] = dayOfMonthMapping(dest)
		default:
			m.mappings[name] = func(f *model.CronField) (*model.CronField, error) { return f, nil }
		}
	}
}

func onZero(name model.CronFieldName) fieldMapping {
	return func(*model.CronField) (*model.CronField, error) {
		return model.NewCronField(name, model.NewOn(0), model.NewFieldConstraintsBuilder().ForField(name).Build()), nil
	}
}

func alwaysField(name model.CronFieldName) fieldMapping {
	return func(*model.CronField) (*model.CronField, error) {
		return model.NewCronField(name, model.AlwaysExpression(), model.NewFieldConstraintsBuilder().ForField(name).Build()), nil
	}
}

// dayOfWeekVisitor renumbers the days of a day-of-week expression.
type dayOfWeekVisitor struct {
	model.VisitorAdaptor
	source, target model.WeekDay
}

func (v dayOfWeekVisitor) mapValue(value model.FieldValue) model.FieldValue {
	if i, ok := value.(model.IntegerFieldValue); ok {
		return model.IntegerFieldValue(model.WeekDayMapping(v.source, v.target, int(i)))
	}
	return value
}

func (v dayOfWeekVisitor) VisitEvery(e *model.Every) (model.FieldExpression, error) {
	inner, err := e.Expression.Accept(v)
	if err != nil {
		return nil, err
	}
	return model.NewEveryOf(inner, e.Period), nil
}

func (v dayOfWeekVisitor) VisitOn(o *model.On) (model.FieldExpression, error) {
	return &model.On{Time: v.mapValue(o.Time).(model.IntegerFieldValue), SpecialChar: o.SpecialChar, Nth: o.Nth}, nil
}

func (v dayOfWeekVisitor) VisitBetween(b *model.Between) (model.FieldExpression, error) {
	return model.NewBetween(v.mapValue(b.From), v.mapValue(b.To)), nil
}

func (v dayOfWeekVisitor) VisitAnd(a *model.And) (model.FieldExpression, error) {
	out := model.NewAnd()
	for _, e := range a.Expressions() {
		mapped, err := e.Accept(v)
		if err != nil {
			return nil, err
		}
		out.And(mapped)
	}
	return out, nil
}

func dayOfWeekMapping(source, target *model.FieldDefinition) fieldMapping {
	return func(f *model.CronField) (*model.CronField, error) {
		if f == nil {
			return nil, cronerr.New(cronerr.NullValue, "")
		}
		dest, err := f.Expression.Accept(dayOfWeekVisitor{source: *source.MondayDoW, target: *target.MondayDoW})
		if err != nil {
			return nil, err
		}
		if _, q := f.Expression.(*model.QuestionMark); q && !target.Constraints.SupportsSpecialChar(model.QuestionMarkChar) {
			dest = model.AlwaysExpression()
		}
		return model.NewCronField(model.DayOfWeek, dest, target.Constraints), nil
	}
}

func dayOfMonthMapping(target *model.FieldDefinition) fieldMapping {
	return func(f *model.CronField) (*model.CronField, error) {
		if f == nil {
			return nil, cronerr.New(cronerr.NullValue, "")
		}
		dest := f.Expression
		if _, q := dest.(*model.QuestionMark); q && !target.Constraints.SupportsSpecialChar(model.QuestionMarkChar) {
			dest = model.AlwaysExpression()
		}
		return model.NewCronField(model.DayOfMonth, dest, target.Constraints), nil
	}
}
