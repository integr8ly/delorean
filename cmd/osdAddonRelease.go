package cmd

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"path"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

const (
	gitlabTokenKey = "gitlab_token"

	// Base URL for gitlab API and for the managed-tenenats fork and origin repos
	gitlabURL = "https://gitlab.cee.redhat.com"

	gitlabAPIEndpoint = "api/v4"

	// Base URL for the integreatly-opeartor repo
	githubURL = "https://github.com"

	// Directory in the integreatly-opeartor repo with the OLM maninfest files
	sourceOLMManifestsDirectory = "deploy/olm-catalog/integreatly-operator/%s"

	// The branch to target with the merge request
	managedTenantsMasterBranch = "master"

	// Info for the commit and merge request
	branchNameTemplate           = "integreatly-operator-%s-v%s"
	commitMessageTemplate        = "update integreatly-operator %s to %s"
	commitAuthorName             = "Delorean"
	commitAuthorEmail            = "cloud-services-delorean@redhat.com"
	mergeRequestTitleTemplate    = "Update integreatly-operator %s to %s" // channel, version
	rhmiOperatorDeploymentName   = "rhmi-operator"
	rhmiOperatorContainerName    = "rhmi-operator"
	envVarNameUseClusterStorage  = "USE_CLUSTER_STORAGE"
	envVarNameAlerEmailAddress   = "ALERTING_EMAIL_ADDRESS"
	integreatlyAlertEmailAddress = "integreatly-notifications@redhat.com"
	cssreAlertEmailAddress       = "cssre-alerts@redhat.com"
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

type osdAddonReleaseFlags struct {
	version                 string
	channel                 string
	mergeRequestDescription string
	managedTenantsOrigin    string
	managedTenantsFork      string
}

type osdAddonReleaseCmd struct {
	flags                  *osdAddonReleaseFlags
	gitlabToken            string
	version                *utils.RHMIVersion
	channel                releaseChannel
	gitlabMergeRequests    services.GitLabMergeRequestsService
	gitlabProjects         services.GitLabProjectsService
	integreatlyOperatorDir string
	managedTenantsDir      string
	managedTenantsRepo     *git.Repository
	gitPushService         services.GitPushService
}

func init() {

	f := &osdAddonReleaseFlags{}

	cmd := &cobra.Command{
		Use:   "osd-addon",
		Short: "Create the integreatly-operator MR to the managed-tenants repo",
		Run: func(cmd *cobra.Command, args []string) {

			gitlabToken, err := requireValue(gitlabTokenKey)
			if err != nil {
				handleError(err)
			}

			// Prepare
			c, err := newOSDAddonReleseCmd(f, gitlabToken)
			if err != nil {
				handleError(err)
			}

			// Run
			err = c.run()
			if err != nil {
				handleError(err)
			}
		},
	}

	releaseCmd.AddCommand(cmd)

	cmd.Flags().StringVar(
		&f.version, "version", "",
		"The RHMI version to push to the managed-tenats repo (ex \"2.0.0\", \"2.0.0-er4\")")
	cmd.MarkFlagRequired("version")

	cmd.Flags().StringVar(
		&f.channel, "channel", string(stageChannel),
		fmt.Sprintf("The OSD channel to which push the RHMI release [%s|%s|%s]", stageChannel, edgeChannel, stableChannel),
	)

	cmd.Flags().String(
		"gitlab-token",
		"",
		"GitLab token to Push the changes and open the MR")
	viper.BindPFlag(gitlabTokenKey, cmd.Flags().Lookup("gitlab-token"))

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

func newOSDAddonReleseCmd(flags *osdAddonReleaseFlags, gitlabToken string) (*osdAddonReleaseCmd, error) {

	version, err := utils.NewRHMIVersion(flags.version)
	if err != nil {
		return nil, err
	}
	fmt.Printf("create osd addon release for RHMI v%s to the %s channel\n", version, flags.channel)

	// Prepare the GitLab Client
	gitlabClient, err := gitlab.NewClient(
		gitlabToken,
		gitlab.WithBaseURL(fmt.Sprintf("%s/%s", gitlabURL, gitlabAPIEndpoint)),
	)
	if err != nil {
		return nil, err
	}
	fmt.Print("gitlab client initialized and authenticated\n")

	gitCloneService := &services.DefaultGitCloneService{}
	// Clone the managed tenants
	// TODO: Move the clone functions inise the run() method to improve the test covered code
	managedTenatsDir, managedTenantsRepo, err := gitCloneService.CloneToTmpDir(
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
	integreatlyOperatorDir, _, err := gitCloneService.CloneToTmpDir(
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
		gitlabToken:            gitlabToken,
		version:                version,
		channel:                releaseChannel(flags.channel),
		gitlabMergeRequests:    gitlabClient.MergeRequests,
		gitlabProjects:         gitlabClient.Projects,
		integreatlyOperatorDir: integreatlyOperatorDir,
		managedTenantsDir:      managedTenatsDir,
		managedTenantsRepo:     managedTenantsRepo,
		gitPushService:         &services.DefaultGitPushService{},
	}, nil
}

func (c *osdAddonReleaseCmd) run() error {

	// verify that the passed version can be pushed to the passed channel
	switch c.channel {
	case stageChannel:
		// noting
	case stableChannel, edgeChannel:
		if c.version.IsPreRrelease() {
			return fmt.Errorf("the prerelease version %s can't be pushed to the %s channel", c.version, c.channel)
		}
	default:
		return fmt.Errorf("invalid channel %s, see the cmd help for the list of valid channels", c.channel)
	}

	managedTenantsHead, err := c.managedTenantsRepo.Head()
	if err != nil {
		return err
	}

	// Verify that the repo is on master
	if managedTenantsHead.Name() != plumbing.NewBranchReferenceName(managedTenantsMasterBranch) {
		return fmt.Errorf("the managed-tenants repo is pointing to %s insteand of master", managedTenantsHead.Name())
	}

	managedTenantsTree, err := c.managedTenantsRepo.Worktree()
	if err != nil {
		return err
	}

	// Create a new branch on the managed-tenants repo
	managedTenantsBranch := fmt.Sprintf(branchNameTemplate, c.channel, c.version)

	fmt.Printf("create the branch %s in the managed-tenants repo\n", managedTenantsBranch)
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(managedTenantsBranch),
		Create: true,
	})
	if err != nil {
		return err
	}

	// Copy the OLM manifests from the integreatly-operator repo to the the managed-tenats repo
	manifestsDirectory, err := c.copyTheOLMManifests(c.channel)
	if err != nil {
		return err
	}

	// Add all changes
	err = managedTenantsTree.AddGlob(fmt.Sprintf("%s/*", manifestsDirectory))
	if err != nil {
		return err
	}

	// Update the integreatly-operator.package.yaml
	packageManfiest, err := c.udpateThePackageManifest(c.channel)
	if err != nil {
		return err
	}

	// Add the integreatly-operator.package.yaml
	_, err = managedTenantsTree.Add(packageManfiest)
	if err != nil {
		return err
	}

	//Update the integreatly-operator.vx.x.x.clusterserviceversion.yaml
	csvManifest, err := c.udpateTheCSVManifest(c.channel)
	if err != nil {
		return err
	}
	_, err = managedTenantsTree.Add(csvManifest)
	if err != nil {
		return err
	}

	// Commit
	fmt.Print("commit all changes in the managed-tenats repo\n")
	_, err = managedTenantsTree.Commit(
		fmt.Sprintf(commitMessageTemplate, c.channel, c.version),
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
	err = c.gitPushService.Push(c.managedTenantsRepo, &git.PushOptions{
		RemoteName: "fork",
		Auth:       &http.BasicAuth{Password: c.gitlabToken},
	})
	if err != nil {
		return err
	}

	// Create the merge request
	targetProject, _, err := c.gitlabProjects.GetProject(c.flags.managedTenantsOrigin, &gitlab.GetProjectOptions{})
	if err != nil {
		return err
	}

	fmt.Print("create the MR to the managed-tenants origin\n")
	mr, _, err := c.gitlabMergeRequests.CreateMergeRequest(c.flags.managedTenantsFork, &gitlab.CreateMergeRequestOptions{
		Title:           gitlab.String(fmt.Sprintf(mergeRequestTitleTemplate, c.channel, c.version)),
		Description:     gitlab.String(c.flags.mergeRequestDescription),
		SourceBranch:    gitlab.String(managedTenantsBranch),
		TargetBranch:    gitlab.String(managedTenantsMasterBranch),
		TargetProjectID: gitlab.Int(targetProject.ID),
	})
	if err != nil {
		return err
	}

	fmt.Printf("merge request for version %s and channel %s created successfully\n", c.version, c.channel)
	fmt.Printf("MR: %s\n", mr.WebURL)

	// Reset the managed repostiroy to master
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(managedTenantsMasterBranch)})
	if err != nil {
		return err
	}

	return nil
}

