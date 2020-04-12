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
	mergeRequestTitleTemplate = "Update integreatly-operator %s to %s" // environment, version
)

type gitServiceImpl struct{}

func (g *gitServiceImpl) PlainClone(path string, isBare bool, o *git.CloneOptions) (*git.Repository, error) {
	return git.PlainClone(path, isBare, o)
}

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
func (c *releaseChannel) directory() string {

	name := c.operatorName()

	var template string
	switch *c {
	case stageChannel:
		template = "addons-stage/%s"
	case edgeChannel:
		template = "addons-production/%s"
	case stableChannel:
		template = "addons-production/%s"
	default:
		panic(fmt.Sprintf("unsopported channel %s", *c))
	}

	return fmt.Sprintf(template, name)
}

// OperatorName returns the name of the integreatly-operator depending on the channel
func (c *releaseChannel) operatorName() string {

	switch *c {
	case stageChannel, stableChannel:
		return "integreatly-operator"
	case edgeChannel:
		return "integreatly-operator-internal"
	default:
		panic(fmt.Sprintf("unsopported channel %s", *c))
	}
}

type osdAddonReleaseCmd struct {
	versionFlag                 string
	gitlabUsernameFlag          string
	gitlabTokenFlag             string
	mergeRequestDescriptionFlag string
	managedTenantsOriginFlag    string
	managedTenantsForkFlag      string
	integreatlyOperatorFlag     string

	print               services.PrintService
	git                 services.GitService
	gitlabMergeRequests services.GitLabMergeRequestsService
	gitlabProjects      services.GitLabProjectsService
}

func (c *osdAddonReleaseCmd) copyTheOLMManifests(
	managedTenantsDirectory string,
	integreatlyOperatorDirectory string,
	channel releaseChannel,
	version *utils.RHMIVersion) (string, error) {

	source := path.Join(integreatlyOperatorDirectory, fmt.Sprintf(sourceOLMManifestsDirectory, version))

	relativeDestination := fmt.Sprintf("%s/%s", channel.directory(), version.String())
	destination := path.Join(managedTenantsDirectory, relativeDestination)

	c.print.Printf("copy files from %s to %s\n", source, destination)
	err := utils.CopyDirectory(source, destination)
	if err != nil {
		return "", err
	}

	return relativeDestination, nil
}

