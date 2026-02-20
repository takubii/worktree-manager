package opener

import "context"

func (o *defaultOpener) openVSCode(ctx context.Context, path string, window WindowMode) error {
	if _, err := o.lookPath("code"); err == nil {
		if window == WindowReuse {
			return o.run(ctx, "code", "--reuse-window", path)
		}
		return o.run(ctx, "code", "--new-window", path)
	}
	return o.openSystem(ctx, path, window)
}
