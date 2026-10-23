// Copyright 2019 fahmpeermoh
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
// Ported to Go from com/cronutils/converter/{CronConverter,BaseCronTransformer,CronToCalendarTransformer,
// CalendarToCronTransformer}.java.

// Package converter shifts the numeric fields of a five-field cron from one time-zone to another, using the current
// date as the reference day.
package converter

import (
	"strconv"
	"strings"
	"time"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/internal/javalang"
)

// calendar is a date-time in a zone whose fields can be set one by one, out-of-range values carrying over.
type calendar struct {
	t time.Time
}

// Positions of the fields of a five-field cron.
const (
	minutePos = iota
	hourPos
	dayOfMonthPos
	monthPos
	dayOfWeekPos
)

func (c *calendar) set(pos, value int) {
	t := c.t
	switch pos {
	case minutePos:
		c.t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), value, t.Second(), t.Nanosecond(), t.Location())
	case hourPos:
		c.t = time.Date(t.Year(), t.Month(), t.Day(), value, t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	case dayOfMonthPos:
		c.t = time.Date(t.Year(), t.Month(), value, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	case monthPos:
		// The cron month is stored as the calendar's zero-based month, as the original converter does.
		c.t = time.Date(t.Year(), time.Month(value+1), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	case dayOfWeekPos:
		// Calendar days of the week run from 1 (Sunday) to 7 (Saturday) within a Sunday-first week.
		current := int(t.Weekday()) + 1
		c.t = t.AddDate(0, 0, value-current)
	}
}

func (c *calendar) get(pos int) int {
	switch pos {
	case minutePos:
		return c.t.Minute()
	case hourPos:
		return c.t.Hour()
	case dayOfMonthPos:
		return c.t.Day()
	case monthPos:
		return int(c.t.Month()) - 1
	}
	return int(c.t.Weekday()) + 1
}

// CronConverter converts the numeric fields of a cron between time-zones. Its steps chain (Using, From, To); the
// first failure is kept and returned by Convert.
type CronConverter struct {
	cronParts []string
	from      *calendar
	now       func() time.Time
	err       error
}

// New returns a converter that takes the current time as the reference day.
func New() *CronConverter {
	return &CronConverter{now: time.Now}
}

// Using sets the cron to convert; its fields are separated by single spaces.
func (c *CronConverter) Using(cronExpression string) *CronConverter {
	c.cronParts = javalang.Split(cronExpression, " ")
	return c
}

// From places the cron's numeric fields in the source zone.
func (c *CronConverter) From(zone *time.Location) *CronConverter {
	if c.err != nil {
		return c
	}
	if zone == nil {
		c.err = cronerr.New(cronerr.NullValue, "source zone must not be nil")
		return c
	}
	c.from = &calendar{t: c.now().In(zone)}
	for i, part := range c.cronParts {
		if i <= dayOfWeekPos && javalang.IsNumeric(part) {
			v, err := javalang.ParseInt(part)
			if err != nil {
				c.err = err
				return c
			}
			c.from.set(i, v)
		}
	}
	return c
}

// To reads the cron's numeric fields back in the target zone.
func (c *CronConverter) To(zone *time.Location) *CronConverter {
	if c.err != nil {
		return c
	}
	if c.from == nil {
		c.err = cronerr.New(cronerr.IllegalState, "From must be called before To")
		return c
	}
	if zone == nil {
		c.err = cronerr.New(cronerr.NullValue, "target zone must not be nil")
		return c
	}
	to := &calendar{t: c.from.t.In(zone)}
	for i, part := range c.cronParts {
		if i <= dayOfWeekPos && javalang.IsNumeric(part) {
			c.cronParts[i] = strconv.Itoa(to.get(i))
		}
	}
	return c
}

// Convert returns the converted cron, or the first error of the chain.
func (c *CronConverter) Convert() (string, error) {
	if c.err != nil {
		return "", c.err
	}
	return strings.Join(c.cronParts, " "), nil
}
