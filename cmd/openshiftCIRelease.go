package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/integr8ly/delorean/pkg/types"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

const (
	DefaultOpenshiftCIRepo        = "release"
	DefaultOpenshiftCIOrgUpstream = "openshift"
	DefaultOpenshiftCIOrgOrigin   = "integr8ly"
)

type openshiftCIReleaseCmdFlags struct {
	openshiftCIRepo        string
	openshiftCIOrgUpstream string
	openshiftCIOrgOrigin   string
	baseBranch             string
}

type openshiftCIReleaseCmd struct {
	version                 *utils.RHMIVersion
	baseBranch              plumbing.ReferenceName
	intlyRepoInfo           *githubRepoInfo
	releaseRepoInfoOrigin   *githubRepoInfo
	releaseRepoInfoUpstream *githubRepoInfo
	githubPRService         services.PullRequestsService
	gitUser                 string
	gitPass                 string
	gitCloneService         services.GitCloneService
	gitPushService          services.GitPushService
	gitRemoteService        services.GitRemoteService
}

/**
// Integreatly Operator updates
//1. Clone the integreatly operator repo and ensure it's on the correct initial minor release tag (v2.1.0, v2.2.0 etc..) for the given tag (v2.1.1, v2.2.1 etc..)
//2. Create a new release branch with the correct naming we expect (release-v2.1) if one does not already exist
//3. Push the branch
*/
func (c *openshiftCIReleaseCmd) DoIntlyOperatorUpdate(ctx context.Context) (string, error) {
	//Clone integreatly operator repo to a temp directory using the base branch
	fmt.Println(fmt.Sprintf("Clone repo from %s/%s/%s.git (%s) to a temporary directory", githubURL, c.intlyRepoInfo.owner, c.intlyRepoInfo.repo, c.baseBranch.String()))
	repoDir, gitRepo, err := c.gitCloneService.CloneToTmpDir("integreatly-operator", fmt.Sprintf("%s/%s/%s.git", githubURL, c.intlyRepoInfo.owner, c.intlyRepoInfo.repo), c.baseBranch)
	if err != nil {
		return "", err
	}
	fmt.Println("Repo cloned to", repoDir)

	worktree, err := gitRepo.Worktree()
	if err != nil {
		return "", err
	}

	//Ensure release branch exists
	branch := plumbing.NewBranchReferenceName(c.version.ReleaseBranchName())
	fmt.Println(fmt.Sprintf("Checkout branch %s", branch))
	err = worktree.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: branch})
	if err != nil {
		err := worktree.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: branch})
		if err != nil {
			return "", nil
		}
		fmt.Println(fmt.Sprintf("Created new branch %s", branch))

		pushOpts := &git.PushOptions{
			RemoteName: "origin",
			Auth:       &http.BasicAuth{Password: c.gitPass, Username: c.gitUser},
			Progress:   os.Stdout,
			RefSpecs: []config.RefSpec{
				config.RefSpec(branch + ":" + branch),
			},
		}
		if err := c.gitPushService.Push(gitRepo, pushOpts); err != nil {
			return "", err
		}
		fmt.Println(fmt.Sprintf("Pushed branch %s", branch))
	}

	return repoDir, nil
}

