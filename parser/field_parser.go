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
// Ported to Go from com/cronutils/parser/{FieldParser,CronParserField}.java.

package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/internal/javalang"
	"github.com/jmrozanec/cron-utils/model"
)

var (
	lPattern      = regexp.MustCompile(`(?i)[0-9]L`)
	wPattern      = regexp.MustCompile(`(?i)[0-9]W`)
	randomPattern = regexp.MustCompile(`^(?:~|[0-9]*~[0-9]*(/[0-9]+)?)$`)
)

// FieldParser parses the expression of one field.
type FieldParser struct {
	constraints *model.FieldConstraints
}

// NewFieldParser returns a parser for a field with the given constraints.
func NewFieldParser(constraints *model.FieldConstraints) *FieldParser {
	return &FieldParser{constraints: constraints}
}

// Parse parses one field expression: *, ?, a value, a range, a step, a list or a special-character form.
func (p *FieldParser) Parse(expression string) (model.FieldExpression, error) {
	if !javalang.ContainsAny(expression, '/', '-', ',') {
		if strings.Contains(expression, "?") && !p.constraints.SupportsSpecialChar(model.QuestionMarkChar) {
			return nil, cronerr.Argument("Invalid expression: %s", expression)
		}
		return p.noSpecialCharsNorStar(expression)
	}
	if randomPattern.MatchString(expression) {
		return p.parseRandom(expression)
	}
	if array := javalang.Split(expression, ","); len(array) > 1 {
		return p.commaSplitResult(array)
	}
	splitted := javalang.Split(expression, "-")
	if strings.Contains(expression, "-") && len(splitted) != 2 {
		return nil, cronerr.Argument("Missing values for range: %s", expression)
	}
	if len(splitted) > 0 && strings.EqualFold(splitted[0], "L") {
		daysBefore, err := p.mapToIntegerFieldValue(splitted[1])
		if err != nil {
			return nil, err
		}
		return p.parseOnWithL(splitted[0], daysBefore)
	}
	return p.dashSplitResult(expression, splitted)
}

func (p *FieldParser) dashSplitResult(expression string, between []string) (model.FieldExpression, error) {
	if len(between) > 1 {
		return p.parseBetween(between)
	}
	return p.slashSplit(expression, javalang.Split(expression, "/"))
}

func (p *FieldParser) parseRandom(expression string) (model.FieldExpression, error) {
	if expression == "~" {
		return &model.RandomExpression{}, nil
	}
	parts := javalang.SplitKeepTrailing(expression, "~")
	if len(parts) < 2 {
		return &model.RandomExpression{}, nil
	}
	r := &model.RandomExpression{}
	if parts[0] != "" {
		from, err := javalang.ParseInt(parts[0])
		if err != nil {
			return nil, err
		}
		r.From = &from
	}
	toAndStep := parts[1]
	if strings.Contains(toAndStep, "/") {
		toStep := javalang.Split(toAndStep, "/")
		if toStep[0] != "" {
			to, err := javalang.ParseInt(toStep[0])
			if err != nil {
				return nil, err
			}
			r.To = &to
		}
		step, err := javalang.ParseInt(toStep[1])
		if err != nil {
			return nil, err
		}
		r.Step = &step
	} else if toAndStep != "" {
		to, err := javalang.ParseInt(toAndStep)
		if err != nil {
			return nil, err
		}
		r.To = &to
	}
	return r, nil
}

func (p *FieldParser) commaSplitResult(array []string) (model.FieldExpression, error) {
	and := model.NewAnd()
	for _, exp := range array {
		e, err := p.Parse(exp)
		if err != nil {
			return nil, err
		}
		and.And(e)
	}
	return and, nil
}

func (p *FieldParser) slashSplit(expression string, values []string) (model.FieldExpression, error) {
	switch len(values) {
	case 2:
		return p.asteriskOrEmpty(values[0], values[1])
	case 1:
		return nil, cronerr.Argument("Missing steps for expression: %s", expression)
	}
	return nil, cronerr.Argument("Invalid expression: %s", expression)
}

