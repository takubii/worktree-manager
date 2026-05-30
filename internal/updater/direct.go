package updater

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type commandContextFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type startCommandFunc func(cmd *exec.Cmd) error
type windowsReplaceFunc func(
	ctx context.Context,
	commandContext commandContextFunc,
	startCommand startCommandFunc,
	stagedBinaryPath string,
	targetBinaryPath string,
) error

// Direct updates wtm by downloading release assets from GitHub directly.
type Direct struct {
	goos            string
	goarch          string
	apiBaseURL      string
	httpClient      *http.Client
	executablePath  func() (string, error)
	commandContext  commandContextFunc
	startCommand    startCommandFunc
	replaceWindows  windowsReplaceFunc
	downloadToFile  func(ctx context.Context, client *http.Client, url string, path string) error
	nowUnixNanoFunc func() int64
}

// NewDirect returns the default direct-release updater implementation.
func NewDirect() Service {
	return &Direct{
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
		apiBaseURL: defaultAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		executablePath:  os.Executable,
		commandContext:  exec.CommandContext,
		startCommand:    (*exec.Cmd).Start,
		replaceWindows:  replaceBinaryWindows,
		downloadToFile:  downloadToFile,
		nowUnixNanoFunc: func() int64 { return time.Now().UnixNano() },
	}
}

// Update updates wtm to the latest release unless a specific version is provided.
func (d *Direct) Update(ctx context.Context, req Request) (Result, error) {
	if d.httpClient == nil {
		return Result{}, fmt.Errorf("updater HTTP client is not configured")
	}
	if d.downloadToFile == nil {
		return Result{}, fmt.Errorf("updater download function is not configured")
	}
	if d.executablePath == nil {
		return Result{}, fmt.Errorf("updater executable path resolver is not configured")
	}
	if d.commandContext == nil {
		return Result{}, fmt.Errorf("updater command runner is not configured")
	}
	if d.startCommand == nil {
		return Result{}, fmt.Errorf("updater async command starter is not configured")
	}
	if d.nowUnixNanoFunc == nil {
		d.nowUnixNanoFunc = func() int64 { return time.Now().UnixNano() }
	}

	version := strings.TrimSpace(req.Version)
	release, err := fetchGitHubRelease(ctx, d.httpClient, d.apiBaseURL, version)
	if err != nil {
		return Result{}, err
	}

	tag := strings.TrimSpace(release.TagName)
	if tag == "" {
		return Result{}, fmt.Errorf("release metadata did not include tag name. Verify release integrity and retry")
	}

	archiveName := releaseArchiveName(tag, d.goos, d.goarch)
	archiveURL, ok := release.findAssetURL(archiveName)
	if !ok {
		return Result{}, fmt.Errorf(
			"release asset %s was not found for %s/%s. Choose a supported platform and retry",
			archiveName,
			d.goos,
			d.goarch,
		)
	}

	checksumsURL, ok := release.findAssetURL(checksumsAsset)
	if !ok {
		return Result{}, fmt.Errorf("release asset %s was not found. Verify release assets and retry", checksumsAsset)
	}

	workDir, err := os.MkdirTemp("", "wtm-update-*")
	if err != nil {
		return Result{}, fmt.Errorf("failed to create updater working directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	archivePath := filepath.Join(workDir, archiveName)
	checksumsPath := filepath.Join(workDir, checksumsAsset)
	if err := d.downloadToFile(ctx, d.httpClient, archiveURL, archivePath); err != nil {
		return Result{}, err
	}
	if err := d.downloadToFile(ctx, d.httpClient, checksumsURL, checksumsPath); err != nil {
		return Result{}, err
	}
	if err := verifyArchiveChecksum(archivePath, checksumsPath, archiveName); err != nil {
		return Result{}, err
	}

	extractDir := filepath.Join(workDir, "extract")
	if err := extractReleaseArchive(archivePath, extractDir); err != nil {
		return Result{}, err
	}

	binaryPath, err := findExtractedBinary(extractDir, d.goos)
	if err != nil {
		return Result{}, err
	}

	targetPath, err := d.executablePath()
	if err != nil {
		return Result{}, fmt.Errorf("failed to resolve current wtm binary path: %w", err)
	}

	if d.goos == "windows" {
		if d.replaceWindows == nil {
			return Result{}, fmt.Errorf("updater windows replace function is not configured")
		}
		stagedPath := filepath.Join(os.TempDir(), fmt.Sprintf("wtm-update-%d-%d.exe", os.Getpid(), d.nowUnixNanoFunc()))
		if err := copyFile(binaryPath, stagedPath, 0o755); err != nil {
			return Result{}, fmt.Errorf("failed to stage updated Windows binary: %w", err)
		}

		if err := d.replaceWindows(ctx, d.commandContext, d.startCommand, stagedPath, targetPath); err != nil {
			return Result{}, err
		}
		return Result{Async: true}, nil
	}

	if err := replaceBinaryUnix(binaryPath, targetPath); err != nil {
		return Result{}, err
	}

	return Result{Async: false}, nil
}
