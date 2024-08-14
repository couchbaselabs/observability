# Writing Hazelnut Rules

Hazelnut is the Health Agent's log analysis engine. It receives logs (currently only from a running cluster via Fluent
Bit) and applies rules to them. This file explains how to write these rules

## Structure

Rules are defined in JSON format in `rules.json`. See below for comments about the meaning of each field (note that
comments aren't normally permitted in JSON, this is for explanation only):

```json5
{
  // A name for this rule - must be unique
  "ruleName": "memcachedCrash",
  // The file to apply this rule to - note that it must be named the same as it would be on the server (so no `ns_server.` prefix),
  // but Fluent Bit will strip off the number of the files (so `memcached.log` rather than `memcached.log.000000.txt`)
  "file": "memcached.log",
  // A string to match *exactly* against the log line. If possible, include one, as it saves having to apply a regex to each line.
  "contains": "Breakpad",
  // A regular expression to apply to the line. Note that if you provide both `contains` and `regexp`, *both* must match.
  "regexp": ".*Breakpad caught(?: a)? crash in (.*).",
  // A set of hints to Hazelnut about where to look (explained below)
  "hints": {
    "level": "warning"
  },
  // A set of rules you can implement for fields other than message to validate their output
  "customFields": [
    {
      // Name of the field, can be anything
      "name": "totalDuration",
      // Type can be of two types: time or string. If you use time, make sure to use timeThreshold as well.
      "type": "time",
      // A threshold which can be applied, only supported as go Duration Parsable format (eg: 1m20s). Triggered only when timing
      // in log exceeds this threshold.
      "timeThreshold": "1m"
    },
    {
      "name": "file_path",
      // A string type requires a regexp to be followed.
      "type": "string",
      // Only basic regexp matching is allowed here. Cannot extract fields from here.
      "regexp": "^/opt/couchbase.*"
    }
  ],
  // The checker result to output if a line matches this rule
  "result": {
    "name": "memcachedCrash",
    // status can only take up values: good, warn and alert
    "status": "alert",
    "remediation": "The Data Service crashed. Examine its logs or contact Couchbase Technical Support.",
    "id": "CB90036"
  },
  // Tests for this rule (explained below)
  "testCases": [
    // ...
  ]
}
```

## Evaluation

The process that Hazelnut follows for each log line is more or less the following:

1. Select the rules for the file in question and apply them.
2. Apply the `hints` against the fields that Fluent Bit gives us, and discard the line if the hints don't match exactly.
3. Check if the line contains the `contains` string from the rule, and discard it if it doesn't.
4. Check if the line matches the `regexp`, and discard it if it doesn't.
5. Check if the rules has any custom fields to check on. Apply custom field rules, otherwise skip this step.
6. Substitute the `result.remediation` (using Go's [Expand](https://pkg.go.dev/regexp#Regexp.Expand) syntax), and store
   the resulting checker result.

This has a few noteworthy implications:

* If `contains` and `regexp` are provided, `contains` is applied first, then `regexp`, and both must match. A string
  comparison is nearly always faster than a regular expression, so make `contains` as specific as possible.
* In custom fields, if the type is set to `string`, a `regexp` field should also be present and for type `time`,
  a `timeThreshold` field should be present.
* Hints are just that - hints - and Hazelnut is free to ignore them. Your rules should still work correctly even without
  using hints (this is checked in unit tests).
  * To see the possible fields to use in `hints`, examine the [couchbase-fluent-bit config](https://github.com/couchbase/couchbase-fluent-bit/tree/main/conf) or configure fluent-bit to output to stdout.

## Testing

All rules should have unit tests, to make sure the rules match appropriately (and, just as importantly, don't match when
they shouldn't).
The tests are written inline next to the rule itself.

A rule should have at least two test cases, one where it matches and one where it doesn't.
Complex rules may merit more than two.

Each test case looks like this:

```json5
{
  // A name for the test case
  // (must be unique for the rule, but tests for different rules can have the same name)
  "name": "match",
  // The input to apply the rule on - should match what Fluent Bit would generate
  "input": {
    "level": "warning",
    "file": "memcached.log",
    "message": "Breakpad caught crash in memcached. Writing crash dump to /opt/couchbase/var/lib/couchbase/crash/1e4a92b9-e350-b8d2-67b966ba-1b89a355.dmp before terminating."
  },
  // The expected checker result for this line. If no result is expected, set this to `null`.
  "expected": {
    "name": "memcachedCrash",
    "status": "alert",
    // remediation can be omitted if it's variable
    "remediation": "The Data Service crashed. Examine its logs or contact Couchbase Technical Support.",
    "requireRemediation": true
  }
}
```

To run the tests, simply run `go test ./agent/pkg/hazelnut`.
