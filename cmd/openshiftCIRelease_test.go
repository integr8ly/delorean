package cmd

import (
	"context"
	"fmt"
	"github.com/integr8ly/delorean/pkg/types"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/config"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
)

type mockGitRemoteService struct {
	createFunc        func(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error)
	CreateAndPullFunc func(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error)
}

func (m mockGitRemoteService) Create(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error) {
	if m.createFunc != nil {
		return m.createFunc(gitRepo, remoteConfig)
	}
	panic("implement me")
}

func (m mockGitRemoteService) CreateAndPull(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error) {
	if m.CreateAndPullFunc != nil {
		return m.CreateAndPullFunc(gitRepo, remoteConfig)
	}
	panic("implement me")
}

func verifyRepo(repoDir, expectedBranch string) (*git.Repository, error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return nil, err
	}
	repoTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	status, err := repoTree.Status()
	if err != nil {
		return nil, err
	}
	if len(status) > 0 {
		return nil, fmt.Errorf("there are uncommited changes in the repo: %v", status)
	}

	h, err := repo.Head()
	if err != nil {
		return nil, err
	}
	actualBranch := h.Name().Short()
	if actualBranch != expectedBranch {
		return nil, fmt.Errorf("current branch name is incorrect, want: %s, got: %s", expectedBranch, actualBranch)
	}

	return repo, nil
}

