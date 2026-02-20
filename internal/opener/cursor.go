package opener

import "context"

func (o *defaultOpener) openCursor(ctx context.Context, path string, window WindowMode) error {
	if _, err := o.lookPath("cursor"); err == nil {
		if window == WindowReuse {
			return o.run(ctx, "cursor", "--reuse-window", path)
		}
		return o.run(ctx, "cursor", "--new-window", path)
	}

	if o.goos == "darwin" {
		if window == WindowReuse {
			return o.run(ctx, "open", "-a", "Cursor", path)
		}
		return o.run(ctx, "open", "-n", "-a", "Cursor", path)
	}

	return o.openSystem(ctx, path, window)
}
