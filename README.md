# git-worktree-opener

`wto` is a CLI tool to create, list, open, and remove Git worktrees quickly.

## Install

Download prebuilt binaries from GitHub Releases:

- https://github.com/takubii/git-worktree-opener/releases
- Latest stable release: `v0.3.0`

### Quick install scripts

Linux/macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/takubii/git-worktree-opener/main/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr https://raw.githubusercontent.com/takubii/git-worktree-opener/main/scripts/install.ps1 -UseBasicParsing | iex
```

Windows (`cmd.exe`, PowerShell not required):

```bat
curl -fsSL -o install.cmd https://raw.githubusercontent.com/takubii/git-worktree-opener/main/scripts/install.cmd
install.cmd v0.3.0
```

Install script options:

- `WTO_VERSION=vX.Y.Z` installs a specific release tag
- `WTO_INSTALL_DIR=<path>` changes installation directory
- `WTO_SKIP_CHECKSUM=1` skips SHA256 verification

`install.cmd` behavior:

- Requires explicit version argument (`install.cmd vX.Y.Z`)
- Supports `WTO_INSTALL_DIR=<path>`
- Verifies SHA256 using release `checksums.txt`
- Adds install directory to current `cmd.exe` session `PATH`
- Does not modify persistent user/system `PATH`

PowerShell install script behavior:

- Adds install directory to the current session `PATH`
- Adds install directory to User `PATH` if missing

### Manual install

Release assets include OS/arch archives (`tar.gz` for Linux/macOS, `zip` for Windows) and `checksums.txt` (SHA256).

Linux/macOS manual example:

```sh
VERSION=v0.3.0
curl -LO "https://github.com/takubii/git-worktree-opener/releases/download/${VERSION}/git-worktree-opener_${VERSION}_linux_amd64.tar.gz"
tar -xzf "git-worktree-opener_${VERSION}_linux_amd64.tar.gz"
chmod +x wto
./wto --help
```

Windows (PowerShell) example:

```powershell
$Version = "v0.3.0"
Invoke-WebRequest -Uri "https://github.com/takubii/git-worktree-opener/releases/download/$Version/git-worktree-opener_${Version}_windows_amd64.zip" -OutFile "wto.zip"
Expand-Archive -Path ".\wto.zip" -DestinationPath "."
.\wto.exe --help
```

Checksum verification:

```sh
# Linux
sha256sum --check checksums.txt

