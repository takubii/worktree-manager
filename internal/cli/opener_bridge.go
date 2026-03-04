package cli

import (
	"context"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/opener"
)

type openerWithResult interface {
	OpenWithResult(ctx context.Context, req opener.OpenRequest) (opener.OpenResult, error)
}

func openPathWithResult(
	ctx context.Context,
	openr opener.Opener,
	kind string,
	path string,
	window opener.WindowMode,
	terminalProvider string,
	tmuxMode opener.TmuxMode,
) (opener.OpenResult, error) {
	if withResult, ok := openr.(openerWithResult); ok {
		return withResult.OpenWithResult(ctx, opener.OpenRequest{
			Kind:             kind,
			Path:             path,
			Window:           window,
			TerminalProvider: terminalProvider,
			TmuxMode:         tmuxMode,
		})
	}

	if err := openr.Open(ctx, kind, path, window); err != nil {
		return opener.OpenResult{}, err
	}

	return opener.OpenResult{
		Provider: strings.TrimSpace(kind),
	}, nil
}
