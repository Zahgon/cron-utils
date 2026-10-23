# cron-utils
_We define crons. And support them._


cron-utils is a Go library to define, parse, validate, migrate crons as well as get human readable descriptions for them. The project follows the [Semantic Versioning Convention](https://semver.org/), uses Apache 2.0 license.

[![Gitter Chat](https://img.shields.io/badge/chat-online-brightgreen.svg)](https://gitter.im/jmrozanec/cron-utils)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=cron-utils&metric=coverage)](https://sonarcloud.io/dashboard/index/cron-utils)


**Download**

cron-utils is a Go module with no dependencies outside the standard library (Go 1.25 or later):

    go get github.com/jmrozanec/cron-utils

Build and test from a checkout with `go build ./...` and `go test ./...`.

**Features**

 * Create arbitrary cron expressions: you can define your own cron format! Supported fields are: second, minute, hour, day of month, month, day of week, year.
 * You can flag last field as optional!
 * Supports all cron special characters: * / , -
    * Non-standard characters L, W, LW, '?' and # are supported as well!
 * Supports cron extensions / nicknames: @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly, @reboot
 * Print to locale specific human readable format (Chinese, English, German, Greek, Indonesian, Japanese, Korean, Polish, Romanian, Spanish, Swahili, and Turkish are fully supported. Dutch, French, Italian, Portuguese and Russian have basic support).
 * Parse and Description process are decoupled: parse once and operate with the result!
 * Build cron expressions using the builder package: 
    * no need to remember fields and constraints for each cron provider.
    * crons become decoupled from cron provider: anytime you can export to another format.
 * Check if cron expressions are equivalent.
 * Squash multiple cron expressions into a single one!
 * Validate if cron string expressions match a cron definition.
 * A `cron-utils` command line parses, validates, describes, maps and schedules expressions.
 * Convert crons between different cron definitions: if you need to migrate expressions, the mapper package may help you!
 * Pre-defined definitions for the following cron libraries are provided:
    * [Unix](https://www.unix.com/man-page/linux/5/crontab/)
    * [Cron4j](https://www.sauronsoftware.it/projects/cron4j/)
    * [Quartz](https://quartz-scheduler.org/)
    * [Spring](https://docs.spring.io/spring-framework/docs/current/javadoc-api/org/springframework/scheduling/support/CronExpression.html). We support two definitions: prior to v5.3 and [from v5.3 onwards](https://spring.io/blog/2020/11/10/new-in-spring-5-3-improved-cron-expressions).
 * Obtain last/next execution time as well as time from last execution/time to next execution.
 * Compute the number of executions between two days, and compare crons based on their execution frequencies.
 * Obtain weekdays count between two dates, considering different weekend policies as well as holidays.
 * Need to map constants between different cron/time libraries? Use mapper.WeekDayMapping.

**Usage Examples**

Below we present some examples. You can find this and others in a [sample repo we created](https://github.com/jmrozanec/cron-utils-examples) to showcase cron-utils libraries!

***Build cron definitions***

```go
// Define your own cron: arbitrary fields are allowed and last field can be optional
cronDefinition, err := model.DefineCron().
	WithSeconds().And().
	WithMinutes().And().
	WithHours().And().
	WithDayOfMonth().SupportsHash().SupportsL().SupportsW().And().
	WithMonth().And().
	WithDayOfWeek().
	WithIntMapping(7, 0). // we support non-standard non-zero-based numbers!
	SupportsHash().SupportsL().SupportsW().And().
	WithYear().Optional().And().
	Instance()

// or get a predefined instance
cronDefinition = model.InstanceDefinitionFor(model.Quartz)
```

***Build a cron expression***
```go
// Create a cron expression. The builder ensures you remain cron provider agnostic
cron, err := builder.Cron(model.InstanceDefinitionFor(model.Quartz)).
	WithYear(model.AlwaysExpression()).
	WithDoM(model.NewBetween(model.SpecialCharFieldValue(model.L), model.IntegerFieldValue(3))).
	WithMonth(model.AlwaysExpression()).
	WithDoW(model.QuestionMarkExpression()).
	WithHour(model.AlwaysExpression()).
	WithMinute(model.AlwaysExpression()).
	WithSecond(model.NewOn(0)).
	Instance()
// Obtain the string expression
cronAsString := cron.AsString() // 0 * * L-3 * ? *
```

***Parse***
```go
// Create a parser based on provided definition
p := parser.New(cronDefinition)
quartzCron, err := p.Parse("0 23 * ? * 1-5 *")
```

... even multi-cron expressions! How about squashing multiple crons into a single line? Instead of writing ```0 0 9 * * ? *```, ```0 0 10 * * ? *```, ```0 30 11 * * ? *``` and ```0 0 12 * * ? *``` we can wrap it into ```0 0|0|30|0 9|10|11|12 * * ? *```


***Describe***
```go
// Create a descriptor for a specific locale (a BCP 47 language tag)
d, err := descriptor.InstanceFor("en-GB")

// Parse some expression and ask descriptor for description
cron, err := p.Parse("*/45 * * * * ?")
description, err := d.Describe(cron)
// Description will be: "every 45 seconds"

description, err = d.Describe(quartzCron)
// Description will be: "every hour at minute 23 every day between Monday and Friday"
// which is the same description we get for "0 23 * ? * MON-FRI *"
```

***Migrate***
```go
// Migration between cron libraries has never been so easy!
// Turn cron expressions into another format by using a mapper:
cron4jCron, err := mapper.FromQuartzToCron4j().Map(quartzCron)
// and to get a string representation of it, we can use
cron4jCron.AsString() // will return: 23 * * * 1-5
```

***Validate***
```go
err := cron4jCron.Validate()
```

***Calculate time from/to execution***
```go
// Execution times are zoned date-times: javatime.FromTime places a time.Time in a zone
zone, err := javatime.ZoneOf("Europe/Paris")
now := javatime.FromTime(time.Now(), zone)
et, err := executiontime.ForCron(cron)

// Get date for last execution
lastExecution, ok, err := et.LastExecution(now)

// Get date for next execution
nextExecution, ok, err := et.NextExecution(now)

// Time from last execution
timeFromLastExecution, ok, err := et.TimeFromLastExecution(now)

// Time to next execution
timeToNextExecution, ok, err := et.TimeToNextExecution(now)
```

***Map constants between libraries***
```go
// Map day of week value from Quartz to the ISO numbering (Monday is 1)
isoDayOfWeek := mapper.WeekDayMapping(mapper.QuartzWeekDay, mapper.Java8WeekDay, 2)
```

***cron-utils CLI***

The `cron-utils` command gives the library's features on the command line:

    go build -o cron-utils ./cmd/cron-utils
    cron-utils parse --type quartz -- '0 23 * ? * 1-5 *'
    cron-utils describe --type unix --locale de -- '0 12 * * MON-FRI'
    cron-utils next --type quartz --zone Europe/Paris --at 2024-03-30T12:00 --count 3 -- '0 30 2 * * ?'
    cron-utils map --from quartz --to unix -- '0 23 * ? * 1-5 *'

See the command's documentation (`go doc ./cmd/cron-utils`) for every command and option.

**Contribute & Support!**

Contributions are welcome! You can contribute by
 * starring this repo!
 * requesting or adding new features. Check our [roadmap](https://github.com/jmrozanec/cron-utils/wiki/Roadmap)!
 * enhancing existing code: ex.: provide more accurate description cases
 * testing
 * enhancing documentation
 * providing translations to support new locales
 * bringing suggestions and reporting bugs
 * spreading the word 
 * telling us how you use it! We look forward to [list you at our wiki](https://github.com/jmrozanec/cron-utils/wiki/Projects-using-cron-utils)!


Check [our page](https://cronutils.com)! For stats about the project, you can visit our [OpenHUB profile](https://www.openhub.net/p/cron-utils).

**Other cron-utils projects**

You are welcome to visit and use the following cron-utils projects:
 * [htime](https://github.com/jmrozanec/htime): A Java library to make it easy for humans format a date. You no longer need to remember date time formatting chars: just write an example, and you will get the appropriate formatter.
 * [cron-utils-spring](https://github.com/jmrozanec/cron-utils-spring): A Java library to describe cron expressions in human readable language at Spring framework, using cron-utils.
 * [cron-utils-sisyphus](https://github.com/jmrozanec/cron-utils-sisyphus): A Scala scheduler that supports multiple cron notations.
 * [cron-utils-scheduler](https://github.com/jmrozanec/cron-utils-scheduler): A Java job scheduler based on cron-utils library.
