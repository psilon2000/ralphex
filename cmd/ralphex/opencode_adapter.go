package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/umputun/ralphex/pkg/config"
	"github.com/umputun/ralphex/pkg/plan"
	"github.com/umputun/ralphex/pkg/processor"
	"github.com/umputun/ralphex/pkg/status"
)

const (
	opencodeCommandEnv = "RALPHEX_OPENCODE_COMMAND"
	opencodeArgsEnv    = "RALPHEX_OPENCODE_ARGS"
)

type opencodeRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type execOpencodeRunner struct{}

func (r execOpencodeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("run %s: %w", name, err)
	}
	return string(out), nil
}

type opencodeAdapter struct {
	command      string
	args         []string
	runner       opencodeRunner
	diffProvider func(ctx context.Context, baseRef string) (string, error)
}

func newOpencodeAdapter(cfg *config.Config) *opencodeAdapter {
	command := ""
	args := ""
	if cfg != nil {
		command = strings.TrimSpace(cfg.OpencodeCommand)
		args = strings.TrimSpace(cfg.OpencodeArgs)
	}

	envCommand := strings.TrimSpace(os.Getenv(opencodeCommandEnv))
	if envCommand != "" {
		command = envCommand
	}
	if command == "" {
		command = "opencode"
	}

	argStr := strings.TrimSpace(os.Getenv(opencodeArgsEnv))
	if argStr == "" {
		argStr = args
	}

	return &opencodeAdapter{
		command:      command,
		args:         strings.Fields(argStr),
		runner:       execOpencodeRunner{},
		diffProvider: gitDiffOutput,
	}
}

func (a *opencodeAdapter) runTaskPhase(ctx context.Context, planFile string, log processor.Logger) error {
	if planFile == "" {
		return fmt.Errorf("opencode adapter requires plan file")
	}
	pl, err := plan.ParsePlanFile(planFile)
	if err != nil {
		return fmt.Errorf("parse plan file: %w", err)
	}

	pending := pendingTasks(pl)
	if len(pending) == 0 {
		log.Print("opencode adapter: no pending tasks found")
		return nil
	}

	for _, task := range pending {
		prompt := buildTaskPrompt(pl.Title, task)
		args := append([]string{}, a.args...)
		args = append(args, prompt)
		output, runErr := a.runner.Run(ctx, a.command, args...)
		if runErr != nil {
			return fmt.Errorf("opencode task %d failed: %w\n%s", task.Number, runErr, strings.TrimSpace(output))
		}
		if sigErr := validateTaskOutput(output, task.Number); sigErr != nil {
			return sigErr
		}
		if strings.TrimSpace(output) != "" {
			log.PrintRaw("%s", output)
		}
	}

	return nil
}

func validateTaskOutput(output string, taskNumber int) error {
	if sigErr := validateSignal(output, status.Completed, fmt.Sprintf("task %d", taskNumber)); sigErr != nil {
		return sigErr
	}
	return nil
}

func validateSignal(output, expectedSignal, phase string) error {
	if strings.Contains(output, status.Failed) {
		return fmt.Errorf("opencode %s returned failed signal", phase)
	}
	if !strings.Contains(output, expectedSignal) {
		return fmt.Errorf("opencode %s missing expected signal %s", phase, expectedSignal)
	}
	return nil
}

func (a *opencodeAdapter) runReviewPhase(ctx context.Context, mode processor.Mode,
	planFile, baseRef string, log processor.Logger) error {
	diffText, err := a.diffProvider(ctx, baseRef)
	if err != nil {
		return fmt.Errorf("collect diff for review: %w", err)
	}
	if strings.TrimSpace(diffText) == "" {
		log.Print("opencode adapter: empty diff for review")
	}

	prompt, expectedSignal := buildReviewPrompt(mode, planFile, baseRef, diffText)
	args := append([]string{}, a.args...)
	args = append(args, prompt)
	output, runErr := a.runner.Run(ctx, a.command, args...)
	if runErr != nil {
		return fmt.Errorf("opencode review failed: %w\n%s", runErr, strings.TrimSpace(output))
	}
	if sigErr := validateSignal(output, expectedSignal, string(mode)); sigErr != nil {
		return sigErr
	}
	if strings.TrimSpace(output) != "" {
		log.PrintRaw("%s", output)
	}
	return nil
}

func buildReviewPrompt(mode processor.Mode, planFile, baseRef, diffText string) (string, string) {
	var b strings.Builder
	expectedSignal := status.ReviewDone
	b.WriteString("You are running ralphex review phase in MVP adapter mode.\n")
	if mode == processor.ModeCodexOnly {
		expectedSignal = status.CodexDone
		b.WriteString("Mode: codex-only\n")
	} else {
		b.WriteString("Mode: review\n")
	}
	if strings.TrimSpace(planFile) != "" {
		b.WriteString("Plan file: ")
		b.WriteString(planFile)
		b.WriteString("\n")
	}
	if strings.TrimSpace(baseRef) != "" {
		b.WriteString("Base ref: ")
		b.WriteString(baseRef)
		b.WriteString("\n")
	}
	b.WriteString("Git diff to review:\n")
	b.WriteString(diffText)
	b.WriteString("\nReturn exactly one signal at the end: ")
	b.WriteString(expectedSignal)
	b.WriteString(" on success or ")
	b.WriteString(status.Failed)
	b.WriteString(" on failure.")
	return b.String(), expectedSignal
}

func gitDiffOutput(ctx context.Context, baseRef string) (string, error) {
	args := []string{"diff"}
	if strings.TrimSpace(baseRef) != "" {
		args = append(args, baseRef+"...HEAD")
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func pendingTasks(pl *plan.Plan) []plan.Task {
	out := make([]plan.Task, 0, len(pl.Tasks))
	for _, t := range pl.Tasks {
		if t.Status != plan.TaskStatusDone {
			out = append(out, t)
		}
	}
	return out
}

func buildTaskPrompt(planTitle string, task plan.Task) string {
	var b strings.Builder
	b.WriteString("You are executing a ralphex plan task in MVP adapter mode.\n")
	if strings.TrimSpace(planTitle) != "" {
		b.WriteString("Plan: ")
		b.WriteString(planTitle)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Task %d: %s\n", task.Number, task.Title))
	if len(task.Checkboxes) > 0 {
		b.WriteString("Checklist:\n")
		for _, cb := range task.Checkboxes {
			b.WriteString("- ")
			b.WriteString(cb.Text)
			b.WriteString("\n")
		}
	}
	b.WriteString("Return exactly one signal at the end: ")
	b.WriteString(status.Completed)
	b.WriteString(" on success or ")
	b.WriteString(status.Failed)
	b.WriteString(" on failure.")
	return b.String()
}
