package doctor

import "fmt"

// FormatLine returns one doctor output line.
func FormatLine(r Result) string {
	return fmt.Sprintf("[%s] %s: %s | next: %s", r.Level, r.Name, r.Message, r.NextAction)
}
