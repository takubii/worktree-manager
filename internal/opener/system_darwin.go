package opener

func systemOpenCommand(path string, window WindowMode) (string, []string, error) {
	if window == WindowReuse {
		return "open", []string{path}, nil
	}

	return "open", []string{"-n", path}, nil
}
