# ADR 0004: Reporter-visible Log interaction

- Status: Accepted
- Date: 2026-08-15

## Context

Screenplay logging must be an actor action rather than text embedded in an Activity description. Reporters need the resolved, ordered values as report data, including values obtained by type-safe Questions in the performing actor's context.

## Decision

Verity ships `verity_interactions/log`. `log.The(values ...any)` creates the fail-fast `#actor logs` Interaction. Static values use `fmt.Sprint`; Questions are passed explicitly with `log.AnswerTo(question)`, which captures typed Question evaluation without reflection or dependence on concrete Question types.

The reporting contract has a mandatory `Reporter.OnLog(LogEntry)` callback. `LogEntry` carries the actor name and ordered string values. The Log Interaction resolves all arguments before delivery, so an error wraps its Question description and produces no partial event.

The core `Actor` interface remains unchanged. The interaction uses a narrow structural `LogSink` boundary; Verity's standard test actor implements it by forwarding the entry to its configured reporter and returns an explicit error when the capability is absent.

The console reporter writes the entry at the active nesting level. The Allure reporter persists a `log` `text/plain` attachment on the currently open Log step.

## Rejected alternatives

- **Reflection-based Question detection.** Rejected because heterogeneous generic Questions cannot be identified reliably without coupling to implementation details.
- **An optional reporter logging interface.** Rejected because it allows silent loss of report data.
- **Mutating the Activity description with resolved values.** Rejected because it destroys the stable lifecycle description and does not deliver structured reporter data.

## Consequences

Custom `verity_reporting.Reporter` implementations must add `OnLog(entry verity_reporting.LogEntry)`. This is intentionally a breaking pre-v1 reporting-contract change. Log values are now reporter-visible, while existing `OnStepStart`/`OnStepFinish` lifecycle behavior remains intact.
