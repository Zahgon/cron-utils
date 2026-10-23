// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command cron-utils parses, validates, describes, maps and schedules cron expressions of the predefined cron
// definitions.
//
//	cron-utils parse          --type T EXPR                       canonical form of the expression
//	cron-utils validate       --type T EXPR                       "valid", or "invalid: " and the reason
//	cron-utils describe       --type T [--locale TAG] EXPR        human readable description
//	cron-utils next           --type T --zone Z --at TIME [--count N] EXPR   next executions, one per line
//	cron-utils last           --type T --zone Z --at TIME [--count N] EXPR   previous executions, one per line
//	cron-utils time-to-next   --type T --zone Z --at TIME EXPR    duration until the next execution
//	cron-utils time-from-last --type T --zone Z --at TIME EXPR    duration since the last execution
//	cron-utils match          --type T --zone Z --at TIME EXPR    "true" or "false"
//	cron-utils executions     --type T --zone Z --at TIME --until TIME EXPR   executions in the window and their count
//	cron-utils map            --from T --to T EXPR                expression converted to another definition
//	cron-utils equivalent     --type T [--from F] --other EXPR2 EXPR   whether EXPR equals EXPR2 (mapped from F)
//	cron-utils compare        --type T --zone Z --at TIME --until TIME --other EXPR2 EXPR   difference of executions
//	cron-utils workdays       --zone Z --at TIME --until TIME [--holidays TIME,...] POLICY   working days
//
// Types are cron4j, quartz, spring, spring53 and unix. Exit status 0 on success, 1 when the library rejects the
// input or fails (with "error: " and the reason on standard error), 2 for a malformed command line ("usage: ").
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmrozanec/cron-utils/cronerr"
	"github.com/jmrozanec/cron-utils/descriptor"
	"github.com/jmrozanec/cron-utils/executiontime"
	"github.com/jmrozanec/cron-utils/javatime"
	"github.com/jmrozanec/cron-utils/mapper"
	"github.com/jmrozanec/cron-utils/model"
	"github.com/jmrozanec/cron-utils/parser"
	"github.com/jmrozanec/cron-utils/utils"
)

const maxCount = 50

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// usageError is a malformed command line.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error { return &usageError{fmt.Sprintf(format, args...)} }

type options struct {
	command, typ, from, to, locale, zone, at, until, count, other, holidays, expression *string
}

func run(args []string, stdout, stderr io.Writer) int {
	o, err := parseOptions(args)
	if err != nil {
		fmt.Fprintln(stderr, "usage: "+err.Error())
		return 2
	}
	var lines []string
	err = execute(o, &lines)
	var usage *usageError
	if errors.As(err, &usage) {
		fmt.Fprintln(stderr, "usage: "+usage.msg)
		return 2
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error: "+describeFailure(err))
		return 1
	}
	return 0
}

// describeFailure returns the message of a rejected input, or the kind of an internal failure.
func describeFailure(err error) string {
	var e *cronerr.Error
	if errors.As(err, &e) && e.Kind.Defect() {
		return e.Kind.String()
	}
	return err.Error()
}

var commands = map[string]bool{"parse": true, "validate": true, "describe": true, "next": true, "last": true,
	"time-to-next": true, "time-from-last": true, "match": true, "executions": true, "map": true, "equivalent": true,
	"compare": true, "workdays": true}

func parseOptions(args []string) (*options, error) {
	if len(args) == 0 {
		return nil, usagef("missing command")
	}
	o := &options{command: &args[0]}
	if !commands[args[0]] {
		return nil, usagef("unknown command '%s'", args[0])
	}
	targets := map[string]**string{"--type": &o.typ, "--from": &o.from, "--to": &o.to, "--locale": &o.locale,
		"--zone": &o.zone, "--at": &o.at, "--until": &o.until, "--count": &o.count, "--other": &o.other,
		"--holidays": &o.holidays}
	i := 1
	for i < len(args) && strings.HasPrefix(args[i], "--") {
		name := args[i]
		if name == "--" {
			i++
			break
		}
		if i+1 >= len(args) {
			return nil, usagef("option %s needs a value", name)
		}
		target, ok := targets[name]
		if !ok {
			return nil, usagef("unknown option %s", name)
		}
		value := args[i+1]
		*target = &value
		i += 2
	}
	if i != len(args)-1 {
		return nil, usagef("expected exactly one expression")
	}
	o.expression = &args[i]
	return o, nil
}

