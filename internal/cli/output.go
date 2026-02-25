package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type outputMode string

const (
	outputModeNone outputMode = "none"
	outputModePath outputMode = "path"
	outputModeJSON outputMode = "json"
)

const supportedOutputModesText = "none|path|json"

type commandOutput struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Created bool   `json:"created"`
	Opened  bool   `json:"opened"`
}

func parseOutputMode(raw string) (outputMode, error) {
	mode := outputMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case outputModeNone, outputModePath, outputModeJSON:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --output value %q. Use one of: %s", raw, supportedOutputModesText)
	}
}

func writeCommandOutput(w io.Writer, mode outputMode, payload commandOutput) error {
	switch mode {
	case outputModeNone:
		return nil
	case outputModePath:
		if _, err := fmt.Fprintln(w, payload.Path); err != nil {
			return fmt.Errorf("failed to write path output: %w", err)
		}
		return nil
	case outputModeJSON:
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(payload); err != nil {
			return fmt.Errorf("failed to write json output: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported output mode: %s", mode)
	}
}
