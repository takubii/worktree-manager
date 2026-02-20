package opener

import (
	"fmt"
	"strings"
)

func systemOpenCommand(path string, window WindowMode) (string, []string, error) {
	path = quoteForPowerShellSingle(path)

	argument := path
	if window == WindowNew {
		argument = "/n," + argument
	}

	command := fmt.Sprintf("Start-Process -FilePath explorer.exe -ArgumentList '%s'", argument)
	return "powershell", []string{"-NoProfile", "-Command", command}, nil
}

func quoteForPowerShellSingle(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}