func (p *FieldParser) asteriskOrEmpty(start, value string) (model.FieldExpression, error) {
	trimmed := javalang.Trim(start)
	if trimmed == "*" && value == "1" {
		return p.noSpecialCharsNorStar(start)
	}
	if trimmed == "*" {
		period, err := javalang.ParseInt(value)
		if err != nil {
			return nil, err
		}
		return model.NewEvery(model.IntegerFieldValue(period)), nil
	}
	time, err := p.mapToIntegerFieldValue(start)
	if err != nil {
		return nil, err
	}
	period, err := javalang.ParseInt(value)
	if err != nil {
		return nil, err
	}
	return model.NewEveryOf(&model.On{Time: time, SpecialChar: model.SpecialCharFieldValue(model.None), Nth: model.DefaultNth},
		model.IntegerFieldValue(period)), nil
}

func (p *FieldParser) noSpecialCharsNorStar(expression string) (model.FieldExpression, error) {
	switch expression {
	case "*":
		return model.AlwaysExpression(), nil
	case "?":
		return model.QuestionMarkExpression(), nil
	}
	return p.parseOn(expression)
}

func (p *FieldParser) parseBetween(array []string) (model.FieldExpression, error) {
	if array[0] == "" || array[1] == "" {
		return nil, cronerr.Argument("Invalid expression! Expression: %s-%s does not describe a range. Negative numbers are not allowed.",
			array[0], array[1])
	}
	if strings.Contains(array[1], "/") {
		every := javalang.Split(array[1], "/")
		from, err := p.mapValue(array[0])
		if err != nil {
			return nil, err
		}
		if len(every) == 0 {
			return nil, indexError(0, 0)
		}
		to, err := p.mapValue(every[0])
		if err != nil {
			return nil, err
		}
		if len(every) < 2 {
			return nil, indexError(1, len(every))
		}
		period, err := p.mapToIntegerFieldValue(every[1])
		if err != nil {
			return nil, err
		}
		return model.NewEveryOf(model.NewBetween(from, to), period), nil
	}
	from, err := p.mapValue(array[0])
	if err != nil {
		return nil, err
	}
	to, err := p.mapValue(array[1])
	if err != nil {
		return nil, err
	}
	return model.NewBetween(from, to), nil
}

func indexError(index, length int) error {
	return cronerr.New(cronerr.IndexOutOfBounds, "Index %d out of bounds for length %d", index, length)
}

func (p *FieldParser) parseOn(exp string) (model.FieldExpression, error) {
	switch {
	case exp == "?":
		return model.NewOnSpecial(model.DefaultNth, model.QuestionMarkChar, model.DefaultNth), nil
	case strings.Contains(exp, "#"):
		return p.parseOnWithHash(exp)
	case strings.Contains(exp, "LW"):
		return p.parseOnWithLW(exp)
	case lPattern.MatchString(exp) || strings.EqualFold(exp, "L"):
		return p.parseOnWithL(exp, model.DefaultNth)
	case wPattern.MatchString(exp):
		time, err := p.mapToIntegerFieldValue(strings.ReplaceAll(exp, "W", ""))
		if err != nil {
			return nil, err
		}
		return &model.On{Time: time, SpecialChar: model.SpecialCharFieldValue(model.W), Nth: model.DefaultNth}, nil
	}
	time, err := p.mapToIntegerFieldValue(exp)
	if err != nil {
		return nil, err
	}
	return &model.On{Time: time, SpecialChar: model.SpecialCharFieldValue(model.None), Nth: model.DefaultNth}, nil
}

