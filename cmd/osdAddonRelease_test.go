package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/types"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/xanzy/go-gitlab"
)

const (
	envVarNameUseClusterStorage = "USE_CLUSTER_STORAGE"
	envVarNameAlertEmailAddress = "ALERTING_EMAIL_ADDRESS"
)

type gitlabMergeRequestMock struct {
	createMergeRequest func(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
}

func (m *gitlabMergeRequestMock) CreateMergeRequest(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
	return m.createMergeRequest(pid, opt, options...)
}

type gitlabProjectsMock struct {
	getProject func(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

func (m *gitlabProjectsMock) GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	return m.getProject(pid, opt, options...)
}

func initRepoFromTestDir(prefix string, testDir string) (string, *git.Repository, error) {
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
	if err := checkoutBranch(tree, false, true, "main"); err != nil {
		return "", nil, err
	}
	return dir, repo, nil
}

func prepareManagedTenants(t *testing.T, basedir string, channel string) (string, *git.Repository) {
	repoPrefix := ""
	repoTestDir := ""
	if channel != "stable" {
		repoPrefix = "managed-tenants-bundles"
		repoTestDir = "testdata/osdAddonReleaseManagedTenantsBundles"
	} else {
		repoPrefix = "managed-tenants"
		repoTestDir = "testdata/osdAddonReleaseManagedTenants"
	}
	dir, repo, err := initRepoFromTestDir(repoPrefix, path.Join(basedir, repoTestDir))
	if err != nil {
		t.Fatal(err)
	}
	return dir, repo
}

func commitObject(t *testing.T, repo *git.Repository, ref string) *object.Commit {
	h, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		t.Fatal(err)
	}

	c, err := repo.CommitObject(*h)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func gitDiff(t *testing.T, repo *git.Repository, from, to string) *object.Patch {

	fromObject := commitObject(t, repo, from)
	toObject := commitObject(t, repo, to)

	patch, err := fromObject.Patch(toObject)
	if err != nil {
		t.Fatal(err)
	}

	return patch
}

func TestOSDAddonRelease(t *testing.T) {

	basedir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		version                     string
		olmType                     string
		channel                     string
		shouldHaveUseClusterStorage bool
		expectError                 bool
	}{
		{version: "1.1.0", olmType: "managed-api-service", channel: "stable", expectError: false},
		{version: "1.1.0-rc1", olmType: "managed-api-service", channel: "stage", expectError: false},
		{version: "1.1.0-rc1", olmType: "managed-api-service", channel: "some", expectError: true},
		{version: "1.1.0", olmType: "managed-api-service", channel: "edge", expectError: false},
		{version: "1.1.0", olmType: "managed-api-service", channel: "stage", expectError: false},
		{version: "1.1.0", olmType: "managed-api-service", channel: "some", expectError: true},
		{version: "1.1.0-rc1", olmType: "integreatly-operator", channel: "stage", expectError: false},
		{version: "1.1.0-rc1", olmType: "integreatly-operator", channel: "some", expectError: true},
		{version: "1.1.0", olmType: "integreatly-operator", channel: "stable", expectError: false},
		{version: "1.1.0", olmType: "integreatly-operator", channel: "stage", expectError: false},
		{version: "1.1.0", olmType: "integreatly-operator", channel: "some", expectError: true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("test create merge request for version %s and channel %s", c.version, c.channel), func(t *testing.T) {

			managedTenantsRepoPushed := false
			managedTenantsMergeRequestCreated := false

			var managedTenantsPatch *object.Patch

			flags := &osdAddonReleaseFlags{version: c.version, channel: c.channel, addonName: c.olmType}

			// Prepare the version
			version, err := utils.NewVersion(flags.version, c.olmType)
			if err != nil {
				t.Fatal(err)
			}

			// Prepare the integreatly-operator directory
			integreatlyOperatorDir := path.Join(basedir, fmt.Sprintf("testdata/osdAddonReleaseIntegreatlyOperator%s", version))

			// Prepare the managed-tenants repo and dir
			managedTenantsDir, managedTenantsRepo := prepareManagedTenants(t, basedir, c.channel)
			var managedTenantsMainBranch string = "main"
			var managedTenantsRef plumbing.ReferenceName = "refs/heads/main"

			// Mock the push service
			mockPushService := &mockGitPushService{pushFunc: func(gitRepo *git.Repository, opts *git.PushOptions) error {
				// Save the last commit diff before HEAD get reset to master
				managedTenantsPatch = gitDiff(t, managedTenantsRepo, managedTenantsMainBranch, "HEAD")

				managedTenantsRepoPushed = true
				return nil
			}}

			// Mock the gitlab api
			gitlabProjectsMock := &gitlabProjectsMock{
				getProject: func(
					_ interface{},
					_ *gitlab.GetProjectOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.Project, *gitlab.Response, error) {
					return &gitlab.Project{}, &gitlab.Response{}, nil
				},
			}
			gitlabMergeRequestMock := &gitlabMergeRequestMock{
				createMergeRequest: func(
					_ interface{},
					_ *gitlab.CreateMergeRequestOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					managedTenantsMergeRequestCreated = true
					return &gitlab.MergeRequest{}, &gitlab.Response{}, nil
				},
			}

			addonsConfig := &addons{}
			var configFile string
			switch c.olmType {
			case types.OlmTypeRhmi:
				configFile = "managed-tenants-addons-config.yaml"
			case types.OlmTypeRhoam:
				configFile = "managed-tenants-addons-config-rhoam.yaml"
			}
			if err := utils.PopulateObjectFromYAML(fmt.Sprintf("../configurations/%s", configFile), addonsConfig); err != nil {
				t.Fatalf("failed to load addon config file")
			}

			currentAddon := findAddon(addonsConfig, c.olmType)
			currentChannel := findChannel(currentAddon, c.channel)

			// Create the osdAddonReleaseCmd object
			cmd := &osdAddonReleaseCmd{
				flags:               flags,
				version:             version,
				currentChannel:      currentChannel,
				gitlabMergeRequests: gitlabMergeRequestMock,
				gitlabProjects:      gitlabProjectsMock,
				addonDir:            integreatlyOperatorDir,
				managedTenantsDir:   managedTenantsDir,
				managedTenantsRepo:  managedTenantsRepo,
				gitPushService:      mockPushService,
				addonConfig:         currentAddon,
			}

			// Run the osdAddonReleaseCmd
			err = cmd.run()

			if c.expectError {
				if err != nil {
					// Test Succeded
					return
				}

				t.Fatalf("expected osdAddonReleaseCmd.run to fails but it succed")
			}

			if err != nil {
				t.Fatalf("osdAddonReleaseCmd.run failed with error: %s", err)
			}

			// Verify the managed-tenants push has been call
			if !managedTenantsRepoPushed {
				t.Fatal("the managed-tenants repo hasn't been pushed")
			}

			// Verify the gitlab create merge request endpoint has been call
			if !managedTenantsMergeRequestCreated {
				t.Fatal("the managed-tenants repo hasn't been created")
			}

			// Verify the repo is clean
			tree, err := managedTenantsRepo.Worktree()
			if err != nil {
				t.Fatal(err)
			}
			s, err := tree.Status()
			if err != nil {
				t.Fatal(err)
			}
			if len(s) != 0 {
				t.Fatalf("the managed-tenants repo is not clean: %s", s)
			}

			// Verify the committed changes
			patches := managedTenantsPatch.FilePatches()

			switch c.channel {
			case "stable":
				switch c.olmType {
				case types.OlmTypeRhmi:
					if found := len(patches); found != 1 {
						t.Fatalf("expected 1 but found %d changed/added files", found)
					}
				case types.OlmTypeRhoam:
					if found := len(patches); found != 1 {
						t.Fatalf("expected 1 but found %d changed/added files", found)
					}
				}
			case "edge":
				switch c.olmType {
				case types.OlmTypeRhmi:
					if found := len(patches); found != 3 {
						t.Fatalf("expected 3 but found %d changed/added files", found)
					}
				case types.OlmTypeRhoam:
					if found := len(patches); found != 3 {
						t.Fatalf("expected 3 but found %d changed/added files", found)
					}
				}
			case "stage":
				switch c.olmType {
				case types.OlmTypeRhmi:
					if found := len(patches); found != 3 {
						t.Fatalf("expected 3 but found %d changed/added files", found)
					}
				case types.OlmTypeRhoam:
					if found := len(patches); found != 3 {
						t.Fatalf("expected 3 but found %d changed/added files", found)
					}
				}
			}

			fmt.Println(currentChannel.bundlesDirectory())
			fmt.Println(version.Base())
			fmt.Println(c.olmType)
			fmt.Println(version.Base())
			clusterServiceVersion := fmt.Sprintf("%s/%s/manifests/%s.clusterserviceversion.yaml", currentChannel.bundlesDirectory(), version.Base(), c.olmType)
			customResourceDefinition := fmt.Sprintf("%s/%s/manifests/integreatly.org_rhmis_crd.yaml", currentChannel.bundlesDirectory(), version.Base())
			annotationsFile := fmt.Sprintf("%s/%s/metadata/annotations.yaml", currentChannel.bundlesDirectory(), version.Base())
			addonImageSetFile := cmd.getDestAddonImageSetPath()

			for _, p := range patches {
				_, file := p.Files()
				switch file.Path() {
				case annotationsFile:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, annotationsFile)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the first and only chunk type should be Add but found %d for %s", found, annotationsFile)
					}
				case clusterServiceVersion:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, clusterServiceVersion)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the first and only chunk type should be Add but found %d for %s", found, clusterServiceVersion)
					}
					content := p.Chunks()[0].Content()
					if found := len(content); found <= 0 {
						t.Fatalf("expected %s to be larger than 0 but found %d", clusterServiceVersion, found)
					}
					csv := &olmapiv1alpha1.ClusterServiceVersion{}
					err := yaml.Unmarshal([]byte(content), csv)
					if err != nil {
						t.Fatalf("invalid CSV file content:\n%s", content)
					}
					_, deployment := utils.FindDeploymentByName(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs, "rhmi-operator")
					if deployment == nil {
						t.Fatalf("can not find rhmi-operator deployment spec in csv file:\n%s", content)
					}
					_, container := utils.FindContainerByName(deployment.Spec.Template.Spec.Containers, "rhmi-operator")
					if container == nil {
						t.Fatalf("can not find rhmi-operator container spec in csv file:\n%s", content)
					}
					if containerEnvFound := len(container.Env); containerEnvFound != 5 {
						t.Fatalf("expected 5 envars to be found but found %v", len(container.Env))
					}

					switch c.olmType {
					case "managed-api-service":
						switch c.channel {
						case "edge":
							if csv.Name != "managed-api-service-internal.v1.1.0" {
								t.Fatalf("CSV name for edge has not been updated correctly, expected managed-api-service-internal.v1.1.0; got: %v", csv.Name)
							}
							if csv.Spec.Replaces != "managed-api-service-internal.v1.0.1" {
								t.Fatalf("CSV replaces for edge has not been updated correctly, expected managed-api-service-internal.v1.0.1; got: %v", csv.Spec.Replaces)
							}
						case "stage":
							if csv.Name != "managed-api-service.v1.1.0" {
								t.Fatalf("CSV name for edge has not been updated correctly, expected managed-api-service-internal.v1.1.0; got: %v", csv.Name)
							}
							if csv.Spec.Replaces != "managed-api-service.v1.0.1" {
								t.Fatalf("CSV replaces for edge has not been updated correctly, expected managed-api-service-internal.v1.0.1; got: %v", csv.Spec.Replaces)
							}
						case "stable":
							if csv.Name != "managed-api-service.v1.1.0" {
								t.Fatalf("CSV name for edge has not been updated correctly, expected managed-api-service-internal.v1.1.0; got: %v", csv.Name)
							}
							if csv.Spec.Replaces != "managed-api-service.v1.0.1" {
								t.Fatalf("CSV replaces for edge has not been updated correctly, expected managed-api-service-internal.v1.0.1; got: %v", csv.Spec.Replaces)
							}
						default:
							t.Fatalf("unexpected channel %s", c.channel)
						}
					case "integreatly-operator":
						if csv.Name != "integreatly-operator.v1.1.0" {
							t.Fatalf("CSV name for edge has not been updated correctly, expected integreatly-operator.v1.1.0; got: %v", csv.Name)
						}
						if csv.Spec.Replaces != "integreatly-operator.v1.0.1" {
							t.Fatalf("CSV replaces for edge has not been updated correctly, expected integreatly-operator.v1.0.1; got: %v", csv.Spec.Replaces)
						}
					default:
						t.Fatalf("unexpected olmType %s", c.olmType)
					}

					storageEnvVarValueFound, alertEnvVarFound := false, false
					for _, env := range container.Env {
						if env.Name == envVarNameUseClusterStorage {
							storageEnvVarValueFound = true
						}
						if env.Name == envVarNameAlertEmailAddress {
							alertEnvVarFound = true
						}
					}
					if storageEnvVarValueFound && c.olmType == types.OlmTypeRhmi {
						t.Fatalf("%s env var should not be present in the CSV bundle:\n%s", envVarNameUseClusterStorage, content)
					}
					if alertEnvVarFound {
						t.Fatalf("%s env var should not be present in the CSV bundle:\n%s", envVarNameAlertEmailAddress, content)
					}
					_, installMode := utils.FindInstallMode(csv.Spec.InstallModes, olmapiv1alpha1.InstallModeTypeSingleNamespace)
					if !installMode.Supported {
						t.Fatalf("%s value should be true in csv file:\n%s", olmapiv1alpha1.InstallModeTypeSingleNamespace, content)
					}

				case customResourceDefinition:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, customResourceDefinition)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the first and only chunk type should be Add but found %d for %s", found, customResourceDefinition)
					}
					if found := len(p.Chunks()[0].Content()); found <= 0 {
						t.Fatalf("expected %s to be larger than 0 but found %d", customResourceDefinition, found)
					}
				case addonImageSetFile:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, addonImageSetFile)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the first and only chunk type should be Add but found %d for %s", found, addonImageSetFile)
					}
					if found := len(p.Chunks()[0].Content()); found <= 0 {
						t.Fatalf("expected %s to be larger than 0 but found %d", addonImageSetFile, found)
					}
				default:
					t.Fatalf("unexpected file %s", file.Path())
				}
			}

			// Verify the manage-tenents repo HEAD is pointing to main
			head, err := managedTenantsRepo.Head()
			if err != nil {
				t.Fatal(err)
			}

			if founded := head.Name(); founded != managedTenantsRef {
				t.Fatalf("the managed-tenants repo HEAD doesn't point to the main branch\nexpected: refs/heads/main\nfounded: %s", founded)
			}
		})
	}
}

