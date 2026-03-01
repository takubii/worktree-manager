package opener

import (
	"context"
	"fmt"
	"strings"
)

func (o *defaultOpener) openTerminal(
	ctx context.Context,
	path string,
	window WindowMode,
	rawProvider string,
) (OpenResult, error) {
	provider, err := o.resolveTerminalProvider(rawProvider)
	if err != nil {
		return OpenResult{}, err
	}

	result := OpenResult{Provider: provider}
	result.Warnings = append(result.Warnings, o.windowWarning(provider, window)...)

	if err := o.runTerminalProvider(ctx, provider, path, window); err != nil {
		return OpenResult{}, err
	}
	return result, nil
}

func (o *defaultOpener) resolveTerminalProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	if provider == "" {
		provider = TerminalProviderAuto
	}
	if provider == TerminalProviderAuto {
		return o.resolveAutoTerminalProvider()
	}

	if !isSupportedTerminalProvider(provider) {
		return "", fmt.Errorf(
			"unknown terminal provider %q. Use one of: auto, windows-terminal, cmd, powershell, terminal, gnome-terminal, wezterm, iterm2, ghostty, warp, tabby",
			raw,
		)
	}
	return provider, nil
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
		return o.lookPathAvailable("wt")
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
		if o.goos != "windows" {
			return fmt.Errorf("terminal provider %q is not supported on %s", provider, o.goos)
		}
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
