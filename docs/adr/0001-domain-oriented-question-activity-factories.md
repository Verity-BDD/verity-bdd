# ADR 0001: Domain-oriented question and activity factories

- Status: Accepted
- Date: 2026-08-08

## Context

Before version 1, the HTTP API exposed constructor-oriented names for domain questions and its fluent request builder: `NewResponseHeader`, `NewJSONPath`, and `NewRequestBuilder`. These names describe object construction rather than the Screenplay-domain intent of each call.

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
