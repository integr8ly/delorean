package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

const (
	// Base URL for gitlab API and for the managed-tenenats fork and origin repos
	gitlabURL = "https://gitlab.cee.redhat.com"

	gitlabAPIEndpoint = "api/v4"

	// Base URL for the integreatly-opeartor repo
	githubURL = "https://github.com"

	// Directory in the integreatly-opeartor repo with the OLM maninfest files
	sourceOLMManifestsDirectory = "deploy/olm-catalog/integreatly-operator/integreatly-operator-%s"

	// The branch to target with the merge request
	managedTenantsMasterBranch = "master"

	// Info for the commit and merge request
	branchNameTemplate        = "integreatly-operator-%s-v%s"
	commitMessageTemplate     = "update integreatly-operator %s to %s"
	gitlabAuthorName          = "Delorean"
	gitlabAuthorEmail         = "cloud-services-delorean@redhat.com"
	mergeRequestTitleTemplate = "Update integreatly-operator %s to %s" // channel, version
)

// releaseChannel rappresents one of the three places (stage, edge, stable)
// where to update the integreatly-operator
type releaseChannel string

const (
	stageChannel  releaseChannel = "stage"
	edgeChannel   releaseChannel = "edge"
	stableChannel releaseChannel = "stable"
)

// directory returns the relative path of the managed-teneants repo to the
// integreatly-operator for the given channel
func (c releaseChannel) directory() string {

	name := c.operatorName()

	var template string
	switch c {
	case stageChannel:
		template = "addons-stage/%s"
	case edgeChannel:
		template = "addons-production/%s"
	case stableChannel:
		template = "addons-production/%s"
	default:
		panic(fmt.Sprintf("unsopported channel %s", c))
	}

	return fmt.Sprintf(template, name)
}

// OperatorName returns the name of the integreatly-operator depending on the channel
func (c releaseChannel) operatorName() string {

	switch c {
	case stageChannel, stableChannel:
		return "integreatly-operator"
	case edgeChannel:
		return "integreatly-operator-internal"
	default:
		panic(fmt.Sprintf("unsopported channel %s", c))
	}
}

// c.print.Printf("clone the managed-tenants repo to %s\n", managedTenatDirectory)
func gitCloneToTmp(prefix string, url string, reference plumbing.ReferenceName) (string, *git.Repository, error) {

	// Clone the managed tenants
	dir, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		return "", nil, err
	}

	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: reference,
	})
	if err != nil {
		return "", nil, err
	}

	return dir, repo, nil
}

type osdAddonReleaseFlags struct {
	version                 string
	gitlabToken             string
	mergeRequestDescription string
	managedTenantsOrigin    string
	managedTenantsFork      string
}

type osdAddonReleaseCmd struct {
	flags                  *osdAddonReleaseFlags
	version                *utils.RHMIVersion
	gitlabMergeRequests    services.GitLabMergeRequestsService
	gitlabProjects         services.GitLabProjectsService
	integreatlyOperatorDir string
	managedTenantsDir      string
	managedTenantsRepo     services.GitRepositoryService
}

func init() {

	f := &osdAddonReleaseFlags{}

	cmd := &cobra.Command{
		Use:   "osd-addon",
		Short: "Create the integreatly-operator MR to the managed-tenants repo",
		Run: func(cmd *cobra.Command, args []string) {

			// Prepare
			c, err := newOSDAddonReleseCmd(f)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				os.Exit(1)
			}

			// Run
			err = c.run()
			if err != nil {
				fmt.Printf("error: %s\n", err)
				os.Exit(1)
			}
		},
	}

	releaseCmd.AddCommand(cmd)

	cmd.Flags().StringVar(
		&f.version, "version", "",
		"The RHMI version to push to the managed-tenats repo (ex \"2.0.0\", \"2.0.0-er4\")")
	cmd.MarkFlagRequired("version")

	cmd.Flags().StringVar(
		&f.gitlabToken,
		"gitlab-token",
		"",
		"GitLab token to Push the changes and open the MR")
	cmd.MarkFlagRequired("gitlab-token")

	cmd.Flags().StringVar(
		&f.mergeRequestDescription,
		"merge-request-description",
		"",
		"Optional merge request description that can be used to notify secific users (ex \"ping: @dbizzarr\")",
	)

	cmd.Flags().StringVar(
		&f.managedTenantsOrigin,
		"managed-tenants-origin",
		"service/managed-tenants",
		"managed-tenants origin repository from where to frok the master branch")

	cmd.Flags().StringVar(
		&f.managedTenantsFork,
		"managed-tenants-fork",
		"integreatly-qe/managed-tenants",
		"managed-tenants fork repository where to push the release files")
}

