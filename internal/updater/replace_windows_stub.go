//go:build !windows

package updater

import (
	"context"
	"fmt"
)

func replaceBinaryWindows(
	_ context.Context,
	_ commandContextFunc,
	_ startCommandFunc,
	_ string,
	_ string,
) error {
	return fmt.Errorf("windows binary replacement is not supported on this platform")
}
