package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/takubii/worktree-manager/internal/config"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args []string, cwd string, stdout io.Writer, stderr io.Writer) error
}

type defaultCommandRunner struct{}

type createBootstrapContext struct {
	repoRoot     string
	worktreePath string
	branch       string
	dryRun       bool
}

func runCreateBootstrap(
	ctx context.Context,
	cfg config.Bootstrap,
	bootstrapCtx createBootstrapContext,
	runner commandRunner,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if !hasBootstrapActions(cfg) {
		return nil
	}

	for _, action := range cfg.CopyFiles {
		if err := runCopyFileAction(action, bootstrapCtx, stderr); err != nil {
			return err
		}
	}

	for _, action := range cfg.PostCreate {
		if err := runHookAction(ctx, action, bootstrapCtx, runner, stdout, stderr); err != nil {
			return err
		}
	}

	return nil
}

func hasBootstrapActions(cfg config.Bootstrap) bool {
	return len(cfg.CopyFiles) > 0 || len(cfg.PostCreate) > 0
}

func runCopyFileAction(action config.CopyFileAction, bootstrapCtx createBootstrapContext, stderr io.Writer) error {
	source, err := resolveBootstrapSourcePath(action.From, bootstrapCtx)
	if err != nil {
		return err
	}
	destination, err := resolveWorktreeContainedPath(action.To, bootstrapCtx)
	if err != nil {
		return fmt.Errorf("invalid bootstrap copy destination %q: %w", action.To, err)
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if action.Required {
				return fmt.Errorf("required bootstrap file is missing: %s. Create the file or remove it from create.bootstrap.copyFiles, then retry", source)
			}
			warnf(stderr, "optional bootstrap file is missing and was skipped: %s", source)
			return nil
		}
		return fmt.Errorf("failed to inspect bootstrap source file %s: %w", source, err)
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("bootstrap source is a directory, but only files are supported in v0.6.0: %s", source)
	}

	if _, err := os.Stat(destination); err == nil {
		if !action.Overwrite {
			warnf(stderr, "bootstrap destination already exists and was skipped: %s", destination)
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect bootstrap destination file %s: %w", destination, err)
	}

	if bootstrapCtx.dryRun {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("failed to create bootstrap destination directory for %s: %w", destination, err)
	}

	return copyRegularFile(source, destination, sourceInfo.Mode().Perm())
}

func runHookAction(
	ctx context.Context,
	action config.HookAction,
	bootstrapCtx createBootstrapContext,
	runner commandRunner,
	stdout io.Writer,
	stderr io.Writer,
) error {
	command, err := renderCommandArgs(action.Command, bootstrapCtx)
	if err != nil {
		return err
	}
	if len(command) == 0 {
		return fmt.Errorf("bootstrap post-create command is empty. Provide argv such as [\"npm\", \"install\"]")
	}
	cwd, err := resolveHookCWD(action.CWD, bootstrapCtx)
	if err != nil {
		return err
	}

	if bootstrapCtx.dryRun {
		return nil
	}

	if runner == nil {
		runner = defaultCommandRunner{}
	}

	if err := runner.Run(ctx, command[0], command[1:], cwd, stdout, stderr); err != nil {
		name := strings.TrimSpace(action.Name)
		if name == "" {
			name = strings.Join(command, " ")
		}
		return fmt.Errorf(
			"post-create hook %q failed in %s while running `%s`: %w. Fix the command or run it manually in the worktree, then retry as needed",
			name,
			cwd,
			strings.Join(command, " "),
			err,
		)
	}

	return nil
}

func (defaultCommandRunner) Run(ctx context.Context, name string, args []string, cwd string, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func resolveBootstrapSourcePath(raw string, bootstrapCtx createBootstrapContext) (string, error) {
	rendered := renderBootstrapPlaceholders(raw, bootstrapCtx)
	if strings.TrimSpace(rendered) == "" {
		return "", fmt.Errorf("bootstrap source path is empty. Fix create.bootstrap.copyFiles and retry")
	}
	if !filepath.IsAbs(rendered) {
		rendered = filepath.Join(bootstrapCtx.repoRoot, rendered)
	}
	return filepath.Clean(rendered), nil
}

func resolveWorktreeContainedPath(raw string, bootstrapCtx createBootstrapContext) (string, error) {
	rendered := renderBootstrapPlaceholders(raw, bootstrapCtx)
	if strings.TrimSpace(rendered) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(rendered) {
		rendered = filepath.Join(bootstrapCtx.worktreePath, rendered)
	}
	resolved := filepath.Clean(rendered)
	if !isPathWithinBase(resolved, bootstrapCtx.worktreePath) {
		return "", fmt.Errorf("path resolves outside the worktree: %s. Use a path inside {worktree}", resolved)
	}
	return resolved, nil
}

func resolveHookCWD(raw string, bootstrapCtx createBootstrapContext) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return filepath.Clean(bootstrapCtx.worktreePath), nil
	}
	cwd, err := resolveWorktreeContainedPath(raw, bootstrapCtx)
	if err != nil {
		return "", fmt.Errorf("invalid bootstrap hook cwd %q: %w", raw, err)
	}
	if bootstrapCtx.dryRun {
		return cwd, nil
	}
	info, err := os.Stat(cwd)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("bootstrap hook cwd does not exist: %s. Create the directory or update create.bootstrap.postCreate.cwd, then retry", cwd)
		}
		return "", fmt.Errorf("failed to inspect bootstrap hook cwd %s: %w", cwd, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("bootstrap hook cwd is not a directory: %s. Choose a directory inside the worktree", cwd)
	}
	return cwd, nil
}

func renderCommandArgs(args []string, bootstrapCtx createBootstrapContext) ([]string, error) {
	rendered := make([]string, 0, len(args))
	for _, arg := range args {
		value := renderBootstrapPlaceholders(arg, bootstrapCtx)
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("bootstrap post-create command contains an empty argument after placeholder rendering")
		}
		rendered = append(rendered, value)
	}
	return rendered, nil
}

func renderBootstrapPlaceholders(value string, bootstrapCtx createBootstrapContext) string {
	rendered := value
	rendered = strings.ReplaceAll(rendered, "{repoRoot}", bootstrapCtx.repoRoot)
	rendered = strings.ReplaceAll(rendered, "{worktree}", bootstrapCtx.worktreePath)
	rendered = strings.ReplaceAll(rendered, "{branch}", filepath.FromSlash(bootstrapCtx.branch))
	return rendered
}

func isPathWithinBase(path string, base string) bool {
	path = filepath.Clean(path)
	base = filepath.Clean(base)
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func copyRegularFile(source string, destination string, perm os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open bootstrap source file %s: %w", source, err)
	}
	defer in.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open bootstrap destination file %s: %w", destination, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("failed to copy bootstrap file from %s to %s: %w", source, destination, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close bootstrap destination file %s: %w", destination, err)
	}

	return nil
}

func warnf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
}
