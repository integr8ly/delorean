package cmd

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/utils"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

func newTestCreateProdsecManifestCmd(olmType string, typeOfManifest string) *createProdsecManifestCmd {
	version, _ := utils.NewVersion("1.0.0", olmType)

	// Clone and initiate the mock integreatly repo from testdata/createRhoamManifest
	cloneService := &mockGitCloneService{cloneToTmpDirFunc: func(prefix string, url string, reference plumbing.ReferenceName) (s string, repository *git.Repository, err error) {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", nil, err
		}
		return initRepoFromTestDirRhoamManifest("test-create-manifest-", path.Join(currentDir, "testdata/createProdsecManifestTest"))
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

	return &createProdsecManifestCmd{
		version:         version,
		repoInfo:        &githubRepoInfo{owner: "test", repo: "test"},
		baseBranch:      "master",
		manifestScript:  "prodsec-manifest-generator.sh",
		typeOfManifest:  typeOfManifest,
		gitUser:         "testuser",
		gitPass:         "testpass",
		gitCloneService: cloneService,
		gitPushService:  pushService,
		githubPRService: prService,
	}
}

func verificationFunction(repoDir string, filePath string) error {
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

	content, err := ioutil.ReadFile(path.Join(repoDir, filePath))
	if err != nil {
		return err
	}
	if strings.Index(string(content), "dummy manifest dependency") < 0 {
		return fmt.Errorf("expected: %s, but got %s", "dummy manifest dependency", string(content))
	}
	return nil
}

func TestCreateRhoamManifestRelease(t *testing.T) {
	cases := []struct {
		description string
		cmd         func() *createProdsecManifestCmd
		expectError bool
		filePath    string
		verify      func(repoDir string, fileLocation string) error
	}{
		{
			description: "should successfully generate the correct manifest for RHMI",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("integreatly-operator", "production")
			},
			filePath:    "prodsec-manifests/rhmi-production-release-manifest.txt",
			expectError: false,
			verify:      verificationFunction,
		},
		{
			description: "should successfully generate the correct manifest for RHOAM",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("managed-api-service", "production")
			},
			filePath:    "prodsec-manifests/rhoam-production-release-manifest.txt",
			expectError: false,
			verify:      verificationFunction,
		},
		{
			description: "should successfully generate the correct master manifest for RHMI",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("integreatly-operator", "master")
			},
			filePath:    "prodsec-manifests/rhmi-master-manifest.txt",
			expectError: false,
			verify:      verificationFunction,
		},
		{
			description: "should successfully generate the correct master manifest for RHOAM",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("managed-api-service", "master")
			},
			filePath:    "prodsec-manifests/rhoam-master-manifest.txt",
			expectError: false,
			verify:      verificationFunction,
		},
		{
			description: "RHOAM compare manifest should return 1",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("managed-api-service", "compare")
			},
			expectError: true,
		},
		{
			description: "RHMI compare manifest should return 0",
			cmd: func() *createProdsecManifestCmd {
				return newTestCreateProdsecManifestCmd("integreatly-operator", "compare")
			},
			expectError: false,
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
			if err = c.verify(repo, c.filePath); err != nil {
				t.Fatalf("verification failed due to error: %v", err)
			}
		}
	}
}

func initRepoFromTestDirRhoamManifest(prefix string, testDir string) (string, *git.Repository, error) {
	dir, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		return "", nil, err
	}

	err = utils.CopyDirectory(testDir, dir)
	if err != nil {
		return "", nil, err
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return "", nil, err
	}

	tree, err := repo.Worktree()
	if err != nil {
		return "", nil, err
	}

	err = tree.AddGlob(".")
	if err != nil {
		return "", nil, err
	}

	_, err = tree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{},
	})
	if err != nil {
		return "", nil, err
	}

	if err := checkoutBranch(tree, false, true, "release-v1.0"); err != nil {
		return "", nil, err
	}

	if err := checkoutBranch(tree, false, true, "rhoam-release-v1.0"); err != nil {
		return "", nil, err
	}

	if err := checkoutBranch(tree, false, false, "master"); err != nil {
		return "", nil, err
	}

	return dir, repo, nil
}