/**
  OpenShift Release updates
  1. Clone the release repo and ensure it is up to date
  2. Create configuration in the release repo for this new branch
  3. Add ci-operator config for branch (https://github.com/openshift/release/blob/master/ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml)
  4. Update promotion config to ensure it pushes the images to the registry with a different tag name (branch name)
  5. Run Make jobs
  6. Update image mirroring to include entries for the new release branch https://github.com/openshift/release/blob/master/core-services/image-mirroring/integr8ly/mapping_integr8ly_operator
  7. Commit all updated and generate configuration
  8. Open a PR
*/
func (c *openshiftCIReleaseCmd) DoOpenShiftReleaseUpdate(ctx context.Context) (string, error) {
	//Clone the release repo to a temp directory
	baseBranch := plumbing.NewBranchReferenceName("master")
	fmt.Println(fmt.Sprintf("Clone repo from %s/%s/%s.git (%s) to a temporary directory", githubURL, c.releaseRepoInfoOrigin.owner, c.releaseRepoInfoOrigin.repo, baseBranch))
	repoDir, gitRepo, err := c.gitCloneService.CloneToTmpDir("release", fmt.Sprintf("%s/%s/%s.git", githubURL, c.releaseRepoInfoOrigin.owner, c.releaseRepoInfoOrigin.repo), baseBranch)
	if err != nil {
		return "", err
	}
	fmt.Println("Repo cloned to", repoDir)

	//Add remote for release repo upstream
	upstream := fmt.Sprintf("%s/%s/%s", githubURL, c.releaseRepoInfoUpstream.owner, c.releaseRepoInfoUpstream.repo)
	_, err = c.gitRemoteService.CreateAndPull(gitRepo, &config.RemoteConfig{
		Name: "upstream",
		URLs: []string{upstream},
	})
	if err != nil {
		return "", err
	}
	fmt.Println(fmt.Sprintf("Added upstream remote (%s)", upstream))

	worktree, err := gitRepo.Worktree()
	if err != nil {
		return "", err
	}

	//Ensure release branch exists
	branch := plumbing.NewBranchReferenceName(c.version.ReleaseBranchName())
	fmt.Println(fmt.Sprintf("Checkout branch %s", branch))
	err = worktree.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: branch})
	if err != nil {
		err := worktree.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: branch})
		if err != nil {
			return "", nil
		}
		fmt.Println(fmt.Sprintf("Created new branch %s", branch))
	}

	//Update CI Operator Config
	err = updateCIOperatorConfig(repoDir, c.version)
	if err != nil {
		return "", err
	}
	commitMsg := fmt.Sprintf("Updated ci operator config for branch %s", branch.Short())

	// Add all changes
	err = worktree.AddGlob("ci-operator/*")
	if err != nil {
		return "", err
	}

	//Update Image Mirror Mapping Config
	err = updateImageMirroringConfig(repoDir, c.version)
	if err != nil {
		return "", err
	}
	commitMsg += fmt.Sprintf("\nUpdated image mirror mapping config for branch %s", branch.Short())

	// Add all changes
	err = worktree.AddGlob("core-services/*")
	if err != nil {
		return "", err
	}

	status, err := worktree.Status()
	if err != nil {
		return "", err
	}

	if len(status) == 0 {
		fmt.Println("No new changes found")
		return repoDir, nil
	}
	// Commit
	fmt.Println(commitMsg)
	_, err = worktree.Commit(
		commitMsg,
		&git.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  commitAuthorName,
				Email: commitAuthorEmail,
				When:  time.Now(),
			},
		},
	)
	if err != nil {
		return "", err
	}

	//Push changes
	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Auth:       &http.BasicAuth{Password: c.gitPass, Username: c.gitUser},
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(branch + ":" + branch),
		},
	}
	err = c.gitPushService.Push(gitRepo, pushOpts)
	if err != nil {
		return "", err
	}
	fmt.Println(fmt.Sprintf("Pushed branch %s", branch))

	//Open Pull Request
	title := fmt.Sprintf("Add CI config for RHMI operator branch %s", branch.Short())
	head := fmt.Sprintf("%s:%s", c.releaseRepoInfoOrigin.owner, branch.Short())
	base := "master"
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
	}
	_, err = c.createPRIfNotExists(ctx, newPR)
	if err != nil {
		return "", err
	}

	return repoDir, nil
}