func (p *FieldParser) parseOnWithHash(exp string) (model.FieldExpression, error) {
	if !p.constraints.SupportsSpecialChar(model.Hash) {
		return nil, cronerr.Argument("Invalid expression: %s", exp)
	}
	array := javalang.Split(exp, "#")
	if len(array) == 0 {
		return nil, cronerr.Argument("Invalid Position of # Character!")
	}
	if len(array) < 2 {
		return nil, indexError(1, len(array))
	}
	nth, err := p.mapToIntegerFieldValue(array[1])
	if err != nil {
		return nil, err
	}
	if array[0] == "" {
		return nil, cronerr.Argument("Time should be specified!")
	}
	time, err := p.mapToIntegerFieldValue(array[0])
	if err != nil {
		return nil, err
	}
	return &model.On{Time: time, SpecialChar: model.SpecialCharFieldValue(model.Hash), Nth: nth}, nil
}

func (p *FieldParser) parseOnWithLW(exp string) (model.FieldExpression, error) {
	if rest := strings.ReplaceAll(exp, "LW", ""); rest != "" {
		return nil, cronerr.Argument("Expected: LW, found: %s", rest)
	}
	return model.NewOnSpecial(model.DefaultNth, model.LW, model.DefaultNth), nil
}

func (p *FieldParser) parseOnWithL(exp string, daysBefore model.IntegerFieldValue) (model.FieldExpression, error) {
	time := model.IntegerFieldValue(model.DefaultNth)
	if rest := strings.ReplaceAll(exp, "L", ""); rest != "" {
		v, err := p.mapToIntegerFieldValue(rest)
		if err != nil {
			return nil, err
		}
		time = v
	}
	return &model.On{Time: time, SpecialChar: model.SpecialCharFieldValue(model.L), Nth: daysBefore}, nil
}

func (p *FieldParser) mapToIntegerFieldValue(s string) (model.IntegerFieldValue, error) {
	v, err := p.stringToInt(s)
	if err != nil {
		return 0, err
	}
	return model.IntegerFieldValue(p.intToInt(v)), nil
}

// mapValue reads a range bound: a special character's name, or a number or field name (without int mapping).
func (p *FieldParser) mapValue(s string) (model.FieldValue, error) {
	if s == "~" {
		return model.SpecialCharFieldValue(model.Tilde), nil
	}
	for _, sc := range model.SpecialChars {
		if sc.String() == s {
			return model.SpecialCharFieldValue(sc), nil
		}
	}
	v, err := p.stringToInt(s)
	if err != nil {
		return nil, err
	}
	return model.IntegerFieldValue(v), nil
}

func (p *FieldParser) stringToInt(exp string) (int, error) {
	if v, ok := p.constraints.StringMappingValue(exp); ok {
		return v, nil
	}
	v, err := javalang.ParseInt(exp)
	if err != nil {
		invalid := model.NewStringValidations(p.constraints).RemoveValidChars(exp)
		return 0, cronerr.Argument("Invalid chars in expression! Expression: %s Invalid chars: %s", exp, invalid)
	}
	return v, nil
}

func (p *FieldParser) intToInt(exp int) int {
	if v, ok := p.constraints.IntMappingValue(exp); ok {
		return v
	}
	return exp
}

// CronParserField parses the expression of one named field of a cron.
type CronParserField struct {
	Field       model.CronFieldName
	Constraints *model.FieldConstraints
	Optional    bool
	parser      *FieldParser
}

// NewCronParserField returns the parser of a field.
func NewCronParserField(field model.CronFieldName, constraints *model.FieldConstraints, optional bool) *CronParserField {
	return &CronParserField{Field: field, Constraints: constraints, Optional: optional, parser: NewFieldParser(constraints)}
}

// Parse parses the field's expression; in the day-of-week field a day name followed by L (FRIL) stands for that
// day's number followed by L.
func (f *CronParserField) Parse(expression string) (*model.CronField, error) {
	if f.Field == model.DayOfWeek && strings.HasSuffix(expression, "L") {
		if v, ok := f.Constraints.StringMappingValue(expression[:len(expression)-1]); ok {
			expression = strconv.Itoa(v) + "L"
		}
	}
	e, err := f.parser.Parse(expression)
	if err != nil {
		return nil, err
	}
	return model.NewCronField(f.Field, e, f.Constraints), nil
}
