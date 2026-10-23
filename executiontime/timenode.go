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
// Ported to Go from com/cronutils/model/time/{TimeNode,NearestValue}.java.

package executiontime

import (
	"slices"

	"github.com/jmrozanec/cron-utils/cronerr"
)

// nearestValue is a value found by moving through a time node, with the number of times the search wrapped
// around the node's values.
type nearestValue struct {
	value, shifts int
}

// timeNode is the sorted, non-empty list of values a field may take.
type timeNode struct {
	values []int
}

func newTimeNode(values []int) (*timeNode, error) {
	if len(values) == 0 {
		return nil, cronerr.Argument("Values must not be empty")
	}
	v := slices.Clone(values)
	slices.Sort(v)
	return &timeNode{v}, nil
}

func (n *timeNode) contains(v int) bool { return slices.Contains(n.values, v) }

func (n *timeNode) first() int { return n.values[0] }

func (n *timeNode) last() int { return n.values[len(n.values)-1] }

// nextValue finds the value reached by moving forward shifts positions from reference (from the first value above
// reference when reference is not a value).
func (n *timeNode) nextValue(reference, shifts int) nearestValue {
	return nearest(n.values, reference, shifts, func(v, ref int) bool { return v > ref })
}

// previousValue finds the value reached by moving backward shifts positions from reference.
func (n *timeNode) previousValue(reference, shifts int) nearestValue {
	reversed := slices.Clone(n.values)
	slices.Reverse(reversed)
	return nearest(reversed, reference, shifts, func(v, ref int) bool { return v < ref })
}

func nearest(values []int, reference, shiftsToApply int, beyond func(v, ref int) bool) nearestValue {
	index := 0
	shift := 0
	if !slices.Contains(values, reference) {
		found := false
		for i, v := range values {
			if beyond(v, reference) {
				index = i
				shiftsToApply--
				found = true
				break
			}
		}
		if !found {
			shift++
		}
	} else {
		index = slices.Index(values, reference)
	}
	value := values[index]
	for j := 0; j < shiftsToApply; j++ {
		value = valueFromList(values, index+1, &shift)
		index = slices.Index(values, value)
	}
	return nearestValue{value, shift}
}

func valueFromList(values []int, index int, shift *int) int {
	for index < 0 || index >= len(values) {
		if index < 0 {
			index += len(values)
		} else {
			index -= len(values)
		}
		*shift++
	}
	return values[index]
}
