package doctor

import "context"

// Level is the severity level of a doctor check result.
type Level string

const (
	LevelOK       Level = "OK"
	LevelWarn     Level = "WARN"
	LevelCritical Level = "CRIT"
)

// Result is a single doctor check outcome.
type Result struct {
	Name       string
	Level      Level
	Message    string
	NextAction string
}

// Report is the full doctor result set.
type Report struct {
	Results     []Result
	HasCritical bool
}

// Service runs environment diagnostics for wto.
type Service interface {
	Run(ctx context.Context) Report
}