func newOSDAddonReleseCmd(flags *osdAddonReleaseFlags) (*osdAddonReleaseCmd, error) {

	version, err := utils.NewRHMIVersion(flags.version)
	if err != nil {
		return nil, err
	}
	fmt.Printf("create osd addon release for RHMI v%s\n", version)

	// Prepare the GitLab Client
	gitlabClient, err := gitlab.NewClient(
		flags.gitlabToken,
		gitlab.WithBaseURL(fmt.Sprintf("%s/%s", gitlabURL, gitlabAPIEndpoint)),
	)
	if err != nil {
		return nil, err
	}
	fmt.Print("gitlab client initialized and authenticated\n")

	// Clone the managed tenants
	managedTenatsDir, managedTenantsRepo, err := gitCloneToTmp(
		"managed-tenants-",
		fmt.Sprintf("%s/%s", gitlabURL, flags.managedTenantsOrigin),
		plumbing.NewBranchReferenceName(managedTenantsMasterBranch),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("managed-tenants repo cloned to %s\n", managedTenatsDir)

	// Add the fork remote to the managed-tenats repo
	_, err = managedTenantsRepo.CreateRemote(&config.RemoteConfig{
		Name: "fork",
		URLs: []string{fmt.Sprintf("%s/%s", gitlabURL, flags.managedTenantsFork)},
	})
	if err != nil {
		return nil, err
	}
	fmt.Print("added the fork remote to the managed-tenants repo\n")

	// Clone the integreatly-operator
	integreatlyOperatorDir, _, err := gitCloneToTmp(
		"integreatly-operator-",
		fmt.Sprintf("%s/%s/%s", githubURL, integreatlyGHOrg, integreatlyOperatorRepo),
		plumbing.NewTagReferenceName(fmt.Sprintf("v%s", version)),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("integreatly-operator cloned to %s\n", integreatlyOperatorDir)

	return &osdAddonReleaseCmd{
		flags:                  flags,
		version:                version,
		gitlabMergeRequests:    gitlabClient.MergeRequests,
		gitlabProjects:         gitlabClient.Projects,
		integreatlyOperatorDir: integreatlyOperatorDir,
		managedTenantsDir:      managedTenatsDir,
		managedTenantsRepo:     managedTenantsRepo,
	}, nil
}

func (c *osdAddonReleaseCmd) run() error {

	if c.version.IsPreRrelease() {

		// Release to stage
		err := c.createReleaseMergeRequest(stageChannel)
		if err != nil {
			return err
		}

	} else {

		// When the version is not a prerelease version and is a final release
		// then create the release against stage, edge and stable
		for _, channel := range []releaseChannel{stageChannel, edgeChannel, stableChannel} {
			err := c.createReleaseMergeRequest(channel)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *osdAddonReleaseCmd) createReleaseMergeRequest(channel releaseChannel) error {

	e := func(err error) error {
		return fmt.Errorf("failed to create the MR for the version %s and channel %s: %s", c.version, channel, err)
	}

	managedTenantsHead, err := c.managedTenantsRepo.Head()
	if err != nil {
		return e(err)
	}

	// Verify that the repo is on master
	if managedTenantsHead.Name() != plumbing.NewBranchReferenceName(managedTenantsMasterBranch) {
		return e(fmt.Errorf("the managed-tenants repo is pointing to %s insteand of master", managedTenantsHead.Name()))
	}

	managedTenantsTree, err := c.managedTenantsRepo.Worktree()
	if err != nil {
		return e(err)
	}

	// Create a new branch on the managed-tenants repo
	managedTenantsBranch := fmt.Sprintf(branchNameTemplate, channel, c.version)

	fmt.Printf("create the branch %s in the managed-tenants repo\n", managedTenantsBranch)
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(managedTenantsBranch),
		Create: true,
	})
	if err != nil {
		return e(err)
	}

	// Copy the OLM manifests from the integreatly-operator repo to the the managed-tenats repo
	manifestsDirectory, err := c.copyTheOLMManifests(channel)
	if err != nil {
		return e(err)
	}

	// Add all changes
	err = managedTenantsTree.AddGlob(fmt.Sprintf("%s/*", manifestsDirectory))
	if err != nil {
		return e(err)
	}

	// Update the integreatly-operator.package.yaml
	packageManfiest, err := c.udpateThePackageManifest(channel)
	if err != nil {
		return e(err)
	}

	// Add the integreatly-operator.package.yaml
	_, err = managedTenantsTree.Add(packageManfiest)
	if err != nil {
		return e(err)
	}

	// Commit
	fmt.Print("commit all changes in the managed-tenats repo\n")
	_, err = managedTenantsTree.Commit(
		fmt.Sprintf(commitMessageTemplate, channel, c.version),
		&git.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  gitlabAuthorName,
				Email: gitlabAuthorEmail,
				When:  time.Now(),
			},
		},
	)
	if err != nil {
		return e(err)
	}

	// Verify tha the tree is clean
	status, err := managedTenantsTree.Status()
	if err != nil {
		return e(err)
	}

	if len(status) != 0 {
		return e(fmt.Errorf("the tree is not clean, uncommited changes:\n%+v", status))
	}

	// Push to fork
	fmt.Printf("push the managed-tenats repo to the fork remote\n")
	err = c.managedTenantsRepo.Push(&git.PushOptions{
		RemoteName: "fork",
		Auth:       &http.BasicAuth{Password: c.flags.gitlabToken},
	})
	if err != nil {
		return e(err)
	}

	// Create the merge request
	targetProject, _, err := c.gitlabProjects.GetProject(c.flags.managedTenantsOrigin, &gitlab.GetProjectOptions{})
	if err != nil {
		return e(err)
	}

	fmt.Print("create the MR to the managed-tenants origin\n")
	mr, _, err := c.gitlabMergeRequests.CreateMergeRequest(c.flags.managedTenantsFork, &gitlab.CreateMergeRequestOptions{
		Title:           gitlab.String(fmt.Sprintf(mergeRequestTitleTemplate, channel, c.version)),
		Description:     gitlab.String(c.flags.mergeRequestDescription),
		SourceBranch:    gitlab.String(managedTenantsBranch),
		TargetBranch:    gitlab.String(managedTenantsMasterBranch),
		TargetProjectID: gitlab.Int(targetProject.ID),
	})
	if err != nil {
		return e(err)
	}

	fmt.Printf("merge request for version %s and channel %s created successfully\n", c.version, channel)
	fmt.Printf("MR: %s\n", mr.WebURL)

	// Reset the managed repostiroy to master
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(managedTenantsMasterBranch)})
	if err != nil {
		return e(err)
	}

	return nil
}

