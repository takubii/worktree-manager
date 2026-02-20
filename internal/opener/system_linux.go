package opener

func systemOpenCommand(path string, _ WindowMode) (string, []string, error) {
	return "xdg-open", []string{path}, nil
}
