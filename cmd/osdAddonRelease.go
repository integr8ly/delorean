package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
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

	// The branch to target with the merge request
	managedTenantsMainBranch = "main"

	// Info for the commit and merge request
	branchNameTemplate               = "%s-%s-v%s"
	commitMessageTemplate            = "update %s %s to %s"
	commitAuthorName                 = "Delorean"
	commitAuthorEmail                = "cloud-services-delorean@redhat.com"
	mergeRequestTitleTemplate        = "Update %s %s to %s" // channel, version
	envVarNameUseClusterStorage      = "USE_CLUSTER_STORAGE"
	envVarNameAlertEmailAddress      = "ALERTING_EMAIL_ADDRESS"
	envVarNameAlertEmailAddressValue = "{{ alertingEmailAddress }}"
)

type releaseChannel struct {
	Name            string `json:"name"`
	Directory       string `json:"directory"`
	Environment     string `json:"environment"`
	AllowPreRelease bool   `json:"allow_pre_release"`
}

type addonCSVConfig struct {
	Repo string `json:"repo"`
	Path string `json:"path"`
}

type deploymentContainerEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type deploymentContainer struct {
	Name    string                      `json:"name"`
	EnvVars []deploymentContainerEnvVar `json:"env_vars"`
}

type deployment struct {
	Name      string              `json:"name"`
	Container deploymentContainer `json:"container"`
}

type override struct {
	Deployment deployment `json:"deployment"`
}

type addonConfig struct {
	Name     string           `json:"name"`
	CSV      addonCSVConfig   `json:"csv"`
	Channels []releaseChannel `json:"channels"`
	Override *override        `json:"override,omitempty"`
}

type addons struct {
	Addons []addonConfig `json:"addons"`
}

// directory returns the relative path of the managed-teneants repo to the
// addon for the given channel
func (c *releaseChannel) bundlesDirectory() string {
	return fmt.Sprintf("addons/%s/bundles", c.Directory)
}

func (c *releaseChannel) addonFile() string {
	return fmt.Sprintf("addons/%s/metadata/%s/addon.yaml", c.Directory, c.Environment)
}

type osdAddonReleaseFlags struct {
	version                 string
	channel                 string
	mergeRequestDescription string
	managedTenantsOrigin    string
	managedTenantsFork      string
	addonName               string
	addonsConfig            string
}

type osdAddonReleaseCmd struct {
	flags               *osdAddonReleaseFlags
	gitlabToken         string
	version             *utils.RHMIVersion
	gitlabMergeRequests services.GitLabMergeRequestsService
	gitlabProjects      services.GitLabProjectsService
	managedTenantsDir   string
	managedTenantsRepo  *git.Repository
	gitPushService      services.GitPushService
	addonConfig         *addonConfig
	currentChannel      *releaseChannel
	addonDir            string
}

type addon struct {
	content string
}

func (a *addon) setCurrentCSV(currentCSV string) {
	r := regexp.MustCompile(`currentCSV: .*`)
	s := r.ReplaceAllString(a.content, fmt.Sprintf("currentCSV: %s", currentCSV))
	a.content = s
}

func newAddon(addonPath string) (*addon, error) {
	c, err := ioutil.ReadFile(addonPath)
	if err != nil {
		return nil, err
	}
	return &addon{content: string(c)}, nil
}

