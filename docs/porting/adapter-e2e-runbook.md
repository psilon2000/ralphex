# Adapter E2E Runbook

This runbook verifies the opencode adapter path end-to-end on a disposable toy repo.

## 1) Build ralphex

```bash
make build
```

## 2) Prepare toy repository

```bash
./scripts/prep-toy-test.sh
cd /tmp/ralphex-test
```

## 3) Configure adapter mode for this shell

```bash
export RALPHEX_OPENCODE_ADAPTER=1
export RALPHEX_OPENCODE_COMMAND=opencode
# optional extra flags
# export RALPHEX_OPENCODE_ARGS='exec --sandbox workspace-write'
```

## 4) Run task-only adapter flow

```bash
/mnt/d/repository/ralphex-opencode/ralphex/.bin/ralphex --tasks-only docs/plans/fix-issues.md
```

Expected:
- starts adapter task phase
- task execution output contains `<<<RALPHEX:ALL_TASKS_DONE>>>`
- process exits with code `0`

## 5) Run review adapter flow

```bash
/mnt/d/repository/ralphex-opencode/ralphex/.bin/ralphex --review --base-ref master
```

Expected:
- starts adapter review phase
- output contains `<<<RALPHEX:REVIEW_DONE>>>`
- process exits with code `0`

## 6) Negative check (failure signal)

Set adapter command to a script/tool that returns `<<<RALPHEX:TASK_FAILED>>>` and rerun task-only mode.

Expected:
- process exits with code `1`
- error mentions failed signal from opencode adapter

## 7) Restore default path

```bash
unset RALPHEX_OPENCODE_ADAPTER
unset RALPHEX_OPENCODE_COMMAND
unset RALPHEX_OPENCODE_ARGS
```
