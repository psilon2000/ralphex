# CLI Compatibility Matrix (ralphex -> opencode MVP)

## Scope

This matrix defines the minimum CLI compatibility for the MVP opencode adapter.

Legend:
- `MVP-required` - must pass in CI gate for merge.
- `Deferred` - intentionally out of MVP scope.

## Commands / Modes

| ID | Scenario | Current ralphex behavior | MVP target behavior | Priority |
|---|---|---|---|---|
| MVP-CLI-01 | default run with positional `plan-file` | runs full loop (`ModeFull`) | same mode selection | MVP-required |
| MVP-CLI-02 | `--tasks-only` / `-t` | runs tasks only (`ModeTasksOnly`) | same mode selection | MVP-required |
| MVP-CLI-03 | `--review` / `-r` | runs review pipeline (`ModeReview`) | same mode selection | MVP-required |
| MVP-CLI-04 | `--external-only` / `-e` | runs external review only (`ModeCodexOnly`) | same mode selection | MVP-required |
| MVP-CLI-05 | `--codex-only` / `-c` | alias for external-only | same alias semantics | MVP-required |
| MVP-CLI-06 | `--plan "..."` | interactive plan creation (`ModePlan`) | keep existing path for MVP | Deferred |
| MVP-CLI-07 | `--serve` with `--watch` and no plan | watch-only web mode | keep existing path for MVP | Deferred |

## Flag Compatibility

| ID | Flag | Current behavior | MVP target | Priority |
|---|---|---|---|---|
| MVP-FLAG-01 | `-m`, `--max-iterations` | set max task iterations | preserve semantics | MVP-required |
| MVP-FLAG-02 | `--max-external-iterations` | override external review limit | preserve semantics | MVP-required |
| MVP-FLAG-03 | `-b`, `--base-ref` | override diff/review base ref | preserve semantics | MVP-required |
| MVP-FLAG-04 | `--wait` | retry delay on rate limit | preserve non-negative validation | MVP-required |
| MVP-FLAG-05 | `--skip-finalize` | disable finalize step for run | preserve semantics | MVP-required |
| MVP-FLAG-06 | `-d`, `--debug` | enable debug output | preserve semantics | MVP-required |
| MVP-FLAG-07 | `--no-color` | disable color output | preserve semantics | MVP-required |
| MVP-FLAG-08 | `-v`, `--version` | print version and exit 0 | preserve semantics | MVP-required |
| MVP-FLAG-09 | `--plan` with positional plan-file | validation error | preserve conflict validation | MVP-required |
| MVP-FLAG-10 | `--wait < 0` | validation error | preserve validation | MVP-required |
| MVP-FLAG-11 | `--serve`, `--port`, `--host`, `--watch` | web dashboard/watch-only flow | leave existing implementation | Deferred |
| MVP-FLAG-12 | `--reset`, `--dump-defaults`, `--config-dir` | config utility flags | leave existing implementation | Deferred |

## Exit Code Contract

| ID | Scenario | Expected exit code | Priority |
|---|---|---|---|
| MVP-EXIT-01 | successful execution or help/version | `0` | MVP-required |
| MVP-EXIT-02 | CLI parse error / validation error | `1` | MVP-required |
| MVP-EXIT-03 | runtime failure (dependency, git, provider, execution) | `1` | MVP-required |

Notes:
- Current `cmd/ralphex/main.go` uses binary exit codes `0` and `1` only.
- MVP keeps this contract unchanged.

## Feature Flag Gate

| ID | Variable | Values | Default | Target behavior | Priority |
|---|---|---|---|---|---|
| MVP-FLAG-GATE-01 | `RALPHEX_OPENCODE_ADAPTER` | `0|1` | `0` | `0`: existing engine path, `1`: opencode adapter path | MVP-required |

## CI Gate Definition

Merge is blocked unless all `MVP-required` IDs above are green in CI.

Minimum CI job groups:
- `unit-cli-compat`
- `integration-opencode-adapter`
- `e2e-cli-smoke`

Test mapping (current):
- `unit-cli-compat`: `TestDetermineMode`, `TestValidateFlags`, `TestUseOpencodeAdapter`, `TestSupportsOpencodeAdapterMode`, `TestRunWithOpencodeAdapter`
- `integration-opencode-adapter`: `TestRunTaskPhaseRunsPendingTasks`, `TestRunTaskPhasePropagatesRunnerError`, `TestRunTaskPhaseRequiresPlan`, `TestValidateTaskOutput`, `TestRunReviewPhase`, `TestRunReviewPhaseDiffError`, `TestBuildReviewPrompt`, `TestGitDiffOutput`, `TestNewOpencodeAdapter`
- `e2e-cli-smoke`: `TestExecOpencodeRunnerSmoke`
