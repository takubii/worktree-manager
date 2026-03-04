package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type rawConfig struct {
	Remote              *string  `json:"remote"`
	BaseBranch          *string  `json:"baseBranch"`
	WorktreeDirTemplate *string  `json:"worktreeDirTemplate"`
	New                 *rawNew  `json:"new"`
	Open                *rawOpen `json:"open"`
	Tmux                *rawTmux `json:"tmux"`
	RM                  *rawRM   `json:"rm"`
}

type rawNew struct {
	Fetch *bool `json:"fetch"`
	Prune *bool `json:"prune"`
}

type rawOpen struct {
	Default          *string `json:"default"`
	Window           *string `json:"window"`
	Prune            *bool   `json:"prune"`
	TerminalProvider *string `json:"terminalProvider"`
}

type rawTmux struct {
	Mode *string `json:"mode"`
}

type rawRM struct {
	DeleteBranch *string `json:"deleteBranch"`
}

func loadOverrideFromFile(path string, readFile func(string) ([]byte, error)) (configOverride, error) {
	if readFile == nil {
		return configOverride{}, fmt.Errorf("readFile function is not configured")
	}

	body, err := readFile(path)
	if err != nil {
		return configOverride{}, err
	}

	raw, err := decodeRawConfig(body)
	if err != nil {
		return configOverride{}, err
	}

	override, err := normalizeOverride(raw)
	if err != nil {
		return configOverride{}, err
	}

	return override, nil
}

func decodeRawConfig(body []byte) (rawConfig, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return rawConfig{}, fmt.Errorf("config file is empty")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var raw rawConfig
	if err := decoder.Decode(&raw); err != nil {
		return rawConfig{}, fmt.Errorf("invalid JSON: %w", err)
	}

	var trailing struct{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return rawConfig{}, fmt.Errorf("invalid JSON: multiple top-level values are not allowed")
		}
		return rawConfig{}, fmt.Errorf("invalid JSON: %w", err)
	}

	return raw, nil
}

func marshalConfig(cfg Config) ([]byte, error) {
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return append(body, '\n'), nil
}
