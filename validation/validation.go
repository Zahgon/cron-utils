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
// Ported to Go from com/cronutils/validation/{Cron,CronValidator}.java.

// Package validation checks cron expressions held in strings, for use in input validation.
package validation

import (
	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
)

// CronValidator checks that strings are valid crons of one dialect.
type CronValidator struct {
	Type model.CronType
}

// IsValid reports whether value is a valid cron; when it is not, the reason is returned. An absent value (nil) is
// valid. Failures other than an invalid expression are returned as errors.
func (v CronValidator) IsValid(value *string) (bool, string, error) {
	if value == nil {
		return true, "", nil
	}
	p := parser.New(model.InstanceDefinitionFor(v.Type))
	cron, err := p.Parse(*value)
	if err == nil {
		err = cron.Validate()
	}
	if err == nil {
		return true, "", nil
	}
	if cronerr.Is(err, cronerr.IllegalArgument) {
		return false, err.Error(), nil
	}
	return false, "", err
}