func Test_updateCIOperatorConfig(t *testing.T) {

	validReleaseDir := "./testdata/release"
	validReleaseDirTmp, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(validReleaseDirTmp)

	err = utils.CopyDirectory(validReleaseDir, validReleaseDirTmp)
	if err != nil {
		t.Fatalf("failed to copy the directory %s to %s: %s", validReleaseDir, validReleaseDirTmp, err)
	}

	type args struct {
		repoDir string
		version string
		olmType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		verify  func(releaseDir string) error
	}{
		{
			name: "valid directory for integreatly-operator olmtype",
			args: args{
				repoDir: validReleaseDirTmp,
				olmType: types.OlmTypeRhmi,
				version: "2.20.0-rc1",
			},
			verify: func(repoDir string) error {
				content, err := ioutil.ReadFile(path.Join(repoDir, "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-release-v2.20.yaml"))
				if err != nil {
					return err
				}
				promotionStr := "promotion:\n  name: \"2.20\""
				if strings.Index(string(content), promotionStr) < 0 {
					return fmt.Errorf("missing content: %s", promotionStr)
				}

				branchStr := "zz_generated_metadata:\n  branch: release-v2.20"
				if strings.Index(string(content), branchStr) < 0 {
					return fmt.Errorf("missing content: %s", branchStr)
				}

				return nil
			},
		},
		{
			name: "valid directory for managed-api-service olmtype",
			args: args{
				repoDir: validReleaseDirTmp,
				olmType: types.OlmTypeRhoam,
				version: "1.20.0-rc1",
			},
			verify: func(repoDir string) error {
				content, err := ioutil.ReadFile(path.Join(repoDir, "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-rhoam-release-v1.20.yaml"))
				if err != nil {
					return err
				}
				promotionStr := "promotion:\n  name: \"1.20\""
				if strings.Index(string(content), promotionStr) < 0 {
					return fmt.Errorf("missing content: %s", promotionStr)
				}

				branchStr := "zz_generated_metadata:\n  branch: rhoam-release-v1.20"
				if strings.Index(string(content), branchStr) < 0 {
					return fmt.Errorf("missing content: %s", branchStr)
				}

				return nil
			},
		},
		{
			name: "invalid directory",
			args: args{
				repoDir: "./testdata",
				olmType: types.OlmTypeRhmi,
				version: "1.2.3",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := utils.NewVersion(tt.args.version, tt.args.olmType)
			if err != nil {
				t.Errorf("updateCIOperatorConfig() utls.NewVersion error = %v", err)
			}
			c := &openshiftCIReleaseCmd{
				version:       version,
				intlyRepoInfo: &githubRepoInfo{owner: "test", repo: "test"},
				gitCloneService: &mockGitCloneService{cloneToTmpDirFunc: func(prefix string, url string, reference plumbing.ReferenceName) (s string, repository *git.Repository, err error) {
					currentDir, err := os.Getwd()
					if err != nil {
						return "", nil, err
					}
					return initRepoFromTestDir("test-openshift-ci-release-", path.Join(currentDir, "testdata/createReleaseTest"))
				}},
				gitPushService: &mockGitPushService{pushFunc: func(gitRepo *git.Repository, opts *git.PushOptions) error {
					return nil
				}},
			}
			if err := c.updateCIOperatorConfig(tt.args.repoDir); (err != nil) != tt.wantErr {
				t.Errorf("updateCIOperatorConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verify != nil {
				if err = tt.verify(tt.args.repoDir); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}
		})
	}
}

func Test_updateImageMirroringConfig(t *testing.T) {

	validReleaseDir := "./testdata/release"
	validReleaseDirTmp, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(validReleaseDirTmp)

	err = utils.CopyDirectory(validReleaseDir, validReleaseDirTmp)
	if err != nil {
		t.Fatalf("failed to copy the directory %s to %s: %s", validReleaseDir, validReleaseDirTmp, err)
	}

	type args struct {
		repoDir string
		olmType string
		version string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		verify  func(releaseDir string) error
	}{
		{
			name: "valid directory and mappings for integreatly-operator",
			args: args{
				repoDir: validReleaseDirTmp,
				olmType: types.OlmTypeRhmi,
				version: "2.0.0",
			},
			verify: func(repoDir string) error {
				content, err := ioutil.ReadFile(path.Join(repoDir, "core-services/image-mirroring/integr8ly/mapping_integr8ly_operator_2_0"))
				if err != nil {
					return err
				}

				expectedMappings := "" +
					"registry.ci.openshift.org/integr8ly/2.0:integreatly-operator " +
					"quay.io/integreatly/integreatly-operator:2.0\n" +
					"registry.ci.openshift.org/integr8ly/2.0:integreatly-operator-test-harness " +
					"quay.io/integreatly/integreatly-operator-test-harness:2.0"
				if strings.Index(string(content), expectedMappings) < 0 {
					return fmt.Errorf("expected: %s, got: %s", expectedMappings, string(content))
				}

				return nil
			},
		},
		{
			name: "valid directory and mappings for managed-api",
			args: args{
				repoDir: validReleaseDirTmp,
				olmType: types.OlmTypeRhoam,
				version: "1.1.0",
			},
			verify: func(repoDir string) error {
				content, err := ioutil.ReadFile(path.Join(repoDir, "core-services/image-mirroring/integr8ly/mapping_integr8ly_operator_1_1"))
				if err != nil {
					return err
				}

				expectedMappings := "" +
					"registry.ci.openshift.org/integr8ly/1.1:integreatly-operator " +
					"quay.io/integreatly/managed-api-service:1.1\n" +
					"registry.ci.openshift.org/integr8ly/1.1:integreatly-operator-test-harness " +
					"quay.io/integreatly/integreatly-operator-test-harness:1.1"
				if strings.Index(string(content), expectedMappings) < 0 {
					return fmt.Errorf("expected: %s, got: %s", expectedMappings, string(content))
				}

				return nil
			},
		},
		{
			name: "invalid directory",
			args: args{
				repoDir: "./testdata",
				olmType: types.OlmTypeRhmi,
				version: "2.2.2",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, _ := utils.NewVersion(tt.args.version, tt.args.olmType)
			if err != nil {
				t.Errorf("updateImageMirroringConfig() utls.NewVersion error = %v", err)
			}

			if err := updateImageMirroringConfig(tt.args.repoDir, version); (err != nil) != tt.wantErr {
				t.Errorf("updateImageMirroringConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verify != nil {
				if err = tt.verify(tt.args.repoDir); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}
		})
	}
}

func Test_openshiftCIReleaseCmd_createPRIfNotExists(t *testing.T) {
	type fields struct {
		releaseRepoInfoUpstream *githubRepoInfo
		githubPRService         services.PullRequestsService
	}
	type args struct {
		ctx   context.Context
		newPR *github.NewPullRequest
	}
	head := "head"
	base := "base"
	prURL1 := "http://testurl1"
	prNumber1 := 1
	prURL2 := "http://testurl2"
	prNumber2 := 2
	testPR1 := &github.PullRequest{HTMLURL: &prURL1, Number: &prNumber1}
	testPR2 := &github.PullRequest{HTMLURL: &prURL2, Number: &prNumber2}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *github.PullRequest
		wantErr bool
	}{
		{
			name: "test find existing PR",
			fields: fields{
				releaseRepoInfoUpstream: &githubRepoInfo{owner: "test", repo: "test"},
				githubPRService: mockPullRequestsService{
					ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
						return []*github.PullRequest{testPR1}, nil, nil
					},
					GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
						return testPR1, nil, nil
					},
					CreateFunc: func(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (request *github.PullRequest, response *github.Response, err error) {
						return testPR2, nil, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				newPR: &github.NewPullRequest{
					Head: &head,
					Base: &base,
				},
			},
			want: testPR1,
		},
		{
			name: "test create new PR",
			fields: fields{
				releaseRepoInfoUpstream: &githubRepoInfo{owner: "test", repo: "test"},
				githubPRService: mockPullRequestsService{
					ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
						return []*github.PullRequest{}, nil, nil
					},
					CreateFunc: func(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (request *github.PullRequest, response *github.Response, err error) {
						return testPR2, nil, nil
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				newPR: &github.NewPullRequest{
					Head: &head,
					Base: &base,
				},
			},
			want: testPR2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &openshiftCIReleaseCmd{
				releaseRepoInfoUpstream: tt.fields.releaseRepoInfoUpstream,
				githubPRService:         tt.fields.githubPRService,
			}
			got, err := c.createPRIfNotExists(tt.args.ctx, tt.args.newPR)
			if (err != nil) != tt.wantErr {
				t.Errorf("openshiftCIReleaseCmd.createPRIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("openshiftCIReleaseCmd.createPRIfNotExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_openshiftCIReleaseCmd_DoIntlyOperatorUpdate(t *testing.T) {

	tests := []struct {
		name       string
		version    string
		olmType    string
		wantBranch string
		wantErr    bool
		verify     func(repoDir, expectedBranch string) (*git.Repository, error)
	}{
		{
			name:       "test success 2.0.0-rc1",
			version:    "2.0.0-rc1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.0.0",
			version:    "2.0.0",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.0.1-rc1",
			version:    "2.0.1-rc1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.1.0-er1",
			version:    "2.1.0-er1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0-rc1",
			version:    "1.1.0",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0-er1",
			version:    "1.1.0",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0",
			version:    "1.1.0",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, _ := utils.NewVersion(tt.version, tt.olmType)
			c := &openshiftCIReleaseCmd{
				version:       version,
				intlyRepoInfo: &githubRepoInfo{owner: "test", repo: "test"},
				gitCloneService: &mockGitCloneService{cloneToTmpDirFunc: func(prefix string, url string, reference plumbing.ReferenceName) (s string, repository *git.Repository, err error) {
					currentDir, err := os.Getwd()
					if err != nil {
						return "", nil, err
					}
					return initRepoFromTestDir("test-openshift-ci-release-", path.Join(currentDir, "testdata/createReleaseTest"))
				}},
				gitPushService: &mockGitPushService{pushFunc: func(gitRepo *git.Repository, opts *git.PushOptions) error {
					return nil
				}},
			}
			repoDir, err := c.DoIntlyOperatorUpdate()
			if repoDir != "" {
				defer os.RemoveAll(repoDir)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("openshiftCIReleaseCmd.DoIntlyOperatorUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if repoDir == "" {
				t.Errorf("openshiftCIReleaseCmd.DoIntlyOperatorUpdate() = %v", repoDir)
			}
			if tt.verify != nil {
				if _, err = tt.verify(repoDir, tt.wantBranch); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}
		})
	}
}

func Test_openshiftCIReleaseCmd_DoOpenShiftReleaseUpdate(t *testing.T) {

	prURL := "http://testurl1"
	prNumber := 1

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name       string
		version    string
		olmType    string
		args       args
		wantBranch string
		wantErr    bool
		verify     func(repoDir, expectedBranch string) (*git.Repository, error)
	}{
		{
			name:       "test success 2.0.0-rc1",
			version:    "2.0.0-rc1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.0.0",
			version:    "2.0.0",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.0.1-rc1",
			version:    "2.0.1-rc1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.0",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 2.1.0-er1",
			version:    "2.1.0-er1",
			olmType:    types.OlmTypeRhmi,
			wantBranch: "release-v2.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0-rc1",
			version:    "1.1.0-rc1",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0-er1",
			version:    "1.1.0-er1",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
		{
			name:       "test success 1.1.0",
			version:    "1.1.0",
			olmType:    types.OlmTypeRhoam,
			wantBranch: "rhoam-release-v1.1",
			verify: func(repoDir, expectedBranch string) (*git.Repository, error) {
				return verifyRepo(repoDir, expectedBranch)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			version, _ := utils.NewVersion(tt.version, tt.olmType)

			c := &openshiftCIReleaseCmd{
				version:                 version,
				releaseRepoInfoOrigin:   &githubRepoInfo{owner: "test", repo: "test"},
				releaseRepoInfoUpstream: &githubRepoInfo{owner: "test2", repo: "test"},
				gitCloneService: &mockGitCloneService{cloneToTmpDirFunc: func(prefix string, url string, reference plumbing.ReferenceName) (s string, repository *git.Repository, err error) {
					currentDir, err := os.Getwd()
					if err != nil {
						return "", nil, err
					}
					return initRepoFromTestDir("test-openshift-ci-release-", path.Join(currentDir, "testdata/release"))
				}},
				gitPushService: &mockGitPushService{pushFunc: func(gitRepo *git.Repository, opts *git.PushOptions) error {
					return nil
				}},
				githubPRService: mockPullRequestsService{
					ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
						return []*github.PullRequest{}, nil, nil
					},
					CreateFunc: func(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (request *github.PullRequest, response *github.Response, err error) {
						return &github.PullRequest{HTMLURL: &prURL, Number: &prNumber}, nil, nil
					},
				},
				gitRemoteService: mockGitRemoteService{
					CreateAndPullFunc: func(gitRepo *git.Repository, remoteConfig *config.RemoteConfig) (*git.Remote, error) {
						return nil, nil
					},
				},
			}
			repoDir, err := c.DoOpenShiftReleaseUpdate(tt.args.ctx)
			if repoDir != "" {
				defer os.RemoveAll(repoDir)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("openshiftCIReleaseCmd.DoOpenShiftReleaseUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if repoDir == "" {
				t.Errorf("openshiftCIReleaseCmd.DoOpenShiftReleaseUpdate() = %v", repoDir)
			}
			if tt.verify != nil {
				if _, err = tt.verify(repoDir, tt.wantBranch); err != nil {
					t.Fatalf("verification failed due to error: %v", err)
				}
			}
		})
	}
}