func (c *osdAddonReleaseCmd) copyTheOLMManifests(channel releaseChannel) (string, error) {

	source := path.Join(c.integreatlyOperatorDir, fmt.Sprintf(sourceOLMManifestsDirectory, c.version.Base()))

	relativeDestination := fmt.Sprintf("%s/%s", channel.directory(), c.version.Base())
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
	p := &registry.PackageManifest{}
	err := utils.PopulateObjectFromYAML(manifest, p)
	if err != nil {
		return "", err
	}

	// Set channels[0].currentCSV value
	p.Channels[0].CurrentCSVName = fmt.Sprintf("integreatly-operator.v%s", c.version.Base())

	err = utils.WriteK8sObjectToYAML(p, manifest)
	if err != nil {
		return "", err
	}

	return relative, nil
}

func (c *osdAddonReleaseCmd) udpateTheCSVManifest(channel releaseChannel) (string, error) {
	relative := fmt.Sprintf("%s/%s/%s.v%s.clusterserviceversion.yaml", channel.directory(), c.version.Base(), "integreatly-operator", c.version.Base())
	csvFile := path.Join(c.managedTenantsDir, relative)

	fmt.Printf("update csv manifest file %s\n", relative)
	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	err := utils.PopulateObjectFromYAML(csvFile, csv)
	if err != nil {
		return "", err
	}

	_, deployment := findDeploymentByName(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs, rhmiOperatorDeploymentName)
	if deployment != nil {
		i, container := findContainerByName(deployment.Spec.Template.Spec.Containers, rhmiOperatorContainerName)
		if container != nil {
			// Update USE_CLUSTER_STORAGE env var to empty string
			container.Env = utils.AddOrUpdateEnvVar(container.Env, envVarNameUseClusterStorage, "")
			// Add ALERTING_EMAIL_ADDRESS env var
			if c.version.IsPreRrelease() {
				container.Env = utils.AddOrUpdateEnvVar(container.Env, envVarNameAlerEmailAddress, integreatlyAlertEmailAddress)
			} else {
				container.Env = utils.AddOrUpdateEnvVar(container.Env, envVarNameAlerEmailAddress, cssreAlertEmailAddress)
			}
		}
		deployment.Spec.Template.Spec.Containers[i] = *container
	}

	//Set SingleNamespace install mode to true
	mi, m := findInstallMode(csv.Spec.InstallModes, olmapiv1alpha1.InstallModeTypeSingleNamespace)
	if m != nil {
		m.Supported = true
	}
	csv.Spec.InstallModes[mi] = *m

	err = utils.WriteK8sObjectToYAML(csv, csvFile)
	if err != nil {
		return "", err
	}
	return relative, nil
}

func findDeploymentByName(deployments []olmapiv1alpha1.StrategyDeploymentSpec, name string) (int, *olmapiv1alpha1.StrategyDeploymentSpec) {
	for i, d := range deployments {
		if d.Name == name {
			return i, &d
		}
	}
	return -1, nil
}

func findContainerByName(containers []corev1.Container, containerName string) (int, *corev1.Container) {
	for i, c := range containers {
		if c.Name == containerName {
			return i, &c
		}
	}
	return -1, nil
}

func findInstallMode(installModes []olmapiv1alpha1.InstallMode, typeName olmapiv1alpha1.InstallModeType) (int, *olmapiv1alpha1.InstallMode) {
	for i, m := range installModes {
		if m.Type == typeName {
			return i, &m
		}
	}
	return -1, nil
}
