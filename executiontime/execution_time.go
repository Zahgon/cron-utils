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
// Ported to Go from com/cronutils/model/time/{ExecutionTime,ExecutionTimeBuilder,CompositeExecutionTime}.java.

// Package executiontime computes when a cron runs: the next and the last execution around a zoned date-time, the
// time to or from them, whether a date-time matches, and the executions inside a window.
package executiontime

import (
	"slices"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

// ExecutionTime computes the executions of a cron. The boolean results report whether an execution exists.
type ExecutionTime interface {
	NextExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error)
	TimeToNextExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error)
	LastExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error)
	TimeFromLastExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error)
	IsMatch(date javatime.ZonedDateTime) (bool, error)
}

// ForCron returns the execution times of a cron.
func ForCron(cron model.Cron) (ExecutionTime, error) {
	switch c := cron.(type) {
	case *model.SingleCron:
		return forSingleCron(c)
	case *model.CompositeCron:
		var members []ExecutionTime
		for _, member := range c.Crons() {
			et, err := ForCron(member)
			if err != nil {
				return nil, err
			}
			members = append(members, et)
		}
		if len(members) == 0 {
			return nil, cronerr.Argument("ExecutionTime list cannot be null or empty")
		}
		return &compositeExecutionTime{members}, nil
	}
	return neverExecutionTime{}, nil
}

// ExecutionDates returns the executions from start (exclusive) to end (inclusive).
func ExecutionDates(et ExecutionTime, start, end javatime.ZonedDateTime) ([]javatime.ZonedDateTime, error) {
	if end.Equal(start) || end.IsBefore(start) {
		return nil, cronerr.Argument("endDate should take place later in time than startDate")
	}
	var executions []javatime.ZonedDateTime
	next, ok, err := et.NextExecution(start)
	for err == nil && ok && (next.IsBefore(end) || next.Equal(end)) {
		executions = append(executions, next)
		next, ok, err = et.NextExecution(next)
	}
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// CountExecutions returns the number of executions from start (exclusive) to end (inclusive).
func CountExecutions(et ExecutionTime, start, end javatime.ZonedDateTime) (int, error) {
	dates, err := ExecutionDates(et, start, end)
	return len(dates), err
}

// neverExecutionTime is the schedule of a cron that never runs on its own, such as @reboot.
type neverExecutionTime struct{}

func (neverExecutionTime) NextExecution(javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	return javatime.ZonedDateTime{}, false, nil
}

func (neverExecutionTime) TimeToNextExecution(javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	return javatime.Duration{}, false, nil
}

func (neverExecutionTime) LastExecution(javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	return javatime.ZonedDateTime{}, false, nil
}

func (neverExecutionTime) TimeFromLastExecution(javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	return javatime.Duration{}, false, nil
}

func (neverExecutionTime) IsMatch(javatime.ZonedDateTime) (bool, error) { return false, nil }

// compositeExecutionTime runs on the union of its members' schedules.
type compositeExecutionTime struct {
	members []ExecutionTime
}

func (c *compositeExecutionTime) pick(date javatime.ZonedDateTime,
	step func(ExecutionTime, javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error), latest bool) (javatime.ZonedDateTime, bool, error) {
	var found []javatime.ZonedDateTime
	for _, m := range c.members {
		d, ok, err := step(m, date)
		if err != nil {
			return javatime.ZonedDateTime{}, false, err
		}
		if ok {
			found = append(found, d)
		}
	}
	if len(found) == 0 {
		return javatime.ZonedDateTime{}, false, nil
	}
	slices.SortStableFunc(found, func(a, b javatime.ZonedDateTime) int {
		if latest {
			return b.Compare(a)
		}
		return a.Compare(b)
	})
	return found[0], true, nil
}

func (c *compositeExecutionTime) NextExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	return c.pick(date, ExecutionTime.NextExecution, false)
}

func (c *compositeExecutionTime) TimeToNextExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	next, ok, err := c.NextExecution(date)
	if err != nil || !ok {
		return javatime.Duration{}, ok, err
	}
	return javatime.Between(date, next), true, nil
}

func (c *compositeExecutionTime) LastExecution(date javatime.ZonedDateTime) (javatime.ZonedDateTime, bool, error) {
	return c.pick(date, ExecutionTime.LastExecution, true)
}

func (c *compositeExecutionTime) TimeFromLastExecution(date javatime.ZonedDateTime) (javatime.Duration, bool, error) {
	last, ok, err := c.LastExecution(date)
	if err != nil || !ok {
		return javatime.Duration{}, ok, err
	}
	return javatime.Between(last, date), true, nil
}

