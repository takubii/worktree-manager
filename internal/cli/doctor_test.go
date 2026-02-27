package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/doctor"
)

type fakeDoctorService struct {
	report doctor.Report
	calls  int
}

func (f *fakeDoctorService) Run(context.Context) doctor.Report {
	f.calls++
	return f.report
}

func TestDoctorCommand_PrintsReportAndSucceedsWithoutCritical(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fake := &fakeDoctorService{
		report: doctor.Report{
			Results: []doctor.Result{
				{
					Name:       "git",
					Level:      doctor.LevelOK,
					Message:    "`git` command is available",
					NextAction: "no action required",
				},
			},
			HasCritical: false,
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Doctor: fake,
	})
	cmd.SetArgs([]string{"doctor"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected doctor service to be called once, got %d", fake.calls)
	}
	if !strings.Contains(stdout.String(), "[OK] git: `git` command is available | next: no action required") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestDoctorCommand_ReturnsErrorWhenCriticalExists(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fake := &fakeDoctorService{
		report: doctor.Report{
			Results: []doctor.Result{
				{
					Name:       "git",
					Level:      doctor.LevelCritical,
					Message:    "`git` command was not found in PATH",
					NextAction: "install Git",
				},
			},
			HasCritical: true,
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Doctor: fake,
	})
	cmd.SetArgs([]string{"doctor"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "doctor found critical issues") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "[CRIT] git: `git` command was not found in PATH | next: install Git") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
