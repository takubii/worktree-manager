package opener

import (
	"context"
	"fmt"
	"strings"
)

const (
	terminalProviderTmux = "tmux"
)

func (o *defaultOpener) openTerminal(
	ctx context.Context,
	path string,
	window WindowMode,
	rawProvider string,
	tmuxMode TmuxMode,
) (OpenResult, error) {
	provider := normalizeTerminalProvider(rawProvider)
	if !isSupportedTerminalProvider(provider) {
		return OpenResult{}, fmt.Errorf(
			"unknown terminal provider %q. Use one of: auto, windows-terminal, cmd, powershell, terminal, gnome-terminal, wezterm, iterm2, ghostty, warp, tabby",
			rawProvider,
		)
	}

	usedTmux, tmuxWarnings, err := o.tryTmuxOpen(ctx, path, window, tmuxMode)
	if err != nil {
		return OpenResult{}, err
	}
	if usedTmux {
		return OpenResult{
			Provider: terminalProviderTmux,
			Warnings: tmuxWarnings,
		}, nil
	}

	provider, err = o.resolveTerminalProvider(provider)
	if err != nil {
		return OpenResult{}, err
	}

	if err := o.ensureTerminalLaunchPrerequisites(provider); err != nil {
		return OpenResult{}, err
	}

	result := OpenResult{Provider: provider}
	result.Warnings = append(result.Warnings, tmuxWarnings...)
	result.Warnings = append(result.Warnings, o.windowWarning(provider, window)...)

	if err := o.runTerminalProvider(ctx, provider, path, window); err != nil {
		return OpenResult{}, err
	}
	return result, nil
}

func (o *defaultOpener) resolveTerminalProvider(raw string) (string, error) {
	provider := normalizeTerminalProvider(raw)
	if provider == TerminalProviderAuto {
		return o.resolveAutoTerminalProvider()
	}
	return provider, nil
}

func normalizeTerminalProvider(raw string) string {
	provider := strings.ToLower(strings.TrimSpace(raw))
	if provider == "" {
		return TerminalProviderAuto
	}
	return provider
}

func isSupportedTerminalProvider(provider string) bool {
	switch provider {
	case TerminalProviderAuto,
		TerminalProviderWindowsTerminal,
		TerminalProviderCMD,
		TerminalProviderPowerShell,
		TerminalProviderMacTerminal,
		TerminalProviderGNOMETerminal,
		TerminalProviderWezTerm,
		TerminalProviderITerm2,
		TerminalProviderGhostty,
		TerminalProviderWarp,
		TerminalProviderTabby:
		return true
	default:
		return false
	}
}

func (o *defaultOpener) resolveAutoTerminalProvider() (string, error) {
	switch o.goos {
	case "windows":
		for _, candidate := range []string{
			TerminalProviderWindowsTerminal,
			TerminalProviderCMD,
			TerminalProviderPowerShell,
		} {
			if o.hasTerminalProvider(candidate) {
				return candidate, nil
			}
		}
	case "darwin":
		if o.hasTerminalProvider(TerminalProviderMacTerminal) {
			return TerminalProviderMacTerminal, nil
		}
	case "linux":
		if o.isWSL2() {
			if o.hasTerminalProvider(TerminalProviderWindowsTerminal) {
				return TerminalProviderWindowsTerminal, nil
			}
			if !o.hasLinuxGUISession() {
				return "", fmt.Errorf("no terminal provider is available for auto mode in WSL2: Windows Terminal bridge (`wt.exe`) is unavailable and Linux GUI session is not detected (`DISPLAY`/`WAYLAND_DISPLAY`). Install Windows Terminal or enable Linux GUI terminal support, then retry")
			}
		} else if !o.hasLinuxGUISession() {
			return "", fmt.Errorf("terminal opener requires a Linux GUI session (`DISPLAY` or `WAYLAND_DISPLAY`). Start a desktop session or use `wto enter --shell` / `wto enter --print-cd`, then retry")
		}

		for _, candidate := range []string{
			TerminalProviderGNOMETerminal,
			terminalProviderXTerminalEmulator,
			terminalProviderXTerm,
		} {
			if o.hasTerminalProvider(candidate) {
				return candidate, nil
			}
		}
	default:
		return "", fmt.Errorf("terminal opener is not supported on this platform")
	}

	return "", fmt.Errorf("no terminal provider is available for auto mode. Install a supported terminal or choose a different opener")
}

