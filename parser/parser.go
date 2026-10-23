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
// Ported to Go from com/cronutils/parser/CronParser.java.

// Package parser parses cron expressions against a cron definition.
package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jmrozanec/cron-utils/builder"
	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/internal/javalang"
	"github.com/jmrozanec/cron-utils/model"
)

// CronParser parses expressions of one cron definition. It is safe for concurrent use.
type CronParser struct {
	definition  *model.CronDefinition
	expressions map[int][]*CronParserField
}

// New returns a parser for the definition. When the last fields are optional, expressions may leave them out.
func New(definition *model.CronDefinition) *CronParser {
	p := &CronParser{definition: definition, expressions: map[int][]*CronParserField{}}
	var sorted []*CronParserField
	for _, fd := range definition.FieldDefinitions() {
		sorted = append(sorted, NewCronParserField(fd.FieldName, fd.Constraints, fd.Optional))
	}
	current := sorted
	for len(current) > 0 && current[len(current)-1].Optional {
		current = current[:len(current)-1]
		p.expressions[len(current)] = current
	}
	p.expressions[len(sorted)] = sorted
	return p
}

// Definition returns the parser's cron definition.
func (p *CronParser) Definition() *model.CronDefinition { return p.definition }

var nicknames = map[string]struct {
	nickname model.CronNickname
	build    func(*model.CronDefinition) (model.Cron, error)
}{
	"@yearly":   {model.Yearly, builder.Yearly},
	"@annually": {model.Annually, builder.Annually},
	"@monthly":  {model.Monthly, builder.Monthly},
	"@weekly":   {model.Weekly, builder.Weekly},
	"@daily":    {model.Daily, builder.Daily},
	"@midnight": {model.Midnight, builder.Midnight},
	"@hourly":   {model.Hourly, builder.Hourly},
	"@reboot": {model.Reboot, func(d *model.CronDefinition) (model.Cron, error) {
		return builder.Reboot(d), nil
	}},
}

// Parse parses an expression. Runs of white space separate the fields; "a||b" is the union of the crons a and b,
// and "a|b" inside fields lists alternative values field by field. The @-nicknames the definition supports stand
// for their crons.
func (p *CronParser) Parse(expression string) (model.Cron, error) {
	replaced := javalang.Trim(javalang.CollapseSpaces(expression))
	if replaced == "" {
		return nil, cronerr.Argument("Empty expression!")
	}
	if strings.HasPrefix(expression, "@") {
		if !p.definition.HasNicknames() {
			return nil, cronerr.Argument("Nicknames not supported!")
		}
		if n, ok := nicknames[expression]; ok {
			if !p.definition.SupportsNickname(n.nickname) {
				return nil, cronerr.Argument("Nickname %s not supported!", expression)
			}
			return n.build(p.definition)
		}
	}
	if strings.Contains(expression, "||") {
		var crons []model.Cron
		for _, part := range javalang.Split(expression, "||") {
			c, err := p.Parse(part)
			if err != nil {
				return nil, err
			}
			crons = append(crons, c)
		}
		return model.NewCompositeCron(crons)
	}
	if strings.Contains(expression, "|") {
		return p.parseAlternatives(expression)
	}
	parts := strings.Split(javalang.ToUpper(replaced), " ")
	for _, part := range parts {
		if strings.HasSuffix(part, ",") {
			return nil, cronerr.Argument("Invalid field value! Trailing commas not permitted! '%s'", part)
		}
	}
	fields, ok := p.expressions[len(parts)]
	if !ok {
		return nil, cronerr.Argument("Cron expression contains %d parts but we expect one of %s", len(parts), p.lengths())
	}
	results := make([]*model.CronField, 0, len(parts))
	for j, part := range parts {
		f, err := fields[j].Parse(part)
		if err != nil {
			return nil, wrapParseError(err)
		}
		results = append(results, f)
	}
	cron := model.NewSingleCron(p.definition, results)
	if err := cron.Validate(); err != nil {
		return nil, wrapParseError(err)
	}
	return cron, nil
}

// parseAlternatives expands "a|b" field values into one cron per alternative.
func (p *CronParser) parseAlternatives(expression string) (model.Cron, error) {
	tokens := javalang.SplitSpaces(expression)
	count := 0
	for _, s := range tokens {
		count = max(count, len(javalang.Split(s, "|")))
	}
	expressions := make([]string, 0, count)
	for j := 0; j < count; j++ {
		var b strings.Builder
		for _, s := range tokens {
			if strings.Contains(s, "|") {
				alternatives := javalang.Split(s, "|")
				if j >= len(alternatives) {
					return nil, cronerr.New(cronerr.IndexOutOfBounds, "Index %d out of bounds for length %d", j,
						len(alternatives))
				}
				fmt.Fprintf(&b, "%s ", alternatives[j])
			} else {
				fmt.Fprintf(&b, "%s ", s)
			}
		}
		expressions = append(expressions, javalang.Trim(b.String()))
	}
	crons := make([]model.Cron, 0, count)
	for _, e := range expressions {
		c, err := p.Parse(e)
		if err != nil {
			return nil, err
		}
		crons = append(crons, c)
	}
	return model.NewCompositeCron(crons)
}

// wrapParseError prefixes an invalid-argument error with the parser's message; other errors pass unchanged.
func wrapParseError(err error) error {
	if cronerr.Is(err, cronerr.IllegalArgument) {
		return cronerr.Wrap(cronerr.IllegalArgument, err, "Failed to parse cron expression. %s", err.Error())
	}
	return err
}

func (p *CronParser) lengths() string {
	keys := make([]int, 0, len(p.expressions))
	for k := range p.expressions {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprint(k)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