func Test_osdAddonReleaseCmd_getLatestStageAddonImageSet(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "test latest addon image set in stage is returned",
			fields: fields{
				managedTenantsDir: "testdata/osdAddonReleaseManagedTenants",
				currentChannel: &releaseChannel{
					Directory: "rhoams",
				},
			},
			want: "testdata/osdAddonReleaseManagedTenants/addons/rhoams/addonimagesets/stage/rhoams.v1.27.5-1.27.0.yaml",
		},
		{
			name: "test error reading directory",
			fields: fields{
				managedTenantsDir: "testdata/osdAddonReleaseManagedTenants",
				currentChannel: &releaseChannel{
					Directory: "nonExistent",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &osdAddonReleaseCmd{
				flags:               tt.fields.flags,
				gitlabToken:         tt.fields.gitlabToken,
				version:             tt.fields.version,
				gitlabMergeRequests: tt.fields.gitlabMergeRequests,
				gitlabProjects:      tt.fields.gitlabProjects,
				managedTenantsDir:   tt.fields.managedTenantsDir,
				managedTenantsRepo:  tt.fields.managedTenantsRepo,
				gitPushService:      tt.fields.gitPushService,
				addonConfig:         tt.fields.addonConfig,
				currentChannel:      tt.fields.currentChannel,
				addonDir:            tt.fields.addonDir,
			}
			got, err := c.getLatestStageAddonImageSetPath()
			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestStageAddonImageSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getLatestStageAddonImageSet() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_osdAddonReleaseCmd_getDestAddonImageSetPath(t *testing.T) {
	version, err := utils.NewVersion("1.27.0", types.OlmTypeRhoam)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test destination path correctly returned",
			fields: fields{
				managedTenantsDir: "testdata/osdAddonReleaseManagedTenants",
				currentChannel: &releaseChannel{
					Directory:   "rhoams",
					Environment: "production",
				},
				version: version,
			},
			want: "addons/rhoams/addonimagesets/production/rhoams.v1.27.0.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &osdAddonReleaseCmd{
				flags:               tt.fields.flags,
				gitlabToken:         tt.fields.gitlabToken,
				version:             tt.fields.version,
				gitlabMergeRequests: tt.fields.gitlabMergeRequests,
				gitlabProjects:      tt.fields.gitlabProjects,
				managedTenantsDir:   tt.fields.managedTenantsDir,
				managedTenantsRepo:  tt.fields.managedTenantsRepo,
				gitPushService:      tt.fields.gitPushService,
				addonConfig:         tt.fields.addonConfig,
				currentChannel:      tt.fields.currentChannel,
				addonDir:            tt.fields.addonDir,
			}
			if got := c.getDestAddonImageSetPath(); got != tt.want {
				t.Errorf("getDestAddonImageSetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_osdAddonReleaseCmd_getAddonImageSetName(t *testing.T) {
	version, err := utils.NewVersion("1.27.0", types.OlmTypeRhoam)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test correct name is returned",
			fields: fields{
				currentChannel: &releaseChannel{
					Directory: "rhoams",
				},
				version: version,
			},
			want: "rhoams.v1.27.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &osdAddonReleaseCmd{
				flags:               tt.fields.flags,
				gitlabToken:         tt.fields.gitlabToken,
				version:             tt.fields.version,
				gitlabMergeRequests: tt.fields.gitlabMergeRequests,
				gitlabProjects:      tt.fields.gitlabProjects,
				managedTenantsDir:   tt.fields.managedTenantsDir,
				managedTenantsRepo:  tt.fields.managedTenantsRepo,
				gitPushService:      tt.fields.gitPushService,
				addonConfig:         tt.fields.addonConfig,
				currentChannel:      tt.fields.currentChannel,
				addonDir:            tt.fields.addonDir,
			}
			if got := c.getAddonImageSetName(); got != tt.want {
				t.Errorf("getAddonImageSetName() = %v, want %v", got, tt.want)
			}
		})
	}
}
