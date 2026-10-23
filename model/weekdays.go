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
// Ported to Go from com/cronutils/model/field/expression/Weekdays.java.

package model

// Weekday names a day of the week, numbered 1 for Monday to 7 for Sunday.
type Weekday int

// The days of the week.
const (
	Monday Weekday = iota + 1
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

// Number returns the day's number in the crontab numbering.
func (w Weekday) Number() int {
	return w.NumberIn(CrontabWeekDay)
}

// NumberIn returns the day's number in the given numbering.
func (w Weekday) NumberIn(weekDay WeekDay) int {
	return WeekDayMapping(CrontabWeekDay, weekDay, int(w))
}

// NumberFor returns the day's number in the day-of-week field of a definition.
func (w Weekday) NumberFor(definition *CronDefinition) int {
	return w.NumberIn(*definition.FieldDefinition(DayOfWeek).MondayDoW)
}
