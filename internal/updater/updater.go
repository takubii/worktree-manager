package updater

import (
	"context"
	"io"
)

const (
	repoOwner         = "takubii"
	repoName          = "git-worktree-opener"
	defaultAPIBaseURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName
	releaseAssetBase  = repoName
	checksumsAsset    = "checksums.txt"
	binaryBaseName    = "wto"
)

// Request defines updater inputs.
type Request struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

// Result describes updater execution mode.
type Result struct {
	Async bool
}

// Service updates the current wto installation.
type Service interface {
	Update(ctx context.Context, req Request) (Result, error)
}
