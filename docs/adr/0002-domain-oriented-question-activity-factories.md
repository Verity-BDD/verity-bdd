# ADR 0002: Domain-oriented question and activity factories

- Status: Accepted
- Date: 2026-08-08

## Context

Before version 1, the HTTP API exposed constructor-oriented names for domain questions and its fluent request builder: `NewResponseHeader`, `NewJSONPath`, and `NewRequestBuilder`. These names describe object construction rather than the Screenplay-domain intent of each call.

Issue #52 requires the exported API to be audited by domain role rather than by the `New*` prefix alone. The inventory below covers public entry points that create Questions, Activities, Interactions, or related fluent builders.

## Public API inventory

Package names below use their normal import aliases: `verity` is the module root, and the other names identify their public subpackages.

### Renamed

| Public entry point | Domain role | Decision |
| --- | --- | --- |
| `api.NewResponseHeader` | Question factory | Rename to `api.LastResponseHeader`. |
| `api.NewJSONPath` | Question factory | Rename to `api.LastResponseBodyAtJSONPath`. |
| `api.NewRequestBuilder` | Fluent HTTP request builder | Rename to `api.RequestFor`. |

### Retained as domain-oriented

| Public entry points | Domain role | Decision |
| --- | --- | --- |
| `verity.Do` | Interaction factory | Retain; the name describes the actor's action. |
| `verity.TaskWhere` | Task/Activity factory | Retain; the name describes task composition. |
| `verity.QuestionAbout` | Question factory | Retain; the name describes the question being asked. |
| `answerable.ValueOf` | Question factory | Retain; the name describes the answer source. |
| `ensure.That` | Assertion Activity factory | Retain; the call site reads as Screenplay vocabulary. |
| `wait.Until`, `wait.UntilReceived` | Fluent wait Activity builders | Retain; the names describe the waiting condition. |
| `take_notes.TakeNoteOf` | Fluent note-taking Activity builder | Retain; the name describes the actor's activity. |
| `take_notes.Note`, `take_notes.NoteValue` | Question factories | Retain; the names describe the requested note. |
| `api.SendRequest`, `api.SendGetRequest`, `api.SendPostRequest`, `api.SendPutRequest`, `api.SendDeleteRequest`, `api.SendPatchRequest` | HTTP Activity factories and fluent request Activities | Retain; the names describe the request being sent. |
| `api.LastResponseBodyAsJSON` | Question factory | Retain; the name describes the requested response representation. |

### Audited non-factory surface and exclusions

- `api.LastResponseStatus`, `api.LastResponseBody`, and `api.ResponseTime` are directly instantiated Question types; `api.LastResponseStatusQ`, `api.LastResponseBodyQ`, and `api.ResponseTimeQ` are pre-built Question values. They are not factory entry points and retain their existing names.
- `api.Using`, `api.CallAnApiAt`, `take_notes.UsingEmptyNotepad`, and `take_notes.Using` create Abilities rather than Questions or Activities and are outside this decision.
- `verity.NewVerityTest`, `verity.NewVerityTestWithContext`, and `verity.NewVerityTestWithReporter` create test-harness objects and are explicitly out of scope.
- `take_notes.NewNoteBook` and `take_notes.NotepadWith` create ordinary ability-state containers and are explicitly out of scope.
- `allure_reporter.NewAllureReporterWithDir`, `console_reporter.NewConsoleReporter`, `reporting.NewTestRunnerAdapter`, `reporting.NewActivityTracker`, and `reporting.NewActivityTrackerWithActor` create reporting infrastructure and are explicitly out of scope.
- Internal-only constructors are not part of the exported API and are outside this decision.

The audit found no additional constructor-oriented public Question, Activity, Interaction, or related fluent-builder factories that require renaming.

## Decision

Expose the same concrete types and behavior through domain-oriented entry points:

- `LastResponseHeader(key string) ResponseHeader`
- `LastResponseBodyAtJSONPath(path string) JSONPath`
- `RequestFor(method, url string) *RequestBuilder`

Remove the old exported names without deprecated aliases. Use the same names internally so tests and implementation share the public domain vocabulary.

## Rejected alternatives

- **Keep deprecated forwarding aliases.** Rejected because the library is pre-v1 and the goal is to freeze a smaller API surface before compatibility is promised.
- **Retain generic `New*` naming.** Rejected because it emphasizes construction rather than questions, activities, interactions, and fluent request intent.
- **Redesign the builder or concrete types.** Rejected as unrelated to the naming decision and unnecessary to preserve behavior.

## Consequences

Callers must migrate to the new names before upgrading. Existing question behavior, request-building behavior, fluent methods, and the exported `ResponseHeader`, `JSONPath`, and `RequestBuilder` types remain unchanged. Compile-time signature contracts cover the new public entry points, while API naming remains an explicit design and review decision rather than a repository-wide restriction on `New*` declarations.
