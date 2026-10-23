// Copyright 2014 jmrozanec
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
// Ported to Go from com/cronutils/utils/{DateUtils,WeekendPolicy,CronFrequencyComparator}.java.

// Package utils counts working days between dates and compares crons by how often they run.
package utils

import (
	"slices"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

// WeekendPolicy names the two days of a weekend.
type WeekendPolicy int

// The weekend policies.
const (
	ThursdayFriday WeekendPolicy = iota
	FridaySaturday
	SaturdaySunday
)

// DaysToWeekend returns the number of working days from date to the next weekend.
func DaysToWeekend(policy WeekendPolicy, date javatime.ZonedDateTime) int {
	dow := date.DayOfWeek()
	switch policy {
	case ThursdayFriday:
		if dow < 4 {
			return 4 - dow
		}
		if dow > 5 {
			return 7 - dow + 4
		}
	case FridaySaturday:
		if dow < 5 {
			return 6 - dow
		}
		if dow > 6 {
			return 7 - dow + 5
		}
	case SaturdaySunday:
		if dow < 6 {
			return max(0, 6-dow)
		}
	}
	return 0
}

// DaysFromWeekend returns the number of working days from the last weekend to date.
func DaysFromWeekend(policy WeekendPolicy, date javatime.ZonedDateTime) int {
	dow := date.DayOfWeek()
	switch policy {
	case ThursdayFriday:
		if dow < 4 {
			return dow
		}
		if dow > 5 {
			return dow - 5
		}
	case FridaySaturday:
		if dow < 5 {
			return dow
		}
		if dow > 6 {
			return dow - 6
		}
	case SaturdaySunday:
		if dow < 6 {
			return dow
		}
	}
	return 0
}

// WorkdaysCountFor returns the number of working days in the days days from start.
func WorkdaysCountFor(start javatime.ZonedDateTime, days int, holidays []javatime.ZonedDateTime, policy WeekendPolicy) (int, error) {
	return WorkdaysCount(start, start.PlusDays(int64(days)), holidays, policy)
}

// WorkdaysCount returns the number of working days from start to end, less the holidays in that range. The holidays
// slice is sorted in place. The holiday search has a defect for some holiday lists: it can select a range that
// ends before it starts (reported as an error) or never settle (reported as an error too).
func WorkdaysCount(start, end javatime.ZonedDateTime, holidays []javatime.ZonedDateTime, policy WeekendPolicy) (int, error) {
	slices.SortStableFunc(holidays, javatime.ZonedDateTime.Compare)
	holidays, err := holidaysInRange(start, end, holidays)
	if err != nil {
		return 0, err
	}
	daysToWeekend := DaysToWeekend(policy, start)
	daysFromWeekend := DaysFromWeekend(policy, end)
	daysBetween := int(javatime.Between(start, end).ToDays()) + 1
	weekdays := (daysBetween-daysToWeekend-daysFromWeekend-2)/7*5 - len(holidays)
	return weekdays + daysToWeekend + daysFromWeekend, nil
}

func holidaysInRange(start, end javatime.ZonedDateTime, holidays []javatime.ZonedDateTime) ([]javatime.ZonedDateTime, error) {
	if len(holidays) == 0 {
		return holidays, nil
	}
	from, err := findIndex(0, len(holidays)-1, start, holidays)
	if err != nil {
		return nil, err
	}
	to, err := findIndex(0, len(holidays)-1, end, holidays)
	if err != nil {
		return nil, err
	}
	if from > to+1 {
		return nil, cronerr.Argument("fromIndex(%d) > toIndex(%d)", from, to+1)
	}
	return holidays[from : to+1], nil
}

// findIndex narrows [startIdx, endIdx] towards date. The pivot is half the width of the range rather than its
// middle, so the search can revisit the same range forever; that is reported as an error.
func findIndex(startIdx, endIdx int, date javatime.ZonedDateTime, holidays []javatime.ZonedDateTime) (int, error) {
	seen := map[[2]int]bool{}
	for startIdx != endIdx {
		if seen[[2]int{startIdx, endIdx}] {
			return 0, cronerr.New(cronerr.IllegalState, "holiday search does not terminate")
		}
		seen[[2]int{startIdx, endIdx}] = true
		pivot := (endIdx - startIdx) / 2
		if holidays[pivot].Equal(date) {
			return pivot, nil
		}
		if holidays[pivot].IsBefore(date) {
			startIdx = pivot + 1
		} else {
			endIdx = pivot
		}
	}
	return startIdx, nil
}

// CronFrequencyComparator orders crons by their number of executions in a window.
type CronFrequencyComparator struct {
	Start, End javatime.ZonedDateTime
}

// Compare returns the number of executions of a less those of b in the window.
func (c CronFrequencyComparator) Compare(a, b model.Cron) (int, error) {
	count := func(cron model.Cron) (int, error) {
		et, err := executiontime.ForCron(cron)
		if err != nil {
			return 0, err
		}
		return executiontime.CountExecutions(et, c.Start, c.End)
	}
	na, err := count(a)
	if err != nil {
		return 0, err
	}
	nb, err := count(b)
	if err != nil {
		return 0, err
	}
	return na - nb, nil
}
