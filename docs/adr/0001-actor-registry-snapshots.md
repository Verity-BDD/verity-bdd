# ADR 0001: Expose actor registry snapshots

## Status

Accepted

## Convention

Architecture decisions live in `docs/adr` and use `NNNN-kebab-case-title.md`, with numbers assigned sequentially starting at `0001`.

## Context

Consumers need to inspect the actors registered by a `VerityTest` without gaining access to, or depending on, its mutable internal registry. The registry is shared with concurrent `ActorCalled` and `Shutdown` operations.

## Decision

Add `Actors() []Actor` to `VerityTest`.

Each call copies all registered actor references into a new non-nil slice while holding the registry read lock, releases the lock, sorts the independent slice by `Actor.Name()` in ascending case-sensitive lexical order, and returns it. The map's existing name key preserves one actor per repeated `ActorCalled` name.

`Shutdown` is a terminal, idempotent lifecycle transition. Under the same lock used by `ActorCalled`, it finishes reporting, sets the actor registry to `nil`, and marks the test shut down. `ActorCalled` uses one write-lock critical section for lookup and creation; after shutdown it explicitly panics with `verity: ActorCalled called after Shutdown`. Consequently an `ActorCalled` racing `Shutdown` either completes before the terminal transition or acquires the lock afterward and panics. `Actors` needs no shutdown-specific branch: after shutdown, copying the nil registry naturally produces a non-nil empty snapshot.

## Rejected alternatives

- Return the registry map: exposes mutable internal state and map iteration order.
- Return a cached slice: allows caller mutations to affect later results and complicates synchronization.
- Return actor copies: breaks actor identity and would require copying actor-owned synchronized state.
- Preserve registration order: requires additional state solely for inspection and conflicts with deterministic name ordering.
- Mask post-shutdown registrations in `Actors`: lets `ActorCalled` create hidden actors after the terminal transition and makes the snapshot misrepresent the registry. Reject post-shutdown mutation at `ActorCalled` instead.

## Consequences

Callers receive deterministic, isolated slice snapshots containing the original actor instances. Snapshot creation costs one allocation plus sorting, but holds the read lock only while copying; sorting cannot block actor registration or shutdown. Shutdown releases registry-held actor references, and lifecycle misuse fails explicitly rather than creating hidden state.