func (c *openshiftCIReleaseCmd) createPRIfNotExists(ctx context.Context, newPR *github.NewPullRequest) (*github.PullRequest, error) {
	prListOpts := &github.PullRequestListOptions{Base: *newPR.Base, Head: *newPR.Head}
	pr, err := findPRForRelease(ctx, c.githubPRService, c.releaseRepoInfoUpstream, prListOpts)
	if err != nil && !isPRNotFoundError(err) {
		return nil, err
	}
	if pr == nil {
		fmt.Println("Create PR for release")
		pr, _, err = c.githubPRService.Create(ctx, c.releaseRepoInfoUpstream.owner, c.releaseRepoInfoUpstream.repo, newPR)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println(fmt.Sprintf("PR created: %s", pr.GetHTMLURL()))
	return pr, nil
}

func updateCIOperatorConfig(repoDir string, version *utils.RHMIVersion) error {
	masterConfig := path.Join(repoDir, "ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-master.yaml")
	releaseConfig := path.Join(repoDir, fmt.Sprintf("ci-operator/config/integr8ly/integreatly-operator/integr8ly-integreatly-operator-%s.yaml", version.ReleaseBranchName()))

	y, err := utils.LoadUnstructYaml(masterConfig)
	if err != nil {
		return err
	}

	err = y.Set("promotion.name", version.MajorMinor())
	if err != nil {
		return err
	}

	err = y.Set("zz_generated_metadata.branch", version.ReleaseBranchName())
	if err != nil {
		return err
	}

	err = y.Write(releaseConfig)
	if err != nil {
		return err
	}

	makeExecutable, err := exec.LookPath("make")
	if err != nil {
		return err
	}

	makeJobCmd := &exec.Cmd{
		Dir:    repoDir,
		Path:   makeExecutable,
		Args:   []string{makeExecutable, "jobs"},
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}

	err = makeJobCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func updateImageMirroringConfig(repoDir string, version *utils.RHMIVersion) error {
	mappingFile := path.Join(repoDir, fmt.Sprintf("core-services/image-mirroring/integr8ly/mapping_integr8ly_operator_%s", strings.ReplaceAll(version.MajorMinor(), ".", "_")))

	internalReg := "registry.svc.ci.openshift.org/integr8ly"
	publicReg := "quay.io/integreatly"

	type imageTemplate struct {
		internalRegTemplate string
		externalRegTemplate string
	}

	var operatorImage imageTemplate
	operatorImage.internalRegTemplate = "%s/%s:integreatly-operator"

	switch version.OlmType() {
	case types.OlmTypeRhmi:
		operatorImage.externalRegTemplate = "%s/integreatly-operator:%s"
	case types.OlmTypeRhoam:
		operatorImage.externalRegTemplate = "%s/managed-api-service:%s"
	default:
		operatorImage.externalRegTemplate = "%s/integreatly-operator:%s"
	}

	imageTemplates := []imageTemplate{
		operatorImage,
		{
			internalRegTemplate: "%s/%s:integreatly-operator-test-harness",
			externalRegTemplate: "%s/integreatly-operator-test-harness:%s",
		},
	}

	file, err := os.Create(mappingFile)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	for _, t := range imageTemplates {
		internalImage := fmt.Sprintf(t.internalRegTemplate, internalReg, version.MajorMinor())
		publicImage := fmt.Sprintf(t.externalRegTemplate, publicReg, version.MajorMinor())
		mapping := fmt.Sprintf("%s %s\n", internalImage, publicImage)
		w.WriteString(mapping)
	}

	return w.Flush()
}

func init() {

	f := &openshiftCIReleaseCmdFlags{}

	cmd := &cobra.Command{
		Use:   "openshift-ci-release",
		Short: "Update openshift CI Release repo",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := newOpenshiftCIReleaseCmd(f)
			if err != nil {
				handleError(err)
			}
			if c.version.IsPreRelease() {
				fmt.Println("Skipping the update to Openshift CI release repo as the release version is not a major or a minor one")
				return
			}
			var intlyOperatorRepoDir string
			if intlyOperatorRepoDir, err = c.DoIntlyOperatorUpdate(cmd.Context()); err != nil {
				handleError(err)
			}
			if intlyOperatorRepoDir != "" {
				fmt.Println("Remove temporary directory:", intlyOperatorRepoDir)
				if err = os.RemoveAll(intlyOperatorRepoDir); err != nil {
					handleError(err)
				}
			}
			var ciReleaseRepoDir string
			if ciReleaseRepoDir, err = c.DoOpenShiftReleaseUpdate(cmd.Context()); err != nil {
				handleError(err)
			}
			if ciReleaseRepoDir != "" {
				fmt.Println("Remove temporary directory:", ciReleaseRepoDir)
				if err = os.RemoveAll(ciReleaseRepoDir); err != nil {
					handleError(err)
				}
			}
		},
	}

	releaseCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.baseBranch, "branch", "b", "master", "Base branch of the release PR")
	cmd.Flags().StringVar(&f.openshiftCIOrgUpstream, "ci-org-upstream", DefaultOpenshiftCIOrgUpstream, "OpenShift CI Release GitHub org (Upstream)")
	cmd.Flags().StringVar(&f.openshiftCIOrgOrigin, "ci-org-origin", DefaultOpenshiftCIOrgOrigin, "OpenShift CI Release GitHub org (Origin)")
	cmd.Flags().StringVar(&f.openshiftCIRepo, "ci-repo", DefaultOpenshiftCIRepo, "OpenShift CI Release GitHub repo")
}

func newOpenshiftCIReleaseCmd(f *openshiftCIReleaseCmdFlags) (*openshiftCIReleaseCmd, error) {
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
	version, err := utils.NewVersion(releaseVersion, olmType)
	if err != nil {
		return nil, err
	}
	baseBranch := plumbing.NewBranchReferenceName(f.baseBranch)
	return &openshiftCIReleaseCmd{
		version:                 version,
		baseBranch:              baseBranch,
		releaseRepoInfoUpstream: &githubRepoInfo{owner: f.openshiftCIOrgUpstream, repo: f.openshiftCIRepo},
		releaseRepoInfoOrigin:   &githubRepoInfo{owner: f.openshiftCIOrgOrigin, repo: f.openshiftCIRepo},
		intlyRepoInfo:           &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo},
		githubPRService:         client.PullRequests,
		gitUser:                 user,
		gitPass:                 token,
		gitCloneService:         &services.DefaultGitCloneService{},
		gitPushService:          &services.DefaultGitPushService{},
		gitRemoteService:        &services.DefaultGitRemoteService{},
	}, nil
}