var cronTypes = map[string]model.CronType{"cron4j": model.Cron4j, "quartz": model.Quartz, "spring": model.Spring,
	"spring53": model.Spring53, "unix": model.Unix}

func cronType(option string, value *string) (model.CronType, error) {
	if value == nil {
		return 0, usagef("missing %s", option)
	}
	t, ok := cronTypes[*value]
	if !ok {
		return 0, usagef("unknown cron type '%s'", *value)
	}
	return t, nil
}

func zone(value *string) (javatime.Zone, error) {
	if value == nil {
		return javatime.Zone{}, usagef("missing --zone")
	}
	z, err := javatime.ZoneOf(*value)
	if err != nil {
		return javatime.Zone{}, usagef("invalid zone '%s'", *value)
	}
	return z, nil
}

func parseTime(option string, value *string, z javatime.Zone) (javatime.ZonedDateTime, error) {
	if value == nil {
		return javatime.ZonedDateTime{}, usagef("missing %s", option)
	}
	t, err := javatime.ParseZoned(*value, z)
	if err != nil {
		return javatime.ZonedDateTime{}, usagef("invalid time '%s'", *value)
	}
	return t, nil
}

var countPattern = regexp.MustCompile(`^[0-9]{1,2}$`)

func count(value *string) (int, error) {
	if value == nil {
		return 1, nil
	}
	if !countPattern.MatchString(*value) {
		return 0, usagef("invalid count '%s'", *value)
	}
	n, err := strconv.Atoi(*value)
	if err != nil || n < 1 || n > maxCount {
		return 0, usagef("invalid count '%s'", *value)
	}
	return n, nil
}

func parse(t model.CronType, expression string) (model.Cron, error) {
	return parser.New(model.InstanceDefinitionFor(t)).Parse(expression)
}

func execute(o *options, lines *[]string) error {
	add := func(s string) { *lines = append(*lines, s) }
	switch *o.command {
	case "map":
		from, err := cronType("--from", o.from)
		if err != nil {
			return err
		}
		to, err := cronType("--to", o.to)
		if err != nil {
			return err
		}
		m, err := mapperFor(from, to)
		if err != nil {
			return err
		}
		cron, err := parse(from, *o.expression)
		if err != nil {
			return err
		}
		mapped, err := m.Map(cron)
		if err != nil {
			return err
		}
		add(mapped.AsString())
		return nil
	case "workdays":
		return workdays(o, add)
	}
	typ, err := cronType("--type", o.typ)
	if err != nil {
		return err
	}
	switch *o.command {
	case "equivalent":
		return equivalent(o, typ, add)
	case "parse":
		cron, err := parse(typ, *o.expression)
		if err != nil {
			return err
		}
		s, err := model.CanonicalString(cron)
		if err != nil {
			return err
		}
		add(s)
		return nil
	case "validate":
		cron, err := parse(typ, *o.expression)
		if err == nil {
			err = cron.Validate()
		}
		switch {
		case err == nil:
			add("valid")
		case cronerr.Is(err, cronerr.IllegalArgument):
			add("invalid: " + err.Error())
		default:
			return err
		}
		return nil
	case "describe":
		tag := descriptor.DefaultLocale
		if o.locale != nil {
			tag = *o.locale
		}
		d, err := descriptor.InstanceFor(tag)
		if err != nil {
			return err
		}
		cron, err := parse(typ, *o.expression)
		if err != nil {
			return err
		}
		text, err := d.Describe(cron)
		if err != nil {
			return err
		}
		add(text)
		return nil
	}
	return schedule(o, typ, add)
}

func equivalent(o *options, typ model.CronType, add func(string)) error {
	if o.other == nil {
		return usagef("missing --other")
	}
	cron, err := parse(typ, *o.expression)
	if err != nil {
		return err
	}
	if o.from == nil {
		other, err := parse(typ, *o.other)
		if err != nil {
			return err
		}
		a, err := model.CanonicalString(cron)
		if err != nil {
			return err
		}
		b, err := model.CanonicalString(other)
		if err != nil {
			return err
		}
		add(strconv.FormatBool(a == b))
		return nil
	}
	from, err := cronType("--from", o.from)
	if err != nil {
		return err
	}
	m, err := mapperFor(from, typ)
	if err != nil {
		return err
	}
	other, err := parse(from, *o.other)
	if err != nil {
		return err
	}
	if _, composite := cron.(*model.CompositeCron); composite {
		return cronerr.New(cronerr.UnsupportedOperation, "Currently not supported for CompositeCron")
	}
	same, err := m.Equivalent(cron, other)
	if err != nil {
		return err
	}
	add(strconv.FormatBool(same))
	return nil
}