func init() {

	f := &osdAddonReleaseFlags{}

	cmd := &cobra.Command{
		Use:   "osd-addon",
		Short: "Create a MR to the managed-tenants repo for the giving addon to update its version",
		Run: func(cmd *cobra.Command, args []string) {

			gitlabToken, err := requireValue(gitlabTokenKey)
			if err != nil {
				handleError(err)
			}

			// Prepare
			c, err := newOSDAddonReleaseCmd(f, gitlabToken)
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
	cmd.Flags().StringVar(&f.addonName, "name", "", "Name of the addon to update")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringVar(
		&f.version, "version", "",
		"The version to push to the managed-tenants repo (ex \"2.0.0\", \"2.0.0-er4\")")
	cmd.MarkFlagRequired("version")

	cmd.Flags().StringVar(&f.addonsConfig, "addons-config", "", "Configuration files for the addons")
	cmd.MarkFlagRequired("addons-config")

	cmd.Flags().StringVar(
		&f.channel, "channel", "stage",
		fmt.Sprintf("The OSD channel to which push the release. The channel values are defined in the addons-config file"),
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
		"managed-tenants origin repository from where to fork the main branch")

	cmd.Flags().StringVar(
		&f.managedTenantsFork,
		"managed-tenants-fork",
		"integreatly-qe/managed-tenants",
		"managed-tenants fork repository where to push the release files")
}

func findAddon(config *addons, addonName string) *addonConfig {
	var currentAddon *addonConfig
	for _, a := range config.Addons {
		v := a
		if a.Name == addonName {
			currentAddon = &v
			break
		}
	}
	return currentAddon
}

func findChannel(addon *addonConfig, channelName string) *releaseChannel {
	var currentChannel *releaseChannel
	for _, c := range addon.Channels {
		v := c
		if c.Name == channelName {
			currentChannel = &v
			break
		}
	}
	return currentChannel
}

func newOSDAddonReleaseCmd(flags *osdAddonReleaseFlags, gitlabToken string) (*osdAddonReleaseCmd, error) {
	version, err := utils.NewVersion(flags.version, olmType)
	if err != nil {
		return nil, err
	}
	addonsConfig := &addons{}
	if err := utils.PopulateObjectFromYAML(flags.addonsConfig, addonsConfig); err != nil {
		return nil, err
	}

	currentAddon := findAddon(addonsConfig, flags.addonName)
	if currentAddon == nil {
		return nil, fmt.Errorf("can not find configuration for addon %s in config file %s", flags.addonName, flags.addonsConfig)
	}

	currentChannel := findChannel(currentAddon, flags.channel)
	if currentChannel == nil {
		return nil, fmt.Errorf("can not find channel %s for addon %s in config file %s", flags.channel, flags.addonName, flags.addonsConfig)
	}

	fmt.Printf("create osd addon release for %s %s to the %s channel\n", flags.addonName, version.TagName(), flags.channel)

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
	// TODO: Move the clone functions inside the run() method to improve the test covered code
	managedTenantsDir, managedTenantsRepo, err := gitCloneService.CloneToTmpDir(
		"managed-tenants-",
		fmt.Sprintf("%s/%s", gitlabURL, flags.managedTenantsOrigin),
		plumbing.NewBranchReferenceName(managedTenantsMainBranch),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("managed-tenants repo cloned to %s\n", managedTenantsDir)

	// Add the fork remote to the managed-tenats repo
	_, err = managedTenantsRepo.CreateRemote(&config.RemoteConfig{
		Name: "fork",
		URLs: []string{fmt.Sprintf("%s/%s", gitlabURL, flags.managedTenantsFork)},
	})
	if err != nil {
		return nil, err
	}
	fmt.Print("added the fork remote to the managed-tenants repo\n")

	// Clone the repo to get the csv for the addon
	csvDir, _, err := gitCloneService.CloneToTmpDir(
		"addon-csv-",
		currentAddon.CSV.Repo,
		plumbing.NewTagReferenceName(version.TagName()),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("addon cloned to %s\n", csvDir)

	return &osdAddonReleaseCmd{
		flags:               flags,
		gitlabToken:         gitlabToken,
		version:             version,
		gitlabMergeRequests: gitlabClient.MergeRequests,
		gitlabProjects:      gitlabClient.Projects,
		managedTenantsDir:   managedTenantsDir,
		managedTenantsRepo:  managedTenantsRepo,
		gitPushService:      &services.DefaultGitPushService{},
		currentChannel:      currentChannel,
		addonConfig:         currentAddon,
		addonDir:            csvDir,
	}, nil
}

func (c *osdAddonReleaseCmd) run() error {
	if c.currentChannel == nil {
		return fmt.Errorf("currentChannel is not valid: %v", c.currentChannel)
	}
	if c.version.IsPreRelease() && !c.currentChannel.AllowPreRelease {
		return fmt.Errorf("the prerelease version %s can't be pushed to the %s channel", c.version, c.currentChannel.Name)
	}

	managedTenantsHead, err := c.managedTenantsRepo.Head()
	if err != nil {
		return err
	}

	// Verify that the repo is on master
	if managedTenantsHead.Name() != plumbing.NewBranchReferenceName(managedTenantsMainBranch) {
		return fmt.Errorf("the managed-tenants repo is pointing to %s instead of main", managedTenantsHead.Name())
	}

	managedTenantsTree, err := c.managedTenantsRepo.Worktree()
	if err != nil {
		return err
	}

	// Create a new branch on the managed-tenants repo
	managedTenantsBranch := fmt.Sprintf(branchNameTemplate, c.addonConfig.Name, c.currentChannel.Name, c.version)
	branchRef := plumbing.NewBranchReferenceName(managedTenantsBranch)

	fmt.Printf("create the branch %s in the managed-tenants repo\n", managedTenantsBranch)
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: true,
	})
	if err != nil {
		return err
	}

	// Copy the OLM manifests from the integreatly-operator repo to the the managed-tenats repo
	manifestsDirectory, err := c.copyTheOLMManifests()
	if err != nil {
		return err
	}

	// Add all changes
	err = managedTenantsTree.AddGlob(fmt.Sprintf("%s/*", manifestsDirectory))
	if err != nil {
		return err
	}

	// Update the addon.yaml file
	addonFile, err := c.updateTheAddonFile()
	if err != nil {
		return err
	}

	// Add the addon.yaml file
	_, err = managedTenantsTree.Add(addonFile)
	if err != nil {
		return err
	}

	//Update the integreatly-operator.vx.x.x.clusterserviceversion.yaml
	_, err = c.updateTheCSVManifest()
	if err != nil {
		return err
	}

	csvTemplate, err := c.renameCSVFile()
	if err != nil {
		return err
	}

	_, err = managedTenantsTree.Add(csvTemplate)
	if err != nil {
		return err
	}

	// Commit
	fmt.Print("commit all changes in the managed-tenants repo\n")
	_, err = managedTenantsTree.Commit(
		fmt.Sprintf(commitMessageTemplate, c.addonConfig.Name, c.currentChannel.Name, c.version),
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
	fmt.Printf("push the managed-tenants repo to the fork remote\n")
	err = c.gitPushService.Push(c.managedTenantsRepo, &git.PushOptions{
		RemoteName: "fork",
		Auth:       &http.BasicAuth{Password: c.gitlabToken},
		RefSpecs: []config.RefSpec{
			config.RefSpec(branchRef + ":" + branchRef),
		},
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
		Title:              gitlab.String(fmt.Sprintf(mergeRequestTitleTemplate, c.addonConfig.Name, c.currentChannel.Name, c.version)),
		Description:        gitlab.String(c.flags.mergeRequestDescription),
		SourceBranch:       gitlab.String(managedTenantsBranch),
		TargetBranch:       gitlab.String(managedTenantsMainBranch),
		TargetProjectID:    gitlab.Int(targetProject.ID),
		RemoveSourceBranch: gitlab.Bool(true),
	})
	if err != nil {
		return err
	}

	fmt.Printf("merge request for version %s and channel %s created successfully\n", c.version, c.currentChannel.Name)
	fmt.Printf("MR: %s\n", mr.WebURL)

	// Reset the managed repostiroy to master
	err = managedTenantsTree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(managedTenantsMainBranch)})
	if err != nil {
		return err
	}

	return nil
}

func (c *osdAddonReleaseCmd) copyTheOLMManifests() (string, error) {
	source := path.Join(c.addonDir, fmt.Sprintf("%s/%s", c.addonConfig.CSV.Path, c.version.Base()))

	relativeDestination := fmt.Sprintf("%s/%s", c.currentChannel.bundlesDirectory(), c.version.Base())
	destination := path.Join(c.managedTenantsDir, relativeDestination)

	fmt.Printf("copy files from %s to %s\n", source, destination)
	err := utils.CopyDirectory(source, destination)
	if err != nil {
		return "", err
	}

	return relativeDestination, nil
}

func (c *osdAddonReleaseCmd) updateTheAddonFile() (string, error) {
	relative := c.currentChannel.addonFile()
	addonFilePath := path.Join(c.managedTenantsDir, relative)

	fmt.Printf("update the currentCSV value in addon file %s to %s\n", relative, c.version)
	addon, err := newAddon(addonFilePath)
	if err != nil {
		return "", err
	}
	// Set currentCSV value
	addon.setCurrentCSV(fmt.Sprintf("%s.v%s", c.addonConfig.Name, c.version.Base()))

	err = ioutil.WriteFile(addonFilePath, []byte(addon.content), os.ModePerm)
	if err != nil {
		return "", err
	}

	return relative, nil
}

func (c *osdAddonReleaseCmd) updateTheCSVManifest() (string, error) {
	relative := fmt.Sprintf("%s/%s/%s.clusterserviceversion.yaml", c.currentChannel.bundlesDirectory(), c.version.Base(), c.addonConfig.Name)
	csvFile := path.Join(c.managedTenantsDir, relative)

	fmt.Printf("update csv manifest file %s\n", relative)
	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	err := utils.PopulateObjectFromYAML(csvFile, csv)
	if err != nil {
		return "", err
	}

	if c.addonConfig.Override != nil {
		_, deployment := utils.FindDeploymentByName(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs, c.addonConfig.Override.Deployment.Name)
		if deployment != nil {
			i, container := utils.FindContainerByName(deployment.Spec.Template.Spec.Containers, c.addonConfig.Override.Deployment.Container.Name)
			if container != nil {
				for _, envVar := range c.addonConfig.Override.Deployment.Container.EnvVars {
					container.Env = utils.AddOrUpdateEnvVar(container.Env, envVar.Name, envVar.Value)
				}
			}
			deployment.Spec.Template.Spec.Containers[i] = *container
		}
	}

	//Set SingleNamespace install mode to true
	mi, m := utils.FindInstallMode(csv.Spec.InstallModes, olmapiv1alpha1.InstallModeTypeSingleNamespace)
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

func (c *osdAddonReleaseCmd) renameCSVFile() (string, error) {
	o := fmt.Sprintf("%s/%s/%s.clusterserviceversion.yaml", c.currentChannel.bundlesDirectory(), c.version.Base(), c.addonConfig.Name)
	n := fmt.Sprintf("%s/%s/%s.v%s.clusterserviceversion.yaml.j2", c.currentChannel.bundlesDirectory(), c.version.Base(), c.addonConfig.Name, c.version.Base())
	fmt.Println(fmt.Sprintf("Rename file from %s to %s", o, n))
	oldPath := path.Join(c.managedTenantsDir, o)
	newPath := path.Join(c.managedTenantsDir, n)
	return n, os.Rename(oldPath, newPath)
}
