//go:build !windows && !darwin && !linux

package opener

import "fmt"

func systemOpenCommand(_ string, _ WindowMode) (string, []string, error) {
	return "", nil, fmt.Errorf("system opener is not supported on this platform")
}
