package updater

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
)

const (
	installScriptURLUnix    = "https://raw.githubusercontent.com/takubii/git-worktree-opener/main/scripts/install.sh"
	installScriptURLWindows = "https://raw.githubusercontent.com/takubii/git-worktree-opener/main/scripts/install.ps1"
)

// Request defines updater inputs.
type Request struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

// Result describes updater execution mode.
type Result struct {
	Async bool
}

// Service updates the current wto installation.
type Service interface {
	Update(ctx context.Context, req Request) (Result, error)
}

// Installer updates wto by re-running platform install scripts.
type Installer struct {
	goos           string
	commandContext func(ctx context.Context, name string, args ...string) *exec.Cmd
	runCommand     func(cmd *exec.Cmd) error
	startCommand   func(cmd *exec.Cmd) error
}

// NewInstaller returns the default updater implementation.
func NewInstaller() Service {
	return &Installer{
		goos:           runtime.GOOS,
		commandContext: exec.CommandContext,
		runCommand:     (*exec.Cmd).Run,
		startCommand:   (*exec.Cmd).Start,
	}
}

// Update updates wto to the latest release unless a specific version is provided.
func (u *Installer) Update(ctx context.Context, req Request) (Result, error) {
	version := strings.TrimSpace(req.Version)

	if u.goos == "windows" {
		return u.updateWindows(ctx, version, req.Stdout, req.Stderr)
	}
	return u.updateUnix(ctx, version, req.Stdout, req.Stderr)
}

func (u *Installer) updateUnix(ctx context.Context, version string, stdout io.Writer, stderr io.Writer) (Result, error) {
	script := buildUnixInstallCommand(version)

	cmd := u.commandContext(ctx, "sh", "-c", script)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := u.runCommand(cmd); err != nil {
		return Result{}, execerr.Build(
			"sh -c installer script",
			err.Error(),
			"run `curl -fsSL "+installScriptURLUnix+" | sh` manually and retry",
		)
	}

	return Result{Async: false}, nil
}

func (u *Installer) updateWindows(ctx context.Context, version string, stdout io.Writer, stderr io.Writer) (Result, error) {
	parts := []string{
		"$ErrorActionPreference = 'Stop'",
		"Start-Sleep -Milliseconds 300",
	}
	if version != "" {
		parts = append(parts, "$env:WTO_VERSION = '"+powerShellSingleQuote(version)+"'")
	}
	parts = append(parts, "iwr "+installScriptURLWindows+" -UseBasicParsing | iex")

	cmd := u.commandContext(
		ctx,
		"powershell",
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-Command",
		strings.Join(parts, "; "),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := u.startCommand(cmd); err != nil {
		return Result{}, execerr.Build(
			"powershell installer bootstrap",
			err.Error(),
			"run `iwr "+installScriptURLWindows+" -UseBasicParsing | iex` in PowerShell and retry",
		)
	}

	return Result{Async: true}, nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func powerShellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func buildUnixInstallCommand(version string) string {
	envPrefix := ""
	if version != "" {
		envPrefix = "WTO_VERSION=" + shellSingleQuote(version) + " "
	}

	return envPrefix + "if command -v curl >/dev/null 2>&1; then " +
		"curl -fsSL " + installScriptURLUnix + " | sh; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -qO- " + installScriptURLUnix + " | sh; " +
		"else echo 'curl or wget is required to update wto' >&2; exit 1; fi"
}
