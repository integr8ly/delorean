package cmd

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type createReleaseCmdFlags struct {
	baseBranch    string
	releaseScript string
}

type createReleaseCmd struct {
	version         *utils.RHMIVersion
	repoInfo        *githubRepoInfo
	baseBranch      plumbing.ReferenceName
	githubPRService services.PullRequestsService
	releaseScript   string
	gitUser         string
	gitPass         string
	gitCloneService services.GitCloneService
	gitPushService  services.GitPushService
}

func init() {
	f := &createReleaseCmdFlags{}

	cmd := &cobra.Command{
		Use:   "create-release",
		Short: "Create a release",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newCreateReleaseCmd(f)
			if err != nil {
				handleError(err)
			}
			var repoDir string
			if repoDir, err = c.run(cmd.Context()); err != nil {
				handleError(err)
			}
			if repoDir != "" {
				fmt.Println("Remove temporary directory:", repoDir)
				if err = os.RemoveAll(repoDir); err != nil {
					handleError(err)
				}
			}
		},
	}

	releaseCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.baseBranch, "baseBranch", "b", "master", "Base branch of the release PR")
	cmd.Flags().StringVar(&f.releaseScript, "releaseScript", "scripts/prepare-release.sh", "Relative path to the script to run before creating the PR")
}

func newCreateReleaseCmd(f *createReleaseCmdFlags) (*createReleaseCmd, error) {
	var token string
	var err error
	if token, err = requireValue(GithubTokenKey); err != nil {
		return nil, err
	}
	var user string
	if user, err = requireValue(GithubUserKey); err != nil {
		return nil, err
	}
	client := newGithubClient(token)
	repoInfo := &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo}
	baseBranch := plumbing.NewBranchReferenceName(f.baseBranch)
	version, err := utils.NewRHMIVersion(releaseVersion)
	if err != nil {
		return nil, err
	}
	return &createReleaseCmd{
		version:         version,
		repoInfo:        repoInfo,
		baseBranch:      baseBranch,
		releaseScript:   f.releaseScript,
		githubPRService: client.PullRequests,
		gitUser:         user,
		gitPass:         token,
		gitCloneService: &services.DefaultGitCloneService{},
		gitPushService:  &services.DefaultGitPushService{},
	}, nil
}

func (c *createReleaseCmd) run(ctx context.Context) (string, error) {
	fmt.Println(fmt.Sprintf("Clone repo from %s/%s/%s.git to a temporary directory", githubURL, c.repoInfo.owner, c.repoInfo.repo))
	repoDir, gitRepo, err := c.gitCloneService.CloneToTmpDir("integreatly-operator", fmt.Sprintf("%s/%s/%s.git", githubURL, c.repoInfo.owner, c.repoInfo.repo), c.baseBranch)
	if err != nil {
		return "", err
	}
	fmt.Println(fmt.Sprintf("Repo cloned to %s", repoDir))

	gitRepoTree, err := gitRepo.Worktree()
	if err != nil {
		return "", err
	}

	releaseBranchName := fmt.Sprintf("prepare-for-release-%s", c.version.TagName())
	fmt.Println(fmt.Sprintf("Checkout branch %s", releaseBranchName))
	if err = checkoutBranchAndPullLatset(gitRepoTree, releaseBranchName); err != nil {
		return "", err
	}

	fmt.Println(fmt.Sprintf("Invoke release script: %s", c.releaseScript))
	if err = c.runReleaseScript(repoDir); err != nil {
		return "", err
	}

	status, err := gitRepoTree.Status()
	if err != nil {
		return "", err
	}
	if len(status) > 0 {
		if err = c.commitAndPushChanges(gitRepo, gitRepoTree); err != nil {
			return "", err
		}
	} else {
		fmt.Println("No new changes found")
	}

	if err = c.createPRIfNotExists(ctx, releaseBranchName); err != nil {
		return "", err
	}

	return repoDir, nil
}

func (c *createReleaseCmd) runReleaseScript(repoDir string) error {
	if err := os.Chmod(path.Join(repoDir, c.releaseScript), 0755); err != nil {
		return err
	}
	args := []string{"", "-v", c.version.Base(), "-t", c.version.Build()}
	releaseScript := &exec.Cmd{Dir: repoDir, Args: args, Path: c.releaseScript, Stdout: os.Stdout, Stderr: os.Stderr}
	fmt.Println("Run command:", releaseScript.String())
	return releaseScript.Run()
}

func (c *createReleaseCmd) commitAndPushChanges(gitRepo *git.Repository, gitRepoTree *git.Worktree) error {
	fmt.Println("Commit new changes")
	if err := gitRepoTree.AddGlob("."); err != nil {
		return err
	}
	if _, err := gitRepoTree.Commit(fmt.Sprintf("prepare for release %v", c.version), &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  commitAuthorName,
			Email: commitAuthorEmail,
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}

	fmt.Println("Push release branch")
	opts := &git.PushOptions{
		RemoteName: "origin",
		Auth:       &http.BasicAuth{Password: c.gitPass, Username: c.gitUser},
		Progress:   os.Stdout,
	}
	if err := c.gitPushService.Push(gitRepo, opts); err != nil {
		return err
	}
	return nil
}

func (c *createReleaseCmd) createPRIfNotExists(ctx context.Context, releaseBranchName string) error {
	h := fmt.Sprintf("%s:%s", c.repoInfo.owner, releaseBranchName)
	prOpts := &github.PullRequestListOptions{Base: c.baseBranch.String(), Head: h}
	pr, err := findPRForRelease(ctx, c.githubPRService, c.repoInfo, prOpts)
	if err != nil && !isPRNotFoundError(err) {
		return err
	}
	if pr == nil {
		fmt.Println("Create PR for release")
		t := fmt.Sprintf("release PR for version %s", c.version)
		b := c.baseBranch.String()
		req := &github.NewPullRequest{
			Title: &t,
			Head:  &h,
			Base:  &b,
		}
		pr, _, err = c.githubPRService.Create(ctx, c.repoInfo.owner, c.repoInfo.repo, req)
		if err != nil {
			return err
		}
	}
	fmt.Println(fmt.Sprintf("PR created: %s", pr.GetHTMLURL()))
	return nil
}

func checkoutBranchAndPullLatset(repoWorkTree *git.Worktree, branchName string) error {
	branch := plumbing.NewBranchReferenceName(branchName)
	if err := repoWorkTree.Checkout(&git.CheckoutOptions{
		Branch: branch,
		Create: true,
	}); err != nil {
		return err
	}
	return nil
}

func isPRNotFoundError(err error) bool {
	if strings.Index(err.Error(), "no open pull request found") > -1 {
		return true
	}
	return false
}
