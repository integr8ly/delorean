package cmd

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

type mockGitCloneService struct {
	cloneToTmpDirFunc func(prefix string, url string, reference plumbing.ReferenceName) (string, *git.Repository, error)
}

func (m mockGitCloneService) CloneToTmpDir(prefix string, url string, reference plumbing.ReferenceName) (string, *git.Repository, error) {
	if m.cloneToTmpDirFunc != nil {
		return m.cloneToTmpDirFunc(prefix, url, reference)
	}
	panic("implement me")
}

type mockGitPushService struct {
	pushFunc func(gitRepo *git.Repository, opts *git.PushOptions) error
}

func (m mockGitPushService) Push(gitRepo *git.Repository, opts *git.PushOptions) error {
	if m.pushFunc != nil {
		return m.pushFunc(gitRepo, opts)
	}
	panic("implement me")
}

func newTestCreateReleaseCmd(serviceAffecting bool) *createReleaseCmd {
	version, _ := utils.NewRHMIVersion("2.0.0-rc1")
	cloneService := &mockGitCloneService{cloneToTmpDirFunc: func(prefix string, url string, reference plumbing.ReferenceName) (s string, repository *git.Repository, err error) {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", nil, err
		}
		return initRepoFromTestDir("test-create-release-", path.Join(currentDir, "testdata/createReleaseTest"))
	}}
	pushService := &mockGitPushService{pushFunc: func(gitRepo *git.Repository, opts *git.PushOptions) error {
		return nil
	}}
	prService := mockPullRequestsService{
		ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
			return []*github.PullRequest{}, nil, nil
		},
		CreateFunc: func(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (request *github.PullRequest, response *github.Response, err error) {
			url := "http://testurl"
			return &github.PullRequest{
				HTMLURL: &url,
			}, nil, nil
		},
	}
	return &createReleaseCmd{
		version:          version,
		repoInfo:         &githubRepoInfo{owner: "test", repo: "test"},
		baseBranch:       "master",
		releaseScript:    "release.sh",
		gitUser:          "testuser",
		gitPass:          "testpass",
		gitCloneService:  cloneService,
		gitPushService:   pushService,
		githubPRService:  prService,
		serviceAffecting: serviceAffecting,
	}
}

func TestCreateRelease(t *testing.T) {
	cases := []struct {
		description string
		cmd         func() *createReleaseCmd
		expectError bool
		verify      func(repoDir string) error
	}{
		{
			description: "should finish successfully when serviceAffecting is true",
			cmd: func() *createReleaseCmd {
				return newTestCreateReleaseCmd(true)
			},
			expectError: false,
			verify: func(repoDir string) error {
				repo, err := git.PlainOpen(repoDir)
				if err != nil {
					return err
				}
				repoTree, err := repo.Worktree()
				if err != nil {
					return err
				}
				status, err := repoTree.Status()
				if err != nil {
					return err
				}
				if len(status) > 0 {
					return fmt.Errorf("there are uncommited changes in the repo: %v", status)
				}
				content, err := ioutil.ReadFile(path.Join(repoDir, "VERSION.txt"))
				if err != nil {
					return err
				}
				if strings.Index(string(content), "2.0.0-rc1") < 0 {
					return fmt.Errorf("expected: %s, but got %s", "2.0.0-rc1", string(content))
				}
				if strings.Index(string(content), "ServiceAffecting=false") >= 0 {
					return fmt.Errorf("unexpected output: ServiceAffecting=false in output: %s", string(content))
				}
				return nil
			},
		},
		{
			description: "should finish successfully when serviceAffecting is false",
			cmd: func() *createReleaseCmd {
				return newTestCreateReleaseCmd(false)
			},
			expectError: false,
			verify: func(repoDir string) error {
				repo, err := git.PlainOpen(repoDir)
				if err != nil {
					return err
				}
				repoTree, err := repo.Worktree()
				if err != nil {
					return err
				}
				status, err := repoTree.Status()
				if err != nil {
					return err
				}
				if len(status) > 0 {
					return fmt.Errorf("there are uncommited changes in the repo: %v", status)
				}
				content, err := ioutil.ReadFile(path.Join(repoDir, "VERSION.txt"))
				if err != nil {
					return err
				}
				if strings.Index(string(content), "2.0.0-rc1") < 0 {
					return fmt.Errorf("expected: %s, but got %s", "2.0.0-rc1", string(content))
				}
				if strings.Index(string(content), "ServiceAffecting=false") < 0 {
					return fmt.Errorf("expecting ServiceAffecting=false in output: %s", string(content))
				}
				return nil
			},
		},
	}

	for _, c := range cases {
		cmd := c.cmd()
		repo, err := cmd.run(context.TODO())
		if repo != "" {
			defer os.RemoveAll(repo)
		}
		if c.expectError && err == nil {
			t.Fatalf("error expected but it is nil")
		} else if !c.expectError && err != nil {
			t.Fatalf("error should be nil but got %v", err)
		}
		if c.verify != nil {
			if err = c.verify(repo); err != nil {
				t.Fatalf("verification failed due to error: %v", err)
			}
		}
	}
}