func (o *defaultOpener) hasTerminalProvider(provider string) bool {
	switch provider {
	case TerminalProviderWindowsTerminal:
		if o.goos == "windows" {
			return o.lookPathAvailable("wt")
		}
		if o.goos == "linux" && o.isWSL2() {
			return o.lookPathAvailable("wt.exe") || o.lookPathAvailable("wt")
		}
		return false
	case TerminalProviderCMD:
		return o.lookPathAvailable("cmd")
	case TerminalProviderPowerShell:
		return o.lookPathAvailable("powershell")
	case TerminalProviderMacTerminal:
		return o.goos == "darwin" && o.lookPathAvailable("open")
	case TerminalProviderGNOMETerminal:
		return o.goos == "linux" && o.lookPathAvailable("gnome-terminal")
	case terminalProviderXTerminalEmulator:
		return o.goos == "linux" && o.lookPathAvailable("x-terminal-emulator")
	case terminalProviderXTerm:
		return o.goos == "linux" && o.lookPathAvailable("xterm")
	case TerminalProviderWezTerm:
		return o.lookPathAvailable("wezterm")
	case TerminalProviderITerm2:
		return o.goos == "darwin" && o.lookPathAvailable("open")
	case TerminalProviderGhostty:
		if o.lookPathAvailable("ghostty") {
			return true
		}
		return o.goos == "darwin" && o.lookPathAvailable("open")
	case TerminalProviderWarp:
		if o.lookPathAvailable("warp") {
			return true
		}
		return o.goos == "darwin" && o.lookPathAvailable("open")
	case TerminalProviderTabby:
		if o.lookPathAvailable("tabby") {
			return true
		}
		return o.goos == "darwin" && o.lookPathAvailable("open")
	default:
		return false
	}
}

