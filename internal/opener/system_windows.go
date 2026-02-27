package opener

func systemOpenCommand(path string, window WindowMode) (string, []string, error) {
	argument := path
	if window == WindowNew {
		argument = "/n," + path
	}

	return "cmd", []string{"/c", "start", "", "explorer.exe", argument}, nil
}
