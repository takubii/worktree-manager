package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func releaseArchiveName(tag string, goos string, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("%s_%s_%s_%s%s", releaseAssetBase, tag, goos, goarch, ext)
}

func extractReleaseArchive(archivePath string, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create extraction directory %s: %w", destDir, err)
	}

	switch {
	case strings.HasSuffix(strings.ToLower(archivePath), ".zip"):
		return extractZipArchive(archivePath, destDir)
	case strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz"):
		return extractTarGzArchive(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format for %s", filepath.Base(archivePath))
	}
}

func findExtractedBinary(root string, goos string) (string, error) {
	binaryName := binaryBaseName
	if goos == "windows" {
		binaryName += ".exe"
	}

	var found string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), binaryName) {
			found = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to inspect extracted archive files: %w", err)
	}

	if strings.TrimSpace(found) == "" {
		return "", fmt.Errorf("archive does not contain %s. Verify release assets and retry", binaryName)
	}

	return found, nil
}

func extractZipArchive(archivePath string, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive %s: %w", archivePath, err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath, err := safeArchiveTargetPath(destDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create extracted directory %s: %w", targetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create extracted parent directory for %s: %w", targetPath, err)
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open archived file %s: %w", file.Name, err)
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			src.Close()
			return fmt.Errorf("failed to create extracted file %s: %w", targetPath, err)
		}

		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}

		if err := dst.Close(); err != nil {
			src.Close()
			return fmt.Errorf("failed to finalize extracted file %s: %w", targetPath, err)
		}
		if err := src.Close(); err != nil {
			return fmt.Errorf("failed to close archived file %s: %w", file.Name, err)
		}
	}

	return nil
}

func extractTarGzArchive(archivePath string, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to read gzip stream in %s: %w", archivePath, err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry in %s: %w", archivePath, err)
		}

		targetPath, err := safeArchiveTargetPath(destDir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create extracted directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create extracted parent directory for %s: %w", targetPath, err)
			}

			mode := os.FileMode(header.Mode)
			if mode == 0 {
				mode = 0o644
			}
			dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("failed to create extracted file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(dst, reader); err != nil {
				dst.Close()
				return fmt.Errorf("failed to extract file %s: %w", header.Name, err)
			}
			if err := dst.Close(); err != nil {
				return fmt.Errorf("failed to finalize extracted file %s: %w", targetPath, err)
			}
		}
	}

	return nil
}

func safeArchiveTargetPath(baseDir string, archiveName string) (string, error) {
	target := filepath.Clean(filepath.Join(baseDir, archiveName))
	base := filepath.Clean(baseDir)
	if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
		return "", fmt.Errorf("archive contains unsafe path %q. Verify release assets and retry", archiveName)
	}
	return target, nil
}
