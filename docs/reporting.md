# Reporting

Verity-BDD reports test lifecycle and activity events through `verity_reporting.Reporter`.

## Default console reporter

`verity.NewVerityTest(t, scene)` uses a console reporter unless `scene.Reporter` is set. A typical test prints one test-start line, one line for each **completed** activity, and one test-finish line:

```text
🚀 Starting: TestAPITesting
  ✅ APITester sends GET request to /posts (0.21s)
  ✅ APITester ensures that the last response status code equals 200 (0.00s)
✅ TestAPITesting: PASSED (0.26s)
```

`ConsoleReporter.OnStepStart` tracks nesting but emits no separate start line. In particular, current output does not include `🔄` lines.

To select the reporter explicitly or redirect output:

```go
func TestCustomReporting(t *testing.T) {
    reporter := console_reporter.NewConsoleReporter()
    reporter.SetOutput(os.Stdout)

    test := verity.NewVerityTestWithReporter(
        context.Background(),
        t,
        reporter,
    )

    actor := test.ActorCalled("API tester").WhoCan(
        api.CallAnApiAt("https://api.example.com"),
    )
    actor.AttemptsTo(api.SendGetRequest("/health"))
}
```

`NewVerityTestWithReporter` requires `(context.Context, verity.TestContext, verity_reporting.Reporter)`. Alternatively, configure both values in a scene:

```go
test := verity.NewVerityTest(t, verity.Scene{
    Context:  ctx,
    Reporter: reporter,
})
```

Each test should create its own reporter instance. `ConsoleReporter` synchronizes its state, but sharing one reporter across simultaneously running tests would also share its current-test and indentation state.

## Reporter contract

A custom reporter implements:

```go
type Reporter interface {
    OnTestStart(testName string)
    OnTestFinish(result verity_reporting.TestResult)
    OnStepStart(stepDescription string)
    OnStepFinish(stepResult verity_reporting.TestResult)
    OnLog(entry verity_reporting.LogEntry)
    SetOutput(io.Writer)
}
```

Reporter callback results expose `Name`, `Status`, duration in seconds, an error, and attachments. These `verity_reporting.TestResult` and `verity_reporting.Status` types are distinct from the root `verity.TestResult` and `verity.Status` core-state types.

`verity_reporting.NewTestRunnerAdapter` wraps a reporter. `NewActivityTracker` and `NewActivityTrackerWithActor` can be used by integrations that need to emit a start/finish pair manually. When an actor name is supplied, a leading `#actor ` placeholder in the activity description is replaced with that name.

## Logging

Actors log report data through the `verity_interactions/log` package:

```go
import log "github.com/verity-bdd/verity-bdd/verity_interactions/log"

actor.AttemptsTo(log.The(
    "Current page",
    log.AnswerTo(currentPageTitle),
    log.AnswerTo(currentPageURL),
))
```

Static values are formatted independently with `fmt.Sprint`; `nil` becomes `<nil>`. Wrap every dynamic `verity.Question[T]` with `log.AnswerTo`; mixed static values and heterogeneous Question types preserve input order. Questions run left to right with the actor and context supplied to the interaction. If one fails, the Log step fails normally with the Question description in its error and no `OnLog` entry is emitted.

`log.The()` is valid and emits an empty entry. It has the stable activity description `#actor logs`, so normal step-start and step-finish callbacks still bracket logging. The console reporter prints entries at the current nesting level, for example `ℹ️ Sam: Current page Checkout`. The Allure reporter stores the space-joined values as a `log` `text/plain` attachment on that Log step, including inside Tasks.

### Migration for custom reporters

`Reporter.OnLog(entry verity_reporting.LogEntry)` is mandatory. Add the method to every custom reporter implementation; it receives the actor name and ordered, already-formatted values. This is a deliberate breaking change before v1 so log data cannot be silently discarded.

## Attachments: current limitation

The reporting contract and Allure reporter support attachments supplied in a `TestResult`. Apart from the `Log` interaction's direct reporter event described above, the normal actor execution path calls the activity tracker without attachments. There is no public activity API for collecting or forwarding other step-level attachments, so `OnStepFinish` receives no produced attachments during normal Verity execution.

During test shutdown, Verity serializes actor notes into a single test-level attachment named `notes` with content type `application/json` when notes exist. This attachment is passed to `OnTestFinish`.

Do not describe activities or custom attachment sources as operational until the execution pipeline exposes and forwards them.

## Allure reporter

```go
func TestWithAllure(t *testing.T) {
    reporter := allure_reporter.NewAllureReporterWithDir("allure-results")
    test := verity.NewVerityTestWithReporter(context.Background(), t, reporter)

    actor := test.ActorCalled("Tester")
    actor.AttemptsTo(
        verity.Do("performs an action", func(context.Context, verity.Actor) error {
            return nil
        }),
    )
}
```

The reporter writes `*-result.json` files, nested step data, and any attachments actually provided by callbacks. With the current built-in execution path, this means test-level notes can be persisted, while normal step-level attachments are empty.

View results with an installed Allure CLI:

```bash
allure serve allure-results
```

`AllureReporter.SetOutput` is intentionally a no-op because output is written to the configured results directory.
