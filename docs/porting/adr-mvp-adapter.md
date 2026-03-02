# ADR: MVP opencode adapter integration

- Status: accepted (MVP)
- Date: 2026-03-02
- Decision owners: ralphex maintainers

## Context

We need to port the ralphex execution loop to opencode primitives while keeping
CLI behavior compatible for the MVP:

- preserve critical modes and key flags
- preserve exit-code contract (`0` success, `1` failure)
- keep public development in fork `psilon2000/ralphex`
- avoid a hard runtime dependency on a separate local checkout of `anomalyco/opencode`

The existing codebase is Go (`cmd/ralphex`, `pkg/*`), while opencode core is
TypeScript/Bun. A full immediate rewrite is high-risk for MVP.

## Decision

Use an incremental adapter strategy with repository-local integration points.

1. Keep current ralphex CLI entrypoint as the stable facade.
2. Introduce a feature gate `RALPHEX_OPENCODE_ADAPTER=0|1` (default `0`).
3. Behind the gate, route only MVP-required execution paths to opencode-backed
   orchestration.
4. Keep non-MVP features (web/watch mode, interactive plan creation, config
   utility commands) on existing implementation until parity is reached.

For repository strategy, use vendored/imported opencode adapter sources in this
fork (subtree-style sync) so the fork builds and tests autonomously in CI.

## Why this is the simplest viable path

- No immediate breaking CLI changes for existing users.
- Safe rollback by switching one env flag back to `0`.
- Enables incremental parity by mode/loop without blocking on full rewrite.
- Keeps artifacts and CI in one public repository.

## Alternatives considered

### A) Full rewrite first (Go -> TS/Bun in one step)

Rejected for MVP: high migration risk, long feedback cycle, hard to isolate
regressions.

### B) Runtime dependency on external opencode checkout

Rejected for MVP: non-reproducible CI, fragile local setup, hidden coupling.

### C) Keep ralphex only, no opencode adapter

Rejected: does not satisfy requested porting goal.

## Repository layout impact

- `docs/porting/cli-compat-matrix.md` tracks accepted MVP contract.
- `docs/porting/adr-mvp-adapter.md` tracks architecture decision.
- Adapter code will be placed under a dedicated internal module path in this
  repo (to be finalized in implementation PR).

## Rollout plan

1. Baseline CLI contract + tests (matrix IDs `MVP-*`).
2. Adapter skeleton under feature flag (no behavior change when flag is `0`).
3. Port planner loop -> integration tests.
4. Port implementer loop -> integration tests.
5. Port reviewer loop -> integration tests.
6. Enable CI gate on `MVP-required` IDs.

## Rollback plan

- Immediate: set `RALPHEX_OPENCODE_ADAPTER=0`.
- If bad merge reached default branch: revert merge commit in fork.
- Keep compatibility matrix and tests as diagnostics even after rollback.

## Consequences

Positive:
- controlled migration with explicit compatibility boundaries.
- low-risk fallback path.

Trade-offs:
- temporary dual-path maintenance during MVP window.
- some features intentionally deferred until post-MVP.
