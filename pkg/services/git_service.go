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

type GitRemoteService interface {
	Create(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error)
	CreateAndPull(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error)
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
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
	})
	if err != nil && err.Error() != "already up-to-date" {
		return "", nil, err
	}

	return dir, repo, nil
}

type DefaultGitPushService struct{}
type DefaultGitRemoteService struct{}

func (s *DefaultGitPushService) Push(gitRepo *git.Repository, opts *git.PushOptions) error {
	return gitRepo.Push(opts)
}

func (s *DefaultGitRemoteService) Create(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error) {
	return gitRepo.CreateRemote(remoteConfig)
}

func (s *DefaultGitRemoteService) CreateAndPull(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error) {

	remote, err := gitRepo.CreateRemote(remoteConfig)
	if err != nil {
		return nil, err
	}

	worktree, err := gitRepo.Worktree()
	if err != nil {
		return nil, err
	}

	//Pull remote into current branch
	err = worktree.Pull(&git.PullOptions{RemoteName: remoteConfig.Name})
	if err != nil {
		return nil, err
	}

	return remote, err
}
