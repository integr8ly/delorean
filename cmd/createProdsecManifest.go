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
	"io"
	"os"
	"os/exec"
	"path"
	"time"
)

type createProdsecManifestCmdFlags struct {
	baseBranch     string
	manifestScript string
}

type createProdsecManifestCmd struct {
	version         *utils.RHMIVersion
	repoInfo        *githubRepoInfo
	baseBranch      plumbing.ReferenceName
	githubPRService services.PullRequestsService
	manifestScript  string
	gitUser         string
	gitPass         string
	gitCloneService services.GitCloneService
	gitPushService  services.GitPushService
}

func init() {
	f := &createProdsecManifestCmdFlags{}

	cmd := &cobra.Command{
		Use:   "create-prodsec-manifest",
		Short: "Create a production manifest of a given version and olm type",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newCreateProdsecManifestCmd(f)
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
	cmd.Flags().StringVarP(&f.baseBranch, "branch", "b", "master", "Base branch of the manifest generation")
	cmd.Flags().StringVar(&f.manifestScript, "manifestGenerationScript", "scripts/prodsec-manifest-generator.sh", "Relative path to the script to run before creating the PR")
}

func newCreateProdsecManifestCmd(f *createProdsecManifestCmdFlags) (*createProdsecManifestCmd, error) {
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
	version, err := utils.NewVersion(releaseVersion, olmType)
	if err != nil {
		return nil, err
	}
	return &createProdsecManifestCmd{
		version:         version,
		repoInfo:        repoInfo,
		baseBranch:      baseBranch,
		manifestScript:  f.manifestScript,
		githubPRService: client.PullRequests,
		gitUser:         user,
		gitPass:         token,
		gitCloneService: &services.DefaultGitCloneService{},
		gitPushService:  &services.DefaultGitPushService{},
	}, nil
}

func (c *createProdsecManifestCmd) run(ctx context.Context) (string, error) {
	// Clone the repo
	fmt.Println(fmt.Sprintf("Clone repo from %s/%s/%s.git to a temporary directory", githubURL, c.repoInfo.owner, c.repoInfo.repo))
	repoDir, gitRepo, err := c.gitCloneService.CloneToTmpDir("integreatly-operator", fmt.Sprintf("%s/%s/%s.git", githubURL, c.repoInfo.owner, c.repoInfo.repo), c.baseBranch)
	if err != nil {
		return "", err
	}

	fmt.Println("Repo cloned to", repoDir)
	gitRepoTree, err := gitRepo.Worktree()
	if err != nil {
		return "", err
	}

	// Checking out the OLM_TYPE-release-VERSION branch as the manifest generation script must be run from this branch
	releasedBranchName := c.version.ReleaseBranchName()
	fmt.Println(fmt.Sprintf("Checkout branch %s", releasedBranchName))
	if err = checkoutBranch(gitRepoTree, false, false, releasedBranchName); err != nil {
		return "", err
	}

	// Invoking manifest generation script
	fmt.Println(fmt.Sprintf("Generate manifest: %s", c.manifestScript))
	if err = c.runManifestScript(repoDir); err != nil {
		return "", err
	}

	manifestFileName := fmt.Sprintf("%s-production-release-manifest.txt", c.version.NameByOlmType())
	manifestPath := fmt.Sprintf("%s/prodsec-manifests/%s", repoDir, manifestFileName)
	temporaryManifestPath := fmt.Sprintf("%s/../temporary-manifest.txt", repoDir)

	// Saving generated manifest file - reason for it is as we need to checkout master without saving local changes to the repo.
	if err = copyManifestFile(manifestPath, temporaryManifestPath); err != nil {
		return "", err
	}

	// Checking out master branch to be able to checkout new branch from it
	fmt.Println("Checkout master branch")
	if err = checkoutBranch(gitRepoTree, true, false, "master"); err != nil {
		return "", err
	}

	// Preparing new branch called OLM_TYPE-manifest-for-release-OLM_TYPE-VERSION of release
	manifestReleaseBranchName := c.version.PrepareProdsecManifestBranchName()
	fmt.Println(fmt.Sprintf("Create new branch %s", manifestReleaseBranchName))
	if err = checkoutBranch(gitRepoTree, true, true, manifestReleaseBranchName); err != nil {
		return "", err
	}

	// Restoring saved file to a new branch
	if err = copyManifestFile(temporaryManifestPath, manifestPath); err != nil {
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
		fmt.Println("No new changes found - seems that repo has up-to-date production manifest!")
		return repoDir, nil
	}

	if err = c.createPRIfNotExists(ctx, manifestReleaseBranchName); err != nil {
		return "", err
	}

	return repoDir, nil
}

func (c *createProdsecManifestCmd) runManifestScript(repoDir string) error {
	if err := os.Chmod(path.Join(repoDir, c.manifestScript), 0755); err != nil {
		return err
	}

	envs := []string{fmt.Sprintf("OLM_TYPE=%s", c.version.OlmType()), "TYPE_OF_MANIFEST=production", fmt.Sprintf("PATH=%s", os.Getenv("PATH")), "HOME=/tmp"}

	manifestGeneratorScript := &exec.Cmd{Dir: repoDir, Env: envs, Path: c.manifestScript, Stdout: os.Stdout, Stderr: os.Stderr}
	return manifestGeneratorScript.Run()
}

func (c *createProdsecManifestCmd) commitAndPushChanges(gitRepo *git.Repository, gitRepoTree *git.Worktree) error {
	fmt.Println("Commit new changes")
	if err := gitRepoTree.AddGlob("."); err != nil {
		return err
	}
	if _, err := gitRepoTree.Commit(fmt.Sprintf("Prepare %s manifest for version %s", c.version.OlmType(), c.version.TagName()), &git.CommitOptions{
		Author: &object.Signature{
			Name:  commitAuthorName,
			Email: commitAuthorEmail,
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}

	fmt.Println("Push manifest release branch")
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

func (c *createProdsecManifestCmd) createPRIfNotExists(ctx context.Context, releaseBranchName string) error {
	h := fmt.Sprintf("%s:%s", c.repoInfo.owner, releaseBranchName)
	prOpts := &github.PullRequestListOptions{Base: c.baseBranch.String(), Head: h}
	pr, err := findPRForRelease(ctx, c.githubPRService, c.repoInfo, prOpts)
	if err != nil && !isPRNotFoundError(err) {
		return err
	}
	if pr == nil {
		fmt.Printf("Create PR for %s manifest version - %s", c.version.OlmType(), c.version.TagName())
		t := fmt.Sprintf("Manifest PR for version %s", c.version.TagName())
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

func checkoutBranch(repoWorkTree *git.Worktree, removeChanges bool, create bool, branchName string) error {
	branch := plumbing.NewBranchReferenceName(branchName)
	if err := repoWorkTree.Checkout(&git.CheckoutOptions{
		Force:  removeChanges,
		Branch: branch,
		Create: create,
	}); err != nil {
		return err
	}
	return nil
}

func copyManifestFile(copyFrom string, copyTo string) error {
	from, err := os.Open(copyFrom)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(copyTo, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}