func (c *osdAddonReleaseCmd) udpateThePackageManifest(
	managedTenantsDirectory string,
	channel releaseChannel,
	version *utils.RHMIVersion) (string, error) {

	relative := fmt.Sprintf("%s/%s.package.yaml", channel.directory(), channel.operatorName())
	manifest := path.Join(managedTenantsDirectory, relative)

	c.print.Printf("upte the version of the manifest files %s to %s\n", relative, version)
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
	p.Channels[0].CurrentCSVName = fmt.Sprintf("integreatly-operator.v%s", version)

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

func (c *osdAddonReleaseCmd) createTheReleaseMergeRequest(
	integreatlyOperatorDirectory string,
	managedTenantsDirectory string,
	managedTenantsRepostiroy services.GitRepositoryService,
	version *utils.RHMIVersion,
	channel releaseChannel,
) error {

	e := func(err error) error {
		return fmt.Errorf("failed to create the MR for the version %s and channel %s: %s", version, channel, err)
	}

	managedTenantsHead, err := managedTenantsRepostiroy.Head()
	if err != nil {
		return e(err)
	}

	// Verify that the repo is on master
	if managedTenantsHead.Name() != plumbing.NewBranchReferenceName(managedTenantsMasterBranch) {
		return e(fmt.Errorf("the managed-tenants repo is pointing to %s insteand of master", managedTenantsHead.Name()))
	}

	managedTenantsTree, err := managedTenantsRepostiroy.Worktree()
	if err != nil {
		return e(err)
	}

	// Create a new branch on the managed-tenants repo
	managedTenantsBranch := fmt.Sprintf(branchNameTemplate, channel, version)

	c.print.Printf("create the branch %s in the managed-tenants repo\n", managedTenantsBranch)
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(managedTenantsBranch),
		Create: true,
	})
	if err != nil {
		return e(err)
	}

	// Copy the OLM manifests from the integreatly-operator repo to the the managed-tenats repo
	manifestsDirectory, err := c.copyTheOLMManifests(
		managedTenantsDirectory,
		integreatlyOperatorDirectory,
		channel,
		version,
	)
	if err != nil {
		return e(err)
	}

	// Add all changes
	err = managedTenantsTree.AddGlob(fmt.Sprintf("%s/*", manifestsDirectory))
	if err != nil {
		return e(err)
	}

	// Update the integreatly-operator.package.yaml
	packageManfiest, err := c.udpateThePackageManifest(managedTenantsDirectory, channel, version)
	if err != nil {
		return e(err)
	}

	// Add the integreatly-operator.package.yaml
	_, err = managedTenantsTree.Add(packageManfiest)
	if err != nil {
		return e(err)
	}

	// Commit
	c.print.Print("commit all changes in the managed-tenats repo\n")
	_, err = managedTenantsTree.Commit(
		fmt.Sprintf(commitMessageTemplate, channel, version),
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
	c.print.Printf("push the managed-tenats repo to the fork remote\n")
	err = managedTenantsRepostiroy.Push(&git.PushOptions{
		RemoteName: "fork",
		Progress:   os.Stdout,
		Auth: &http.BasicAuth{
			Username: c.gitlabUsernameFlag,
			Password: c.gitlabTokenFlag,
		},
	})
	if err != nil {
		return e(err)
	}

	// Create the merge request
	targetProject, _, err := c.gitlabProjects.GetProject(c.managedTenantsOriginFlag, &gitlab.GetProjectOptions{})
	if err != nil {
		return e(err)
	}

	c.print.Print("create the MR to the managed-tenants origin\n")
	mr, _, err := c.gitlabMergeRequests.CreateMergeRequest(c.managedTenantsForkFlag, &gitlab.CreateMergeRequestOptions{
		Title:           gitlab.String(fmt.Sprintf(mergeRequestTitleTemplate, channel, version)),
		Description:     gitlab.String(c.mergeRequestDescriptionFlag),
		SourceBranch:    gitlab.String(managedTenantsBranch),
		TargetBranch:    gitlab.String(managedTenantsMasterBranch),
		TargetProjectID: gitlab.Int(targetProject.ID),
	})
	if err != nil {
		return e(err)
	}

	c.print.Printf("Merge request for version %s and environment %s created successfully\n", version, channel)
	c.print.Printf("MR: %s\n", mr.WebURL)

	// Reset the managed repostiroy to master
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(managedTenantsMasterBranch)})
	if err != nil {
		return e(err)
	}

	return nil
}