func schedule(o *options, typ model.CronType, add func(string)) error {
	z, err := zone(o.zone)
	if err != nil {
		return err
	}
	at, err := parseTime("--at", o.at, z)
	if err != nil {
		return err
	}
	n, err := count(o.count)
	if err != nil {
		return err
	}
	var until javatime.ZonedDateTime
	if *o.command == "executions" || *o.command == "compare" {
		if until, err = parseTime("--until", o.until, z); err != nil {
			return err
		}
	}
	if *o.command == "compare" {
		if o.other == nil {
			return usagef("missing --other")
		}
		p := parser.New(model.InstanceDefinitionFor(typ))
		cron, err := p.Parse(*o.expression)
		if err != nil {
			return err
		}
		other, err := p.Parse(*o.other)
		if err != nil {
			return err
		}
		diff, err := utils.CronFrequencyComparator{Start: at, End: until}.Compare(cron, other)
		if err != nil {
			return err
		}
		add(strconv.Itoa(diff))
		return nil
	}
	cron, err := parse(typ, *o.expression)
	if err != nil {
		return err
	}
	et, err := executiontime.ForCron(cron)
	if err != nil {
		return err
	}
	switch *o.command {
	case "next", "last":
		step := et.NextExecution
		if *o.command == "last" {
			step = et.LastExecution
		}
		current := at
		for i := 0; i < n; i++ {
			next, ok, err := step(current)
			if err != nil {
				return err
			}
			if !ok {
				add("none")
				return nil
			}
			current = next
			add(current.String())
		}
	case "time-to-next", "time-from-last":
		step := et.TimeToNextExecution
		if *o.command == "time-from-last" {
			step = et.TimeFromLastExecution
		}
		d, ok, err := step(at)
		if err != nil {
			return err
		}
		if !ok {
			add("none")
		} else {
			add(d.String())
		}
	case "match":
		ok, err := et.IsMatch(at)
		if err != nil {
			return err
		}
		add(strconv.FormatBool(ok))
	case "executions":
		dates, err := executiontime.ExecutionDates(et, at, until)
		if err != nil {
			return err
		}
		if len(dates) > 1000 {
			return usagef("window holds more than 1000 executions")
		}
		for _, d := range dates {
			add(d.String())
		}
		add(fmt.Sprintf("count %d", len(dates)))
	}
	return nil
}

var policies = map[string]utils.WeekendPolicy{"thursday-friday": utils.ThursdayFriday,
	"friday-saturday": utils.FridaySaturday, "saturday-sunday": utils.SaturdaySunday}

func workdays(o *options, add func(string)) error {
	z, err := zone(o.zone)
	if err != nil {
		return err
	}
	at, err := parseTime("--at", o.at, z)
	if err != nil {
		return err
	}
	until, err := parseTime("--until", o.until, z)
	if err != nil {
		return err
	}
	policy, ok := policies[*o.expression]
	if !ok {
		return usagef("unknown weekend policy '%s'", *o.expression)
	}
	var holidays []javatime.ZonedDateTime
	if o.holidays != nil && *o.holidays != "" {
		for _, h := range strings.Split(*o.holidays, ",") {
			value := h
			t, err := parseTime("--holidays", &value, z)
			if err != nil {
				return err
			}
			holidays = append(holidays, t)
		}
	}
	days, err := utils.WorkdaysCount(at, until, holidays, policy)
	if err != nil {
		return err
	}
	add(strconv.Itoa(days))
	return nil
}

func mapperFor(from, to model.CronType) (*mapper.CronMapper, error) {
	type pair struct{ from, to model.CronType }
	if from == to {
		return mapper.SameCronMapper(model.InstanceDefinitionFor(from)), nil
	}
	known := map[pair]func() *mapper.CronMapper{
		{model.Cron4j, model.Quartz}: mapper.FromCron4jToQuartz,
		{model.Quartz, model.Cron4j}: mapper.FromQuartzToCron4j,
		{model.Quartz, model.Unix}:   mapper.FromQuartzToUnix,
		{model.Unix, model.Quartz}:   mapper.FromUnixToQuartz,
		{model.Quartz, model.Spring}: mapper.FromQuartzToSpring,
		{model.Spring, model.Quartz}: mapper.FromSpringToQuartz,
	}
	if f, ok := known[pair{from, to}]; ok {
		return f(), nil
	}
	return nil, usagef("no mapping from %s to %s", strings.ToLower(from.String()), strings.ToLower(to.String()))
}
