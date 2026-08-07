# ADR 0001: Expose actor registry snapshots

## Status

Accepted

## Convention

Architecture decisions live in `docs/adr` and use `NNNN-kebab-case-title.md`, with numbers assigned sequentially starting at `0001`.

## Context

Consumers need to inspect the actors registered by a `VerityTest` without gaining access to, or depending on, its mutable internal registry. The registry is shared with concurrent `ActorCalled` and `Shutdown` operations.

## Decision

Add `Actors() []Actor` to `VerityTest`.

Each call acquires the registry read lock and checks the test lifecycle state under that lock. After `Shutdown`, it returns a non-nil empty slice regardless of any later `ActorCalled` calls. Otherwise, it copies all registered actor references into a new non-nil slice, sorts that slice by `Actor.Name()` in ascending case-sensitive lexical order, and returns it. The map's existing name key preserves one actor per repeated `ActorCalled` name. `Shutdown` also clears the registry.

## Rejected alternatives

- Return the registry map: exposes mutable internal state and map iteration order.
- Return a cached slice: allows caller mutations to affect later results and complicates synchronization.
- Return actor copies: breaks actor identity and would require copying actor-owned synchronized state.
- Preserve registration order: requires additional state solely for inspection and conflicts with deterministic name ordering.

## Consequences

Callers receive deterministic, isolated slice snapshots containing the original actor instances. Snapshot creation costs one allocation plus sorting, and holds a read lock while copying and sorting; actor registration and shutdown wait briefly for that lock.
