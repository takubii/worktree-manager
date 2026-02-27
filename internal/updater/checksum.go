package updater

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func verifyArchiveChecksum(archivePath string, checksumsPath string, archiveName string) error {
	expected, err := findChecksumValue(checksumsPath, archiveName)
	if err != nil {
		return err
	}

	actual, err := sha256File(archivePath)
	if err != nil {
		return fmt.Errorf("failed to compute SHA256 for %s: %w", archiveName, err)
	}

	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf(
			"checksum mismatch for %s. Re-download the release or verify release integrity and retry",
			archiveName,
		)
	}

	return nil
}

func findChecksumValue(checksumsPath string, assetName string) (string, error) {
	file, err := os.Open(checksumsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read checksums file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		hash := strings.TrimSpace(fields[0])
		name := strings.TrimPrefix(strings.TrimSpace(fields[len(fields)-1]), "*")
		if name != assetName {
			continue
		}

		return strings.ToLower(hash), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed while scanning checksums file: %w", err)
	}

	return "", fmt.Errorf("checksum for %s was not found. Verify release assets and retry", assetName)
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
