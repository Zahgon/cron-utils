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
// Ported to Go from the overlap methods of com/cronutils/model/{SingleCron,CompositeCron,RebootCron}.java.

package executiontime

import (
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/model"
)

// overlapChecks is the number of upcoming executions of each cron checked against the other.
const overlapChecks = 10

// Overlap reports whether two crons share an execution among the next ten executions of either one after now. A
// composite cron overlaps when any of its members does; @reboot only overlaps @reboot.
func Overlap(a, b model.Cron, now javatime.ZonedDateTime) (bool, error) {
	switch c := a.(type) {
	case *model.RebootCron:
		_, reboot := b.(*model.RebootCron)
		return reboot, nil
	case *model.CompositeCron:
		for _, member := range c.Crons() {
			ok, err := Overlap(member, b, now)
			if err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	}
	etA, err := ForCron(a)
	if err != nil {
		return false, err
	}
	etB, err := ForCron(b)
	if err != nil {
		return false, err
	}
	nextA, okA, err := etA.NextExecution(now)
	if err != nil {
		return false, err
	}
	nextB, okB, err := etB.NextExecution(now)
	if err != nil {
		return false, err
	}
	if !okA || !okB {
		return false, nil
	}
	for _, pair := range []struct {
		first javatime.ZonedDateTime
		steps ExecutionTime
		other ExecutionTime
	}{{nextA, etA, etB}, {nextB, etB, etA}} {
		check := pair.first
		for i := 0; i < overlapChecks; i++ {
			match, err := pair.other.IsMatch(check)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
			n, ok, err := pair.steps.NextExecution(check)
			if err != nil {
				return false, err
			}
			if !ok {
				break
			}
			check = n
		}
	}
	return false, nil
}