# macOS
shasum -a 256 -c checksums.txt
```

```powershell
# Windows (PowerShell)
Get-FileHash .\wto.exe -Algorithm SHA256
```

`checksums.txt` is provided so you can verify the downloaded archive before installation.

## Quickstart

Run commands inside a Git repository.

1. List worktrees:

```sh
wto list
```

2. Create a worktree:

```sh
wto new
wto new feature/my-task
wto new feature/my-task --open system
wto new feature/my-task --open terminal --terminal-provider auto
```

3. Open an existing worktree:

```sh
wto open
wto open --branch feature/my-task
wto open --open terminal --terminal-provider auto
```

4. Select a worktree for terminal workflows:

```sh
wto enter
wto enter --print-cd
wto enter --shell
```

5. Remove a worktree:

```sh
wto rm
wto rm feature/my-task
```

6. Diagnose environment issues:

```sh
wto doctor
```

7. Update `wto`:

```sh
wto update
wto update --version vX.Y.Z
```

8. Show current version:

```sh
wto version
wto --version
```

## Command Reference

### `wto list`

Examples:

```sh
wto list
wto list --format table
wto list --format raw
wto list --format json
```

Default behavior:

- Runs `git worktree list --porcelain`
- Renders a readable table by default (`--format table`)

Main option:

- `--format table|raw|json`

### `wto new`

Examples:

```sh
wto new
wto new feature/my-task
wto new feature/my-task --base main --open vscode
wto new feature/my-task --open terminal --terminal-provider windows-terminal
wto new feature/my-task --output path
wto new feature/my-task --output json
```

Default behavior:

- Runs `git worktree prune --expire now`, then `git fetch origin --prune` (unless skipped via flags or config defaults)
- Uses `main` as base when creating a new branch
- Creates worktrees under `<repo-parent>/worktrees/<branch>`
- Does not open the created worktree unless `--open` is explicitly set

Main options:

- `--base <branch>`
- `--open none|system|vscode|cursor|vim|terminal`
- `--terminal-provider auto|windows-terminal|cmd|powershell|terminal|gnome-terminal|wezterm|iterm2|ghostty|warp|tabby` (only with `--open terminal`)
- `--no-fetch`
- `--no-prune`
- `--output none|path|json`

### `wto open`

Examples:

```sh
wto open
wto open --branch feature/my-task
wto open --print-cd
wto open --after "echo {path}"
wto open --output path
wto open --output json
wto open --open vscode
wto open --open cursor
wto open --open vim
wto open --open terminal --terminal-provider auto
wto open --open terminal --terminal-provider powershell
wto open --window reuse
```

Default behavior:

- Runs `git worktree prune --expire now` before listing candidates (unless `--no-prune` is set or config default disables prune)
- Skips stale (`prunable`) and missing local-path entries
- Opens selected worktree using `system` opener
- Prefers opening in a new window

Main options:

- `--branch <branch>`
- `--print-cd`
- `--after "<command>"`
- `--no-prune`
- `--output none|path|json`
- `--open system|vscode|cursor|vim|terminal`
- `--terminal-provider auto|windows-terminal|cmd|powershell|terminal|gnome-terminal|wezterm|iterm2|ghostty|warp|tabby` (only with `--open terminal`)
- `--window new|reuse`

Note:

- `--branch` opens the worktree linked to that local branch without showing the selector
- If the branch does not have a linked active worktree, `wto open --branch` returns an actionable error
- `--print-cd` prints shell navigation hints for the selected worktree path
- `--print-cd` cannot be combined with `--output`
- `--after` runs a follow-up command after open (`{path}` is replaced with the selected path)
- `--window` applies to `system`, `vscode`, `cursor`, and `terminal` (best-effort by provider)
- `--open terminal` supports provider selection via `--terminal-provider` or `open.terminalProvider`
- auto terminal provider policy:
  - Windows: `windows-terminal` (`wt`) -> `cmd` -> `powershell`
  - macOS: `terminal` (Terminal.app) only
  - Linux: `gnome-terminal` -> `x-terminal-emulator` -> `xterm`
- `ghostty`, `warp`, and `tabby` are explicit providers (not part of auto detection)
- some terminal providers do not guarantee `--window reuse`; `wto` prints a warning and continues
- `vim` currently uses best-effort behavior
- If `--open vscode` or `--open cursor` is explicitly set, missing CLI (`code` / `cursor`) returns an error (no silent fallback)

### `wto enter`

Examples:

```sh
wto enter
wto enter --branch feature/my-task
wto enter --print-cd
wto enter --shell
```

Default behavior:

- Runs `git worktree prune --expire now` before listing candidates
- Skips stale (`prunable`) entries
- Prints the selected worktree path to stdout

Main options:

- `--branch <branch>` enters the linked worktree without interactive selection
- `--print-cd` prints `cd` command hints for your shell
- `--shell` starts a subshell in the selected worktree

Note:

- A CLI command cannot directly change the current directory of its parent shell
- Use `wto enter --print-cd` when you want explicit, copyable navigation commands

### `wto rm`

Examples:

```sh
wto rm
wto rm feature/my-task
wto rm feature/my-task --dry-run
wto rm feature/my-task --delete-branch none
wto rm feature/my-task --force
```

Default behavior:

- Removes the selected worktree
- Safely deletes the local branch with `git branch -d`

Main options:

- `--delete-branch none|safe|force`
- `--force`
- `--dry-run`

### `wto config`

Examples:

```sh
wto config init
wto config init --force
wto config show
```

Default behavior:

- Config is optional (0-config works)
- `config init` creates global config file
- `config show` prints effective config as JSON

### `wto update`

Examples:

```sh
wto update
wto update --version vX.Y.Z
```

Default behavior:

- Resolves release metadata from GitHub Releases (`latest` or `--version` tag)
- Downloads platform archive + `checksums.txt`
- Verifies SHA256 checksum before applying update
- On Windows, starts replacement in background to avoid self-overwrite issues

Main option:

- `--version <tag>`

### `wto doctor`

Examples:

```sh
wto doctor
```

Default behavior:

- Runs environment checks and prints `[OK]` / `[WARN]` / `[CRIT]` with next actions
- Includes terminal provider availability checks (`terminal/*`)
- Optional providers that are not installed are shown as `[OK] ... (optional)`
- If configured terminal provider is unavailable, doctor reports `[WARN]` with remediation
- Returns non-zero only when critical checks fail

### `wto version`

Examples:

```sh
wto version
wto --version
```

Default behavior:

- Prints the current `wto` version

## Global Flags

- `--verbose`: prints trace logs to `stderr` for troubleshooting while keeping `stdout` contracts unchanged

## Configuration

Config precedence:

```text
flag > repo (.wtoconfig.json) > global (config.json) > built-in defaults
```

Note:

- `open.default` is used by `wto open`
- `open.terminalProvider` sets the default provider for `--open terminal` (`flag > config > auto`)
- `wto new` does not auto-open unless you set `--open`
- `new.fetch` / `new.prune` set defaults for `wto new` network/prune behavior
- `open.prune` sets default prune behavior for `wto open`

Config file locations:

- Global: `<os.UserConfigDir()>/git-worktree-opener/config.json`
- Repo override: `<repo-root>/.wtoconfig.json`

Supported keys:

```json
{
  "remote": "origin",
  "baseBranch": "main",
  "worktreeDirTemplate": "{repoParent}/worktrees/{branch}",
  "new": {
    "fetch": true,
    "prune": true
  },
  "open": {
    "default": "system",
    "window": "new",
    "prune": true,
    "terminalProvider": "auto"
  },
  "rm": {
    "deleteBranch": "safe"
  }
}
```

`worktreeDirTemplate` placeholders:

- `{repoParent}`
- `{repoRoot}`
- `{branch}`

If global/repo config is invalid (unknown keys or invalid values), `wto` prints a warning to `stderr` and continues with lower-priority values.

## Advanced Behavior

### `wto list` output formats

- `table` (default): marker, branch, status, short head, path
- `raw`: original `git worktree list --porcelain` output
- `json`: machine-readable array

`STATUS` values:

- `active`: entry is healthy and path exists
- `stale`: entry is marked `prunable`
- `missing`: path does not exist locally and is not marked `prunable`

### Branch selection in `wto new`

When `wto new` runs without a branch argument:

- Uses `fzf` if installed
- Otherwise uses `promptui`
- Falls back to numeric selection if interactive UI is unavailable
- Branches that already have an active linked worktree are shown with ` [worktree]`

You can create a new branch from the selector flow:

- `fzf`: type a new branch name and press Enter
- `promptui`: choose `Create a new branch` and enter a name

Entered names are validated with `git check-ref-format --branch`.

### Worktree removal behavior

- Selection shows stale entries with `[stale]`
- Selecting a stale entry cleans metadata via `git worktree prune --expire now`
- If `--force` is set and `--delete-branch` is not explicitly set, branch deletion mode also becomes force
- `wto rm` refuses removal if your current directory is inside the target worktree

## Troubleshooting

### `wto list` fails outside a Git repository

Run the command inside a Git repository directory.

### `--open vscode` or `--open cursor` fails

Install the corresponding CLI command and ensure it is on `PATH`:

- VS Code: `code`
- Cursor: `cursor`

Or use `--open system`.

### `--open terminal` fails

Check the selected provider availability:

- auto mode uses OS defaults:
  - Windows: `wt` -> `cmd` -> `powershell`
  - macOS: Terminal.app
  - Linux: `gnome-terminal` -> `x-terminal-emulator` -> `xterm`
- explicit providers require corresponding CLI/app availability (`wezterm`, `iterm2`, `ghostty`, `warp`, `tabby`)

Use `wto doctor` to verify terminal provider status and recommended next actions.

### `wto rm` refuses to remove

This happens when your current directory is inside the target worktree.
Move to another directory (for example, the repo root), then run `wto rm` again.

### `wto update` fails

Use `wto doctor` first, then verify:

- network access to GitHub Releases
- release tag exists (`--version vX.Y.Z` when pinned)
- current binary location is writable

## For Maintainers

Release operation steps are documented in:

- `docs/RELEASING.md`