func (c *osdAddonReleaseCmd) run() error {

	version, err := utils.NewRHMIVersion(c.versionFlag)
	if err != nil {
		return err
	}

	// Clone the managed tenants
	managedTenatDirectory, err := ioutil.TempDir(os.TempDir(), "managed-tenants-")
	if err != nil {
		return err
	}

	c.print.Printf("clone the managed-tenants repo to %s\n", managedTenatDirectory)
	managedTenantsRepository, err := c.git.PlainClone(
		managedTenatDirectory,
		false,
		&git.CloneOptions{
			URL:           fmt.Sprintf("%s/%s", gitlabURL, c.managedTenantsOriginFlag),
			ReferenceName: plumbing.NewBranchReferenceName(managedTenantsMasterBranch),
		},
	)
	if err != nil {
		return err
	}
	// defer os.RemoveAll(managedTenatDirectory)

	// Add the fork remote to the managed-tenats repo
	_, err = managedTenantsRepository.CreateRemote(&config.RemoteConfig{
		Name: "fork",
		URLs: []string{fmt.Sprintf("%s/%s", gitlabURL, c.managedTenantsForkFlag)},
	})
	if err != nil {
		return err
	}

	// Clone the integreatly-operator
	integreatlyOperatorDirectory, err := ioutil.TempDir(os.TempDir(), "integreatly-operator-")
	if err != nil {
		return err
	}

	c.print.Printf("clone the integreatly-operator to %s\n", integreatlyOperatorDirectory)
	_, err = c.git.PlainClone(
		integreatlyOperatorDirectory,
		false,
		&git.CloneOptions{
			URL:           fmt.Sprintf("%s/%s", githubURL, c.integreatlyOperatorFlag),
			ReferenceName: plumbing.NewTagReferenceName(fmt.Sprintf("v%s", version)),
		},
	)
	if err != nil {
		return err
	}
	// defer os.RemoveAll(integreatlyOperatorDirectory)

	if version.IsPreRrelease() {

		// Release to stage
		err = c.createTheReleaseMergeRequest(
			integreatlyOperatorDirectory,
			managedTenatDirectory,
			managedTenantsRepository,
			version,
			stageChannel,
		)
		if err != nil {
			return err
		}

	} else {

		// When the version is not a prerelease version and is a final release
		// then create the release against stage, edge and stable
		for _, channel := range []releaseChannel{stageChannel, edgeChannel, stableChannel} {
			err = c.createTheReleaseMergeRequest(
				integreatlyOperatorDirectory,
				managedTenatDirectory,
				managedTenantsRepository,
				version,
				channel,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func init() {

	c := &osdAddonReleaseCmd{}

	cmd := &cobra.Command{
		Use:   "osd-addon-release",
		Short: "crete a release MR for the integreatly-operator to the managed-tenats repo",
		Run: func(cmd *cobra.Command, args []string) {

			// Prepare the GitLab Client
			gitlabClient, err := gitlab.NewClient(
				c.gitlabTokenFlag,
				gitlab.WithBaseURL(fmt.Sprintf("%s/%s", gitlabURL, gitlabAPIEndpoint)),
			)
			if err != nil {
				panic(err)
			}
			c.gitlabMergeRequests = gitlabClient.MergeRequests
			c.gitlabProjects = gitlabClient.Projects

			// Run
			err = c.run()
			if err != nil {
				panic(err)
			}
		},
	}

	rootCmd.AddCommand(cmd)

	cmd.Flags().StringVar(
		&c.versionFlag, "version", "",
		"the integreatly-operator version to push to the managed-tenats repo (ex: 2.0.0, 2.0.0-er4)")
	cmd.MarkFlagRequired("version")

	cmd.Flags().StringVar(&c.gitlabUsernameFlag, "gitlab-user", "", "the gitlab user for commiting the changes")
	cmd.MarkFlagRequired("gitlab-user")

	cmd.Flags().StringVar(&c.gitlabTokenFlag, "gitlab-token", "", "the gitlab token to commit the changes and open the MR")
	cmd.MarkFlagRequired("gitlab-token")

	cmd.Flags().StringVar(
		&c.mergeRequestDescriptionFlag,
		"merge-request-description",
		"",
		"an optional merge request description that can be used to notify secific users (ex \"ping: @dbizzarr\"",
	)

	cmd.Flags().StringVar(
		&c.managedTenantsOriginFlag,
		"managed-tenants-origin",
		"service/managed-tenants",
		"managed-tenants origin repository namespace and name")

	cmd.Flags().StringVar(
		&c.managedTenantsForkFlag,
		"managed-tenants-fork",
		"integreatly-qe/managed-tenants",
		"managed-tenants fork where to push the release files")

	cmd.Flags().StringVar(
		&c.integreatlyOperatorFlag,
		"integreatly-operator",
		"integr8ly/integreatly-operator.git",
		"integreatly operator branch where to take the release file")
}
