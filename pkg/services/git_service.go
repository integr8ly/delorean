package services

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitRepositoryService interface {
	Head() (*plumbing.Reference, error)
	Worktree() (*git.Worktree, error)
	Push(o *git.PushOptions) error
}