func (c *osdAddonReleaseCmd) copyTheOLMManifests(channel releaseChannel) (string, error) {

	source := path.Join(c.integreatlyOperatorDir, fmt.Sprintf(sourceOLMManifestsDirectory, c.version))

	relativeDestination := fmt.Sprintf("%s/%s", channel.directory(), c.version)
	destination := path.Join(c.managedTenantsDir, relativeDestination)

	fmt.Printf("copy files from %s to %s\n", source, destination)
	err := utils.CopyDirectory(source, destination)
	if err != nil {
		return "", err
	}

	return relativeDestination, nil
}

func (c *osdAddonReleaseCmd) udpateThePackageManifest(channel releaseChannel) (string, error) {

	relative := fmt.Sprintf("%s/%s.package.yaml", channel.directory(), channel.operatorName())
	manifest := path.Join(c.managedTenantsDir, relative)

	fmt.Printf("update the version of the manifest files %s to %s\n", relative, c.version)
	read, err := os.Open(manifest)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(read)

	err = read.Close()
	if err != nil {
		return "", err
	}

	var p registry.PackageManifest
	err = yaml.Unmarshal(bytes, &p)
	if err != nil {
		return "", err
	}

	// Set channels[0].currentCSV value
	p.Channels[0].CurrentCSVName = fmt.Sprintf("integreatly-operator.v%s", c.version)

	bytes, err = yaml.Marshal(p)
	if err != nil {
		return "", err
	}

	// truncate the existing file
	write, err := os.Create(manifest)
	if err != nil {
		return "", err
	}

	_, err = write.Write(bytes)
	if err != nil {
		return "", err
	}

	err = write.Close()
	if err != nil {
		return "", err
	}

	return relative, nil
}