func (c *compositeExecutionTime) IsMatch(date javatime.ZonedDateTime) (bool, error) {
	matched := false
	for _, m := range c.members {
		ok, err := m.IsMatch(date)
		if err != nil {
			return false, err
		}
		matched = matched || ok
	}
	return matched, nil
}

// constraintsOf returns a field's constraints in the cron's definition, or the field's default constraints.
func constraintsOf(cron model.Cron, name model.CronFieldName) *model.FieldConstraints {
	if fd := cron.Definition().FieldDefinition(name); fd != nil {
		return fd.Constraints
	}
	return model.NewFieldConstraintsBuilder().ForField(name).Build()
}

func timeNodeFor(f *model.CronField, lower, higher int) (*timeNode, error) {
	g, err := forCronField(f)
	if err != nil {
		return nil, err
	}
	values, err := candidates(g, lower, higher)
	if err != nil {
		return nil, err
	}
	return newTimeNode(values)
}

func forSingleCron(cron *model.SingleCron) (*singleExecutionTime, error) {
	var seconds, minutes, hours, months *timeNode
	var daysOfWeek, daysOfMonth, daysOfYear *model.CronField
	for _, f := range cron.FieldsInOrder() {
		var err error
		switch f.Field {
		case model.Second:
			seconds, err = timeNodeFor(f, 0, 59)
		case model.Minute:
			minutes, err = timeNodeFor(f, 0, 59)
		case model.Hour:
			hours, err = timeNodeFor(f, 0, 23)
		case model.DayOfWeek:
			daysOfWeek = f
		case model.DayOfMonth:
			daysOfMonth = f
		case model.Month:
			months, err = timeNodeFor(f, 1, 12)
		case model.Year:
			_, err = forCronField(f)
		case model.DayOfYear:
			daysOfYear = f
		}
		if err != nil {
			return nil, err
		}
	}
	lowest := func(name model.CronFieldName, lower, higher int) (*timeNode, error) {
		return timeNodeFor(model.NewCronField(name, model.NewOn(lower), constraintsOf(cron, name)), lower, higher)
	}
	always := func(name model.CronFieldName, lower, higher int) (*timeNode, error) {
		return timeNodeFor(model.NewCronField(name, model.AlwaysExpression(), constraintsOf(cron, name)), lower, higher)
	}
	pick := func(assigned bool, name model.CronFieldName, lower, higher int) (*timeNode, error) {
		if assigned {
			return always(name, lower, higher)
		}
		return lowest(name, lower, higher)
	}
	var err error
	lowestAssigned := false
	if seconds == nil {
		if seconds, err = lowest(model.Second, 0, 59); err != nil {
			return nil, err
		}
	} else {
		lowestAssigned = true
	}
	if minutes == nil {
		if minutes, err = pick(lowestAssigned, model.Minute, 0, 59); err != nil {
			return nil, err
		}
	} else {
		lowestAssigned = true
	}
	if hours == nil {
		if hours, err = pick(lowestAssigned, model.Hour, 0, 23); err != nil {
			return nil, err
		}
	} else {
		lowestAssigned = true
	}
	if daysOfMonth == nil {
		var e model.FieldExpression = model.NewOn(1)
		if lowestAssigned {
			e = model.AlwaysExpression()
		}
		daysOfMonth = model.NewCronField(model.DayOfMonth, e, constraintsOf(cron, model.DayOfMonth))
	} else {
		lowestAssigned = true
	}
	if daysOfWeek == nil {
		var e model.FieldExpression = model.NewOn(1)
		if lowestAssigned {
			e = model.AlwaysExpression()
		}
		daysOfWeek = model.NewCronField(model.DayOfWeek, e, constraintsOf(cron, model.DayOfWeek))
	} else {
		lowestAssigned = true
	}
	if months == nil {
		if months, err = pick(lowestAssigned, model.Month, 1, 12); err != nil {
			return nil, err
		}
	}
	if daysOfYear == nil {
		var e model.FieldExpression = model.AlwaysExpression()
		if lowestAssigned {
			e = model.QuestionMarkExpression()
		}
		daysOfYear = model.NewCronField(model.DayOfYear, e, constraintsOf(cron, model.DayOfYear))
	}
	return newSingleExecutionTime(cron.Definition(), cron.Field(model.Year), daysOfWeek, daysOfMonth, daysOfYear, months, hours,
		minutes, seconds)
}
