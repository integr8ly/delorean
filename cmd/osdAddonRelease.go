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
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/integr8ly/delorean/pkg/version"
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

var (
	versionFlag                 string
	gitlabUsernameFlag          string
	gitlabTokenFlag             string
	mergeRequestDescriptionFlag string
	managedTenantsOriginFlag    string
	managedTenantsForkFlag      string
	integreatlyOperatorFlag     string
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

func copyTheOLMManifests(
	managedTenantsDirectory, integreatlyOperatorDirectory string,
	channel releaseChannel, version *version.RHMIVersion) (string, error) {

	source := path.Join(integreatlyOperatorDirectory, fmt.Sprintf(sourceOLMManifestsDirectory, version))

	relativeDestination := fmt.Sprintf("%s/%s", channel.directory(), version.String())
	destination := path.Join(managedTenantsDirectory, relativeDestination)

	fmt.Printf("copy files from %s to %s\n", source, destination)
	err := utils.CopyDirectory(source, destination)
	if err != nil {
		return "", err
	}

	return relativeDestination, nil
}

func udpateThePackageManifest(
	managedTenantsDirectory string,
	channel releaseChannel,
	version *version.RHMIVersion) (string, error) {

	relative := fmt.Sprintf("%s/%s.package.yaml", channel.directory(), channel.operatorName())
	manifest := path.Join(managedTenantsDirectory, relative)

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

func createTheReleaseMergeRequest(
	integreatlyOperatorDirectory string,
	managedTenantsDirectory string,
	managedTenantsRepostiroy *git.Repository,
	version *version.RHMIVersion,
	channel releaseChannel) error {

	managedTenantsHead, err := managedTenantsRepostiroy.Head()
	if err != nil {
		return err
	}

	// Verify that the repo is on master
	if managedTenantsHead.Name() != plumbing.NewBranchReferenceName(managedTenantsMasterBranch) {
		return fmt.Errorf("the managed-tenants repo is pointing to %s insteand of master", managedTenantsHead.Name())
	}

	managedTenantsTree, err := managedTenantsRepostiroy.Worktree()
	if err != nil {
		return err
	}

	// Create a new branch on the managed-tenants repo
	managedTenantsBranch := fmt.Sprintf(branchNameTemplate, channel, version)

	fmt.Printf("create the branch %s in the managed-tenants repo\n", managedTenantsBranch)
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(managedTenantsBranch),
		Create: true,
	})
	if err != nil {
		return err
	}

	// Copy the OLM manifests from the integreatly-operator repo to the the managed-tenats repo
	manifestsDirectory, err := copyTheOLMManifests(managedTenantsDirectory, integreatlyOperatorDirectory, channel, version)
	if err != nil {
		return err
	}

	// Add all changes
	err = managedTenantsTree.AddGlob(fmt.Sprintf("%s/*", manifestsDirectory))
	if err != nil {
		return err
	}

	// Update the integreatly-operator.package.yaml
	packageManfiest, err := udpateThePackageManifest(managedTenantsDirectory, channel, version)
	if err != nil {
		return err
	}

	// Add the integreatly-operator.package.yaml
	_, err = managedTenantsTree.Add(packageManfiest)
	if err != nil {
		return err
	}

	// Commit
	fmt.Print("commit all changes in the managed-tenats repo\n")
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
		return err
	}

	// Verify tha the tree is clean
	status, err := managedTenantsTree.Status()
	if err != nil {
		return err
	}

	if len(status) != 0 {
		return fmt.Errorf("the tree is not clean, uncommited changes:\n%+v", status)
	}

	// Push to fork
	fmt.Printf("push the managed-tenats repo to the fork remote\n")
	err = managedTenantsRepostiroy.Push(&git.PushOptions{
		RemoteName: "fork",
		Progress:   os.Stdout,
		Auth: &http.BasicAuth{
			Username: gitlabUsernameFlag,
			Password: gitlabTokenFlag,
		},
	})
	if err != nil {
		return err
	}

	// Create the merge request
	gitlabClient, err := gitlab.NewClient(
		gitlabTokenFlag,
		gitlab.WithBaseURL(fmt.Sprintf("%s/%s", gitlabURL, gitlabAPIEndpoint)),
	)
	if err != nil {
		return err
	}

	project, _, err := gitlabClient.Projects.GetProject(managedTenantsOriginFlag, &gitlab.GetProjectOptions{})
	if err != nil {
		return err
	}

	fmt.Print("create the MR to the managed-tenants origin\n")
	mr, _, err := gitlabClient.MergeRequests.CreateMergeRequest(managedTenantsForkFlag, &gitlab.CreateMergeRequestOptions{
		Title:           gitlab.String(fmt.Sprintf(mergeRequestTitleTemplate, channel, version)),
		Description:     gitlab.String(mergeRequestDescriptionFlag),
		SourceBranch:    gitlab.String(managedTenantsBranch),
		TargetBranch:    gitlab.String(managedTenantsMasterBranch),
		TargetProjectID: gitlab.Int(project.ID),
	})
	if err != nil {
		return err
	}

	fmt.Printf("Merge request for version %s and environment %s created successfully\n", version, channel)
	fmt.Printf("MR: %s\n", mr.WebURL)

	// Reset the managed repostiroy to master
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(managedTenantsMasterBranch)})
	if err != nil {
		return err
	}

	return nil
}

