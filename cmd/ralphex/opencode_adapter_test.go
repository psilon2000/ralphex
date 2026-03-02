package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/ralphex/pkg/plan"
	"github.com/umputun/ralphex/pkg/processor"
	"github.com/umputun/ralphex/pkg/status"
)

type testOpencodeRunner struct {
	calls [][]string
	err   error
	out   string
}

func (r *testOpencodeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	call := []string{name}
	call = append(call, args...)
	r.calls = append(r.calls, call)
	if r.err != nil {
		return r.out, r.err
	}
	return r.out, nil
}

type testLogger struct{}

func (l testLogger) Print(_ string, _ ...any)         {}
func (l testLogger) PrintRaw(_ string, _ ...any)      {}
func (l testLogger) PrintSection(_ status.Section)    {}
func (l testLogger) PrintAligned(_ string)            {}
func (l testLogger) LogQuestion(_ string, _ []string) {}
func (l testLogger) LogAnswer(_ string)               {}
func (l testLogger) LogDraftReview(_, _ string)       {}
func (l testLogger) Path() string                     { return "" }

func TestPendingTasks(t *testing.T) {
	pl := &plan.Plan{Tasks: []plan.Task{
		{Number: 1, Status: plan.TaskStatusPending},
		{Number: 2, Status: plan.TaskStatusDone},
		{Number: 3, Status: plan.TaskStatusActive},
	}}

	result := pendingTasks(pl)
	require.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Number)
	assert.Equal(t, 3, result[1].Number)
}

func TestBuildTaskPrompt(t *testing.T) {
	task := plan.Task{
		Number: 7,
		Title:  "implement adapter",
		Checkboxes: []plan.Checkbox{
			{Text: "add env switch"},
			{Text: "add tests"},
		},
	}

	prompt := buildTaskPrompt("port plan", task)
	assert.Contains(t, prompt, "Plan: port plan")
	assert.Contains(t, prompt, "Task 7: implement adapter")
	assert.Contains(t, prompt, "- add env switch")
	assert.Contains(t, prompt, status.Completed)
}

func TestRunTaskPhaseRequiresPlan(t *testing.T) {
	a := &opencodeAdapter{command: "opencode", runner: &testOpencodeRunner{}}
	err := a.runTaskPhase(context.Background(), "", testLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires plan file")
}

func TestRunTaskPhaseRunsPendingTasks(t *testing.T) {
	planFile := filepath.Join(t.TempDir(), "plan.md")
	content := "# test plan\n\n### Task 1: first\n- [ ] do one\n\n### Task 2: done\n- [x] done\n"
	require.NoError(t, os.WriteFile(planFile, []byte(content), 0o600))

	runner := &testOpencodeRunner{out: status.Completed}
	a := &opencodeAdapter{command: "opencode", args: []string{"exec"}, runner: runner}
	err := a.runTaskPhase(context.Background(), planFile, testLogger{})
	require.NoError(t, err)
	require.Len(t, runner.calls, 1)
	assert.Equal(t, "opencode", runner.calls[0][0])
	assert.Equal(t, "exec", runner.calls[0][1])
	assert.Contains(t, runner.calls[0][2], "Task 1: first")
}

func TestRunTaskPhasePropagatesRunnerError(t *testing.T) {
	planFile := filepath.Join(t.TempDir(), "plan.md")
	content := "# test plan\n\n### Task 1: first\n- [ ] do one\n"
	require.NoError(t, os.WriteFile(planFile, []byte(content), 0o600))

	runner := &testOpencodeRunner{err: errors.New("boom"), out: "stderr output"}
	a := &opencodeAdapter{command: "opencode", runner: runner}
	err := a.runTaskPhase(context.Background(), planFile, testLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode task 1 failed")
	assert.Contains(t, err.Error(), "stderr output")
}

func TestValidateTaskOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantError bool
		errPart   string
	}{
		{name: "completed", output: "ok\n" + status.Completed, wantError: false},
		{name: "missing_completed", output: "plain output", wantError: true, errPart: "missing expected signal"},
		{name: "failed_signal", output: status.Failed, wantError: true, errPart: "returned failed signal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTaskOutput(tc.output, 3)
			if tc.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errPart)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildReviewPrompt(t *testing.T) {
	prompt, sig := buildReviewPrompt(processor.ModeReview, "docs/plans/p.md", "main", "diff")
	assert.Equal(t, status.ReviewDone, sig)
	assert.Contains(t, prompt, "Mode: review")
	assert.Contains(t, prompt, status.ReviewDone)

	prompt, sig = buildReviewPrompt(processor.ModeCodexOnly, "", "main", "diff")
	assert.Equal(t, status.CodexDone, sig)
	assert.Contains(t, prompt, "Mode: codex-only")
	assert.Contains(t, prompt, status.CodexDone)
}

func TestRunReviewPhase(t *testing.T) {
	tests := []struct {
		name      string
		mode      processor.Mode
		runnerOut string
		wantErr   string
	}{
		{name: "review_ok", mode: processor.ModeReview, runnerOut: status.ReviewDone},
		{name: "codex_ok", mode: processor.ModeCodexOnly, runnerOut: status.CodexDone},
		{name: "missing_signal", mode: processor.ModeReview, runnerOut: "plain", wantErr: "missing expected signal"},
		{name: "failed_signal", mode: processor.ModeReview, runnerOut: status.Failed, wantErr: "returned failed signal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner := &testOpencodeRunner{out: tc.runnerOut}
			a := &opencodeAdapter{
				command: "opencode",
				runner:  runner,
				diffProvider: func(_ context.Context, _ string) (string, error) {
					return "diff --git a/x b/x", nil
				},
			}
			err := a.runReviewPhase(context.Background(), tc.mode, "docs/plans/p.md", "main", testLogger{})
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, runner.calls, 1)
			assert.Equal(t, "opencode", runner.calls[0][0])
		})
	}
}

func TestRunReviewPhaseDiffError(t *testing.T) {
	a := &opencodeAdapter{
		command: "opencode",
		runner:  &testOpencodeRunner{out: status.ReviewDone},
		diffProvider: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("diff failed")
		},
	}
	err := a.runReviewPhase(context.Background(), processor.ModeReview, "", "main", testLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collect diff for review")
}

func TestExecOpencodeRunnerSmoke(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell smoke test is unix-specific")
	}
	r := execOpencodeRunner{}
	out, err := r.Run(context.Background(), "sh", "-c", "printf '%s' \""+status.Completed+"\"")
	require.NoError(t, err)
	assert.Contains(t, out, status.Completed)
}

func TestGitDiffOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	runGitCmd(t, repo, "init")
	runGitCmd(t, repo, "config", "user.email", "test@example.com")
	runGitCmd(t, repo, "config", "user.name", "test")

	f := filepath.Join(repo, "a.txt")
	require.NoError(t, os.WriteFile(f, []byte("one\n"), 0o600))
	runGitCmd(t, repo, "add", "a.txt")
	runGitCmd(t, repo, "commit", "-m", "init")

	require.NoError(t, os.WriteFile(f, []byte("one\ntwo\n"), 0o600))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repo))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	diff, err := gitDiffOutput(context.Background(), "")
	require.NoError(t, err)
	assert.Contains(t, diff, "+two")
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
