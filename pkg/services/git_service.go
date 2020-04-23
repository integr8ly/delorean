package services

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"io/ioutil"
	"os"
)

type GitCloneService interface {
	CloneToTmpDir(prefix string, url string, reference plumbing.ReferenceName) (string, *git.Repository, error)
}

type GitPushService interface {
	Push(gitRepo *git.Repository, opts *git.PushOptions) error
}

type DefaultGitCloneService struct{}

func (s *DefaultGitCloneService) CloneToTmpDir(prefix string, url string, reference plumbing.ReferenceName) (string, *git.Repository, error) {
	dir, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		return "", nil, err
	}

	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: reference,
		Progress:      os.Stdout,
	})
	if err != nil {
		return "", nil, err
	}

	err = repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	})
	if err != nil {
		return "", nil, err
	}

	return dir, repo, nil
}

type DefaultGitPushService struct{}

func (s *DefaultGitPushService) Push(gitRepo *git.Repository, opts *git.PushOptions) error {
	return gitRepo.Push(opts)
}
