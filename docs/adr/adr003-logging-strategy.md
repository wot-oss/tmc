# Logging Strategy

## Status

accepted

## Context

We have not defined a logging strategy and, as a consequence, our code only contains sporadic log statements. The
produced logs give little help in way of tracking app's behaviour or finding bugs.

When used to serve the REST API, the application needs to produce some king of access/request logs to enable app
usage analysis and find possible performance bottlenecks.

## Decision

We have two use-cases: CLI and API server, which differ in the requirements to logging.

CLI doesn't need much in terms of logging, only to make sure that any errors are propagated to the user. The logs in
this case are only useful if they contain some additional data helpful for understanding the reasons for errors, but
which was deemed too noisy for including in stdout printouts.

In the API server case, it is necessary to distinguish between log statements from different requests. We will generate
a request id for each request, extend the default logger with fields containing information about request, and pass that
logger in the request's context down the stack.

### Logging Guidelines

What follows, should be seen as guidelines, rather than hard and fast rules. Feel free to deviate from them if the
situation warrants.

- Err on the side of not logging, unless you can explain the value-add of a logging statement. It's better to extend
  logging later when a specific need is determined than pollute logs with noise.
- Indicate which component is logging. Use any string as a component identifier that helps to pinpoint the location
  where the log is produced. This is a cheap poor man's replacement for collecting stacktrace information which is good
  enough most of the time.
- Log at INFO every http request's start and end
- Log errors at those points where information may get lost, e.g. if the \[original] error is not propagated up the
  stack, or if the message printed to stdout does not contain all error details. All errors returned via http will be
  logged automatically.
- Log expected normal errors at DEBUG, e.g. user attempted to import a TM conflicting with existing one
- Log unexpected errors at WARN (e.g. remote tmc repo returned 5**) or ERROR (e.g. index file could not be locked)
- When logging errors, log any parameters that may have caused the error
- Log auth events with authorization=true flag

## Consequences

The logs should get more informative and helpful in tracing errors, while remaining relatively concise.
There will be a number of functions that include `context.Context` in their signature just for access to the logger, as
opposed to being able to get canceled by the context.
