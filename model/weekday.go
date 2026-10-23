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
// Ported to Go from com/cronutils/mapper/{WeekDay,ConstantsMapper}.java.

package model

import "github.com/jmrozanec/cron-utils/cronerr"

// WeekDay describes how a cron dialect numbers the days of the week: the number of Monday and whether the numbering
// starts at zero.
type WeekDay struct {
	mondayDoWValue int
	firstDayZero   bool
}

// NewWeekDay returns a day-of-week numbering; mondayDoWValue must not be negative.
func NewWeekDay(mondayDoWValue int, firstDayZero bool) (WeekDay, error) {
	if mondayDoWValue < 0 {
		return WeekDay{}, cronerr.Argument("Monday Day of Week value must be greater or equal to zero")
	}
	return WeekDay{mondayDoWValue, firstDayZero}, nil
}

func mustWeekDay(mondayDoWValue int, firstDayZero bool) WeekDay {
	return WeekDay{mondayDoWValue, firstDayZero}
}

// Predefined day-of-week numberings.
var (
	// QuartzWeekDay numbers Sunday 1 to Saturday 7.
	QuartzWeekDay = mustWeekDay(2, false)
	// Java8WeekDay numbers Monday 1 to Sunday 7.
	Java8WeekDay = mustWeekDay(1, false)
	// CrontabWeekDay numbers Sunday 0 to Saturday 6.
	CrontabWeekDay = mustWeekDay(1, true)
)

// MondayDoWValue returns the number of Monday.
func (w WeekDay) MondayDoWValue() int { return w.mondayDoWValue }

// IsFirstDayZero reports whether the numbering starts at zero.
func (w WeekDay) IsFirstDayZero() bool { return w.firstDayZero }

// MapTo converts a day number of this numbering to the target numbering.
func (w WeekDay) MapTo(dayOfWeek int, target WeekDay) int {
	if w.firstDayZero && target.firstDayZero {
		return bothSameStartOfRange(0, 6, w, target, dayOfWeek)
	}
	if !w.firstDayZero && !target.firstDayZero {
		return bothSameStartOfRange(1, 7, w, target, dayOfWeek)
	}
	if target.firstDayZero {
		return w.MapTo(dayOfWeek, mustWeekDay(target.mondayDoWValue+1, false)) - 1
	}
	return w.MapTo(dayOfWeek, mustWeekDay(target.mondayDoWValue-1, true))%7 + 1
}

func bothSameStartOfRange(startRange, endRange int, source, target WeekDay, value int) int {
	diff := target.mondayDoWValue - source.mondayDoWValue
	result := value
	if diff == 0 {
		return value
	}
	if diff < 0 {
		result = value + diff
		distanceToStartRange := startRange - result
		if result < startRange {
			result = endRange + 1 - distanceToStartRange
		}
	}
	if diff > 0 {
		result = value + diff
		if result > endRange {
			result -= endRange
		}
	}
	return result
}

// WeekDayMapping converts a day number from the source numbering to the target numbering.
func WeekDayMapping(source, target WeekDay, weekday int) int {
	return source.MapTo(weekday, target)
}
