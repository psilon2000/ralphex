# MVP Status (ralphex -> opencode)

Last updated: 2026-03-02

## Acceptance status

- [x] `MVP-FLAG-GATE-01`: adapter gate `RALPHEX_OPENCODE_ADAPTER=0|1` is implemented, default path remains existing engine.
- [x] `MVP-CLI-01..05`: adapter mode routing covers full/tasks-only/review/codex-only flows.
- [x] `MVP-EXIT-01..03`: success/error contract remains binary (`0` success, `1` failure).
- [x] `MVP-required` CI gate jobs are defined in `.github/workflows/ci.yml`.
- [x] Compatibility and architecture docs are in place (`cli-compat-matrix.md`, `adr-mvp-adapter.md`).

## Evidence

Code paths:
- `cmd/ralphex/main.go`: adapter dispatch + mode routing.
- `cmd/ralphex/opencode_adapter.go`: task/review execution and signal validation.
- `pkg/config/config.go`: config surface for adapter command/args.
- `pkg/config/values.go`: config parsing/merge logic for adapter fields.
- `docs/porting/adapter-e2e-runbook.md`: manual e2e validation steps for adapter mode.

Tests:
- Unit compatibility: `TestDetermineMode`, `TestValidateFlags`, `TestUseOpencodeAdapter`, `TestSupportsOpencodeAdapterMode`, `TestRunWithOpencodeAdapter`.
- Integration adapter: `TestRunTaskPhaseRunsPendingTasks`, `TestRunTaskPhasePropagatesRunnerError`, `TestRunTaskPhaseRequiresPlan`, `TestValidateTaskOutput`, `TestRunReviewPhase`, `TestRunReviewPhaseDiffError`, `TestBuildReviewPrompt`, `TestGitDiffOutput`, `TestNewOpencodeAdapter`.
- Smoke/e2e adapter: `TestExecOpencodeRunnerSmoke`, `TestOpencodeAdapterTaskPhaseSmoke`, `TestOpencodeAdapterReviewPhaseSmoke`.

## Local verification snapshot

- `go test ./cmd/ralphex` -> pass
- `go test ./pkg/config` -> pass
- `go test ./...` -> pass

## Post-MVP backlog (not blocking)

- Add CI artifact/log collection for adapter smoke failures to speed up debugging.
- Expand traceability from test names to each individual `MVP-FLAG-*` row.
