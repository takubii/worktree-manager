package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectUpdate_UnixLatestReplacesBinary(t *testing.T) {
	t.Parallel()

	archiveName := "git-worktree-opener_v9.9.9_linux_amd64.tar.gz"
	archive := buildTarGzArchive(t, "wto", []byte("new-binary"))
	archiveHash := sha256Hex(archive)
	checksums := archiveHash + "  " + archiveName + "\n"

	var latestCalled bool
	server := newReleaseServer(
		t,
		"/repos/takubii/git-worktree-opener/releases/latest",
		"v9.9.9",
		archiveName,
		archive,
		[]byte(checksums),
		func(path string) {
			if path == "/repos/takubii/git-worktree-opener/releases/latest" {
				latestCalled = true
			}
		},
	)
	defer server.Close()

	targetPath := filepath.Join(t.TempDir(), "wto")

	svc := &Direct{
		goos:           "linux",
		goarch:         "amd64",
		apiBaseURL:     server.URL + "/repos/takubii/git-worktree-opener",
		httpClient:     server.Client(),
		executablePath: func() (string, error) { return targetPath, nil },
		commandContext: exec.CommandContext,
		startCommand:   (*exec.Cmd).Start,
		downloadToFile: downloadToFile,
	}

	result, err := svc.Update(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	if result.Async {
		t.Fatal("expected synchronous update on non-Windows")
	}

	if !latestCalled {
		t.Fatal("expected latest release endpoint to be called")
	}

	updated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() returned error: %v", err)
	}
	if string(updated) != "new-binary" {
		t.Fatalf("unexpected updated binary content: %q", string(updated))
	}
}

func TestDirectUpdate_UsesTagEndpointWhenVersionSpecified(t *testing.T) {
	t.Parallel()

	archiveName := "git-worktree-opener_v1.2.3_windows_amd64.zip"
	archive := buildZipArchive(t, "wto.exe", []byte("new-binary"))
	archiveHash := sha256Hex(archive)
	checksums := archiveHash + "  " + archiveName + "\n"

	var requestedPath string
	server := newReleaseServer(
		t,
		"/repos/takubii/git-worktree-opener/releases/tags/v1.2.3",
		"v1.2.3",
		archiveName,
		archive,
		[]byte(checksums),
		func(path string) {
			if strings.Contains(path, "/releases/") {
				requestedPath = path
			}
		},
	)
	defer server.Close()

	var replaceCalls int

	svc := &Direct{
		goos:       "windows",
		goarch:     "amd64",
		apiBaseURL: server.URL + "/repos/takubii/git-worktree-opener",
		httpClient: server.Client(),
		executablePath: func() (string, error) {
			return filepath.Join(t.TempDir(), "wto.exe"), nil
		},
		commandContext: exec.CommandContext,
		startCommand:   (*exec.Cmd).Start,
		replaceWindows: func(
			_ context.Context,
			_ commandContextFunc,
			_ startCommandFunc,
			stagedBinaryPath string,
			targetBinaryPath string,
		) error {
			replaceCalls++
			if filepath.Ext(stagedBinaryPath) != ".exe" {
				t.Fatalf("unexpected staged binary path: %s", stagedBinaryPath)
			}
			if !strings.HasSuffix(strings.ToLower(targetBinaryPath), "wto.exe") {
				t.Fatalf("unexpected target binary path: %s", targetBinaryPath)
			}
			return nil
		},
		downloadToFile:  downloadToFile,
		nowUnixNanoFunc: func() int64 { return 12345 },
	}

	result, err := svc.Update(context.Background(), Request{Version: "v1.2.3"})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	if !result.Async {
		t.Fatal("expected asynchronous update on Windows")
	}
	if replaceCalls != 1 {
		t.Fatalf("expected replaceWindows call once, got %d", replaceCalls)
	}
	if requestedPath != "/repos/takubii/git-worktree-opener/releases/tags/v1.2.3" {
		t.Fatalf("unexpected requested path: %q", requestedPath)
	}
}

func TestDirectUpdate_ReturnsErrorOnChecksumMismatch(t *testing.T) {
	t.Parallel()

	archiveName := "git-worktree-opener_v9.9.9_linux_amd64.tar.gz"
	archive := buildTarGzArchive(t, "wto", []byte("new-binary"))
	checksums := "0000000000000000000000000000000000000000000000000000000000000000  " + archiveName + "\n"

	server := newReleaseServer(
		t,
		"/repos/takubii/git-worktree-opener/releases/latest",
		"v9.9.9",
		archiveName,
		archive,
		[]byte(checksums),
		nil,
	)
	defer server.Close()

	svc := &Direct{
		goos:       "linux",
		goarch:     "amd64",
		apiBaseURL: server.URL + "/repos/takubii/git-worktree-opener",
		httpClient: server.Client(),
		executablePath: func() (string, error) {
			return filepath.Join(t.TempDir(), "wto"), nil
		},
		commandContext: exec.CommandContext,
		startCommand:   (*exec.Cmd).Start,
		downloadToFile: downloadToFile,
	}

	_, err := svc.Update(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newReleaseServer(
	t *testing.T,
	releasePath string,
	tag string,
	archiveName string,
	archiveBody []byte,
	checksumsBody []byte,
	onRequest func(path string),
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(releasePath, func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r.URL.Path)
		}

		response := githubRelease{
			TagName: tag,
			Assets: []githubReleaseRef{
				{
					Name: archiveName,
					URL:  serverURLFromRequest(r) + "/assets/archive",
				},
				{
					Name: checksumsAsset,
					URL:  serverURLFromRequest(r) + "/assets/checksums",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode release response: %v", err)
		}
	})

	mux.HandleFunc("/assets/archive", func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r.URL.Path)
		}
		_, _ = w.Write(archiveBody)
	})
	mux.HandleFunc("/assets/checksums", func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r.URL.Path)
		}
		_, _ = w.Write(checksumsBody)
	})

	return httptest.NewServer(mux)
}

func serverURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func buildTarGzArchive(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() returned error: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() returned error: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() returned error: %v", err)
	}

	return buffer.Bytes()
}

func buildZipArchive(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}

	return buffer.Bytes()
}