// processCSVImagesCmd represents the processCSVImages command
var osdAddonReleaseCmd = &cobra.Command{
	Use:   "osd-addon-release",
	Short: "crete a release MR for the integreatly-operator to the managed-tenats repo",
	Run: func(cmd *cobra.Command, args []string) {

		version, err := version.NewRHMIVersion(versionFlag)
		if err != nil {
			panic(err)
		}

		// Clone the managed tenants
		managedTenatDirectory, err := ioutil.TempDir(os.TempDir(), "managed-tenants-")
		if err != nil {
			panic(err)
		}

		cmd.Printf("clone the managed-tenants repo to %s\n", managedTenatDirectory)
		managedTenantsRepository, err := git.PlainClone(
			managedTenatDirectory,
			false,
			&git.CloneOptions{
				URL: fmt.Sprintf("%s/%s", gitlabURL, managedTenantsOriginFlag),
				// Progress: os.Stdout,
				ReferenceName: plumbing.NewBranchReferenceName(managedTenantsMasterBranch),
			},
		)
		if err != nil {
			panic(err)
		}
		// defer os.RemoveAll(managedTenatDirectory)

		// Add the fork remote to the managed-tenats repo
		_, err = managedTenantsRepository.CreateRemote(&config.RemoteConfig{
			Name: "fork",
			URLs: []string{fmt.Sprintf("%s/%s", gitlabURL, managedTenantsForkFlag)},
		})
		if err != nil {
			panic(err)
		}

		// Clone the integreatly-operator
		integreatlyOperatorDirectory, err := ioutil.TempDir(os.TempDir(), "integreatly-operator-")
		if err != nil {
			panic(err)
		}

		cmd.Printf("clone the integreatly-operator to %s\n", integreatlyOperatorDirectory)
		_, err = git.PlainClone(integreatlyOperatorDirectory, false, &git.CloneOptions{
			URL: fmt.Sprintf("%s/%s", githubURL, integreatlyOperatorFlag),
			// Progress:      os.Stdout,
			ReferenceName: plumbing.NewTagReferenceName(fmt.Sprintf("v%s", version)),
		})
		if err != nil {
			panic(err)
		}
		// defer os.RemoveAll(integreatlyOperatorDirectory)

		if version.IsPreRrelease() {

			// Release to stage
			err = createTheReleaseMergeRequest(
				integreatlyOperatorDirectory,
				managedTenatDirectory,
				managedTenantsRepository,
				version,
				stageChannel,
			)
			if err != nil {
				panic(err)
			}

		} else {

			// When the version is not a prerelease version and is a final release
			// then create the release against stage, edge and stable
			err = createTheReleaseMergeRequest(
				integreatlyOperatorDirectory,
				managedTenatDirectory,
				managedTenantsRepository,
				version,
				stageChannel,
			)
			if err != nil {
				panic(err)
			}

			err = createTheReleaseMergeRequest(
				integreatlyOperatorDirectory,
				managedTenatDirectory,
				managedTenantsRepository,
				version,
				edgeChannel,
			)
			if err != nil {
				panic(err)
			}

			err = createTheReleaseMergeRequest(
				integreatlyOperatorDirectory,
				managedTenatDirectory,
				managedTenantsRepository,
				version,
				stableChannel,
			)
			if err != nil {
				panic(err)
			}

		}
	},
}

func init() {
	rootCmd.AddCommand(osdAddonReleaseCmd)

	osdAddonReleaseCmd.Flags().StringVar(
		&versionFlag, "version", "",
		"the integreatly-operator version to push to the managed-tenats repo (ex: 2.0.0, 2.0.0-er4)")
	osdAddonReleaseCmd.MarkFlagRequired("version")

	osdAddonReleaseCmd.Flags().StringVar(&gitlabUsernameFlag, "gitlab-user", "", "the gitlab user for commiting the changes")
	osdAddonReleaseCmd.MarkFlagRequired("gitlab-user")

	osdAddonReleaseCmd.Flags().StringVar(&gitlabTokenFlag, "gitlab-token", "", "the gitlab token to commit the changes and open the MR")
	osdAddonReleaseCmd.MarkFlagRequired("gitlab-token")

	osdAddonReleaseCmd.Flags().StringVar(
		&mergeRequestDescriptionFlag,
		"merge-request-description",
		"",
		"an optional merge request description that can be used to notify secific users (ex \"ping: @dbizzarr\"",
	)

	osdAddonReleaseCmd.Flags().StringVar(
		&managedTenantsOriginFlag,
		"managed-tenants-origin",
		"service/managed-tenants",
		"managed-tenants origin repository namespace and name")

	osdAddonReleaseCmd.Flags().StringVar(
		&managedTenantsForkFlag,
		"managed-tenants-fork",
		"integreatly-qe/managed-tenants",
		"managed-tenants fork where to push the release files")

	osdAddonReleaseCmd.Flags().StringVar(
		&integreatlyOperatorFlag,
		"integreatly-operator",
		"integr8ly/integreatly-operator.git",
		"integreatly operator branch where to take the release file")
}