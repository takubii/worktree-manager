package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const userAgent = "worktree-manager-updater"

type githubRelease struct {
	TagName string             `json:"tag_name"`
	Assets  []githubReleaseRef `json:"assets"`
}

type githubReleaseRef struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func fetchGitHubRelease(ctx context.Context, client *http.Client, apiBaseURL string, version string) (githubRelease, error) {
	endpoint := strings.TrimRight(apiBaseURL, "/") + "/releases/latest"
	if strings.TrimSpace(version) != "" {
		endpoint = strings.TrimRight(apiBaseURL, "/") + "/releases/tags/" + strings.TrimSpace(version)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return githubRelease{}, fmt.Errorf("failed to create GitHub release request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("failed to resolve release metadata from GitHub. Check network connectivity and retry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf(
			"failed to resolve release metadata (HTTP %d). Verify tag/release existence and retry",
			resp.StatusCode,
		)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("failed to parse release metadata from GitHub: %w", err)
	}

	return release, nil
}

func (r githubRelease) findAssetURL(name string) (string, bool) {
	for _, asset := range r.Assets {
		if asset.Name == name && strings.TrimSpace(asset.URL) != "" {
			return asset.URL, true
		}
	}
	return "", false
}

func downloadToFile(ctx context.Context, client *http.Client, url string, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s. Check network connectivity and retry: %w", filepath.Base(path), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"failed to download %s (HTTP %d). Verify release assets and retry",
			filepath.Base(path),
			resp.StatusCode,
		)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to prepare download directory for %s: %w", path, err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create download file %s: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write downloaded file %s: %w", path, err)
	}

	return nil
}
