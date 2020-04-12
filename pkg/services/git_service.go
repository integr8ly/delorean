package services

import (
	"github.com/go-git/go-git/v5"
)

type GitService interface {
	PlainClone(path string, isBare bool, o *git.CloneOptions) (GitRepositoryService, error)
}