func (o *defaultOpener) runTerminalProvider(ctx context.Context, provider, path string, window WindowMode) error {
	switch provider {
	case TerminalProviderWindowsTerminal:
		switch o.goos {
		case "windows":
			if _, err := o.lookPath("wt"); err != nil {
				return fmt.Errorf("terminal provider %q is unavailable: install Windows Terminal (`wt`) and retry", provider)
			}
			if window == WindowReuse {
				if err := o.run(ctx, "wt", "-w", "0", "nt", "-d", path); err != nil {
					return o.run(ctx, "wt", "-d", path)
				}
				return nil
			}
			return o.run(ctx, "wt", "-d", path)
		case "linux":
			if !o.isWSL2() {
				return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
			}

			wtCommand, ok := o.findAvailableCommand("wt.exe", "wt")
			if !ok {
				return fmt.Errorf("terminal provider %q is unavailable in WSL2: install Windows Terminal (`wt.exe`) and retry", provider)
			}
			wslCommand, ok := o.findAvailableCommand("wsl.exe", "wsl")
			if !ok {
				return fmt.Errorf("terminal provider %q requires `wsl.exe` in PATH for WSL2 bridge mode", provider)
			}

			bridgeArgs := o.wslBridgeArgs(wslCommand, path)
			if window == WindowReuse {
				reuseArgs := append([]string{"-w", "0", "nt"}, bridgeArgs...)
				if err := o.run(ctx, wtCommand, reuseArgs...); err != nil {
					return o.run(ctx, wtCommand, bridgeArgs...)
				}
				return nil
			}

			return o.run(ctx, wtCommand, bridgeArgs...)
		default:
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
	case TerminalProviderCMD:
		if o.goos != "windows" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		if _, err := o.lookPath("cmd"); err != nil {
			return fmt.Errorf("terminal provider %q is unavailable: install/enable cmd and retry", provider)
		}
		command := fmt.Sprintf(`cd /d "%s"`, strings.ReplaceAll(path, `"`, `\"`))
		return o.run(ctx, "cmd", "/c", "start", "", "cmd", "/K", command)
	case TerminalProviderPowerShell:
		if o.goos != "windows" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		if _, err := o.lookPath("powershell"); err != nil {
			return fmt.Errorf("terminal provider %q is unavailable: install/enable powershell and retry", provider)
		}
		if _, err := o.lookPath("cmd"); err != nil {
			return fmt.Errorf("terminal provider %q requires `cmd` to start a new terminal window", provider)
		}
		safePath := strings.ReplaceAll(path, "'", "''")
		return o.run(
			ctx,
			"cmd",
			"/c",
			"start",
			"",
			"powershell",
			"-NoExit",
			"-Command",
			"Set-Location -LiteralPath '"+safePath+"'",
		)
	case TerminalProviderMacTerminal:
		if o.goos != "darwin" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		return o.run(ctx, "open", "-a", "Terminal", path)
	case TerminalProviderGNOMETerminal:
		if o.goos != "linux" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		return o.run(ctx, "gnome-terminal", "--working-directory="+path)
	case terminalProviderXTerminalEmulator:
		if o.goos != "linux" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		return o.run(ctx, "x-terminal-emulator", "--working-directory="+path)
	case terminalProviderXTerm:
		if o.goos != "linux" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		return o.run(ctx, "xterm", "-e", "sh", "-lc", "cd '"+strings.ReplaceAll(path, "'", "'\\''")+"' && exec \"${SHELL:-sh}\"")
	case TerminalProviderWezTerm:
		return o.run(ctx, "wezterm", "start", "--cwd", path)
	case TerminalProviderITerm2:
		if o.goos != "darwin" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
		return o.run(ctx, "open", "-a", "iTerm", path)
	case TerminalProviderGhostty:
		if _, err := o.lookPath("ghostty"); err == nil {
			return o.runInDir(ctx, path, "ghostty")
		}
		if o.goos == "darwin" {
			return o.run(ctx, "open", "-a", "Ghostty", path)
		}
		return fmt.Errorf("terminal provider %q is unavailable: install `ghostty` and retry", provider)
	case TerminalProviderWarp:
		if _, err := o.lookPath("warp"); err == nil {
			return o.runInDir(ctx, path, "warp")
		}
		if o.goos == "darwin" {
			return o.run(ctx, "open", "-a", "Warp", path)
		}
		return fmt.Errorf("terminal provider %q is unavailable: install `warp` and retry", provider)
	case TerminalProviderTabby:
		if _, err := o.lookPath("tabby"); err == nil {
			return o.runInDir(ctx, path, "tabby")
		}
		if o.goos == "darwin" {
			return o.run(ctx, "open", "-a", "Tabby", path)
		}
		return fmt.Errorf("terminal provider %q is unavailable: install `tabby` and retry", provider)
	default:
		return fmt.Errorf("terminal provider %q is not supported", provider)
	}
}

func (o *defaultOpener) tryTmuxOpen(ctx context.Context, path string, window WindowMode, mode TmuxMode) (bool, []string, error) {
	mode = normalizeTmuxMode(mode)
	if mode == TmuxModeOff {
		return false, nil, nil
	}

	if !o.isInsideTmuxSession() {
		if mode == TmuxModeAuto {
			return false, nil, nil
		}
		return false, []string{
			fmt.Sprintf("tmux mode %q was requested but no active tmux session was detected; falling back to terminal provider", mode),
		}, nil
	}

	if !o.lookPathAvailable("tmux") {
		if mode == TmuxModeAuto {
			return false, []string{
				"tmux session was detected but `tmux` command is unavailable; falling back to terminal provider",
			}, nil
		}
		return false, []string{
			fmt.Sprintf("tmux mode %q was requested but `tmux` command is unavailable; falling back to terminal provider", mode),
		}, nil
	}

	args := []string{"split-window", "-c", path}
	if tmuxOpenAction(mode, window) == TmuxModeWindow {
		args = []string{"new-window", "-c", path}
	}

	if err := o.run(ctx, "tmux", args...); err != nil {
		return false, nil, err
	}

	return true, nil, nil
}

func normalizeTmuxMode(mode TmuxMode) TmuxMode {
	if strings.TrimSpace(string(mode)) == "" {
		return TmuxModeAuto
	}
	return mode
}

func tmuxOpenAction(mode TmuxMode, window WindowMode) TmuxMode {
	switch mode {
	case TmuxModeWindow:
		return TmuxModeWindow
	case TmuxModeSplit:
		return TmuxModeSplit
	default:
		if window == WindowReuse {
			return TmuxModeSplit
		}
		return TmuxModeWindow
	}
}

func (o *defaultOpener) ensureTerminalLaunchPrerequisites(provider string) error {
	if o.goos != "linux" {
		return nil
	}
	if !terminalProviderRequiresLinuxGUI(provider) {
		return nil
	}
	if o.hasLinuxGUISession() {
		return nil
	}

	if o.isWSL2() {
		return fmt.Errorf("linux GUI session is unavailable (`DISPLAY`/`WAYLAND_DISPLAY` are empty). Configure WSLg/X11 or use `--terminal-provider windows-terminal`, then retry")
	}
	return fmt.Errorf("linux GUI session is unavailable (`DISPLAY`/`WAYLAND_DISPLAY` are empty). Start a desktop session or use `wto enter --shell` / `wto enter --print-cd`, then retry")
}

func terminalProviderRequiresLinuxGUI(provider string) bool {
	switch provider {
	case TerminalProviderGNOMETerminal,
		terminalProviderXTerminalEmulator,
		terminalProviderXTerm,
		TerminalProviderWezTerm,
		TerminalProviderGhostty,
		TerminalProviderWarp,
		TerminalProviderTabby:
		return true
	default:
		return false
	}
}

func (o *defaultOpener) isInsideTmuxSession() bool {
	if o.goos == "windows" {
		return false
	}
	return strings.TrimSpace(o.env("TMUX")) != ""
}

func (o *defaultOpener) isWSL2() bool {
	if o.goos != "linux" {
		return false
	}
	return strings.TrimSpace(o.env("WSL_INTEROP")) != "" || strings.TrimSpace(o.env("WSL_DISTRO_NAME")) != ""
}

func (o *defaultOpener) hasLinuxGUISession() bool {
	return strings.TrimSpace(o.env("DISPLAY")) != "" || strings.TrimSpace(o.env("WAYLAND_DISPLAY")) != ""
}

func (o *defaultOpener) env(key string) string {
	if o.getEnv == nil {
		return ""
	}
	return o.getEnv(key)
}

func (o *defaultOpener) findAvailableCommand(candidates ...string) (string, bool) {
	for _, candidate := range candidates {
		if o.lookPathAvailable(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func (o *defaultOpener) wslBridgeArgs(wslCommand, path string) []string {
	args := []string{wslCommand}
	distro := strings.TrimSpace(o.env("WSL_DISTRO_NAME"))
	if distro != "" {
		args = append(args, "-d", distro)
	}
	args = append(args, "--cd", path)
	return args
}

func (o *defaultOpener) windowWarning(provider string, window WindowMode) []string {
	if window != WindowReuse {
		return nil
	}

	switch provider {
	case TerminalProviderWindowsTerminal:
		return nil
	default:
		return []string{
			fmt.Sprintf("terminal provider %q does not guarantee --window reuse; opening a new terminal session", provider),
		}
	}
}

func (o *defaultOpener) lookPathAvailable(file string) bool {
	if o.lookPath == nil {
		return false
	}
	_, err := o.lookPath(file)
	return err == nil
}
