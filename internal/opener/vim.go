package opener

import (
	"context"
	"fmt"
)

func (o *defaultOpener) openVim(ctx context.Context, path string, _ WindowMode) error {
	if _, err := o.lookPath("nvr"); err == nil {
		return o.run(ctx, "nvr", "--remote-tab-cwd", path)
	}
	if _, err := o.lookPath("nvim"); err == nil {
		return o.run(ctx, "nvim", path)
	}
	if _, err := o.lookPath("vim"); err == nil {
		return o.run(ctx, "vim", path)
	}

	return fmt.Errorf("vim opener is unavailable. Install one of: nvr, nvim, or vim")
}
