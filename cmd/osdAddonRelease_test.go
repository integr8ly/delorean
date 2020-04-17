package cmd

import (
	"fmt"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/xanzy/go-gitlab"
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

type gitRepositoryMock struct {
	repository *git.Repository
	push       func(o *git.PushOptions) error
}

func (m *gitRepositoryMock) Head() (*plumbing.Reference, error) {
	return m.repository.Head()
}

func (m *gitRepositoryMock) Worktree() (*git.Worktree, error) {
	return m.repository.Worktree()
}

func (m *gitRepositoryMock) Push(o *git.PushOptions) error {
	return m.push(o)
}

func prepareManagedTenants(t *testing.T, basedir string) (string, *git.Repository) {

	dir, err := ioutil.TempDir(os.TempDir(), "managed-tenants-")
	if err != nil {
		t.Fatal(err)
	}

	err = utils.CopyDirectory(path.Join(basedir, "testdata/osdAddonReleaseManagedTenants"), dir)
	if err != nil {
		t.Fatal(err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	err = tree.AddGlob(".")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tree.Commit("fake", &git.CommitOptions{
		Author: &object.Signature{},
	})
	if err != nil {
		t.Fatal(err)
	}

	return dir, repo
}

func prepareTestEnviornment() {

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

func readFile(t *testing.T, file string) []byte {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func TestCreateReleaseMergeRequest(t *testing.T) {

	basedir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		version string
		channel releaseChannel
	}{
		{version: "2.1.0-rc1", channel: stageChannel},
		{version: "2.1.0", channel: stageChannel},
		{version: "2.1.0", channel: edgeChannel},
		{version: "2.1.0", channel: stableChannel},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("test create merge request for version %s and channel %s", c.version, c.channel), func(t *testing.T) {

			managedTenantsRepoPushed := false
			managedTenantsMergeRequestCreated := false

			var managedTenantsPatch *object.Patch

			flags := &osdAddonReleaseFlags{version: c.version}

			// Prepare the version
			version, err := utils.NewRHMIVersion(flags.version)
			if err != nil {
				t.Fatal(err)
			}

			// Prepare the integreatly-operator directory
			integreatlyOperatorDir := path.Join(basedir, fmt.Sprintf("testdata/osdAddonReleaseIntegreatlyOperator%s", version))

			// Prepare the managed-teneants repo and dir
			managedTenantsDir, managedTenantsRepo := prepareManagedTenants(t, basedir)

			// Mock the managed-tenants repo
			managedTenantsRepoMock := &gitRepositoryMock{
				repository: managedTenantsRepo,
				push: func(o *git.PushOptions) error {

					// Save the last commit diff before HEAD get reset to master
					managedTenantsPatch = gitDiff(t, managedTenantsRepo, "master", "HEAD")

					managedTenantsRepoPushed = true
					return nil
				},
			}

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

			// Create the osdAddonReleaseCmd object
			cmd := &osdAddonReleaseCmd{
				flags:                  flags,
				version:                version,
				gitlabMergeRequests:    gitlabMergeRequestMock,
				gitlabProjects:         gitlabProjectsMock,
				integreatlyOperatorDir: integreatlyOperatorDir,
				managedTenantsDir:      managedTenantsDir,
				managedTenantsRepo:     managedTenantsRepoMock,
			}

			// Run the osdAddonReleaseCmd
			err = cmd.createReleaseMergeRequest(c.channel)
			if err != nil {
				t.Fatalf("createReleaseMergeRequest failed with error: %s", err)
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

			// Verify the commited changes
			patches := managedTenantsPatch.FilePatches()

			if found := len(patches); found != 3 {
				t.Fatalf("expected 3 but found %d changed/added files", found)
			}

			packageManifest := fmt.Sprintf("%s/%s.package.yaml", c.channel.directory(), c.channel.operatorName())
			clusterServiceVersion := fmt.Sprintf("%s/%s/integreatly-operator.v%s.clusterserviceversion.yaml", c.channel.directory(), version.Base(), version.Base())
			customResourceDefinition := fmt.Sprintf("%s/%s/integreatly.org_rhmis_crd.yaml", c.channel.directory(), version.Base())

			for _, p := range patches {
				_, file := p.Files()
				switch file.Path() {
				case packageManifest:
					if found := len(p.Chunks()); found != 4 {
						t.Fatalf("expected 4 but found %d chunk changes for %s", found, packageManifest)
					}
					expected := fmt.Sprintf("currentCSV: integreatly-operator.v%s\n", version.Base())
					chunks := p.Chunks()
					found := false
					for _, c := range chunks {
						if strings.Index(c.Content(), expected) > -1 {
							found = true
						}
					}
					if !found {
						t.Fatalf("can not find expected change: %s", expected)
					}

				case clusterServiceVersion:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, clusterServiceVersion)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the frist and only chunk type should be Add but found %d for %s", found, clusterServiceVersion)
					}
					content := p.Chunks()[0].Content()
					if found := len(content); found <= 0 {
						t.Fatalf("expected %s to be largern then 0 but found %d", clusterServiceVersion, found)
					}
					csv := &olmapiv1alpha1.ClusterServiceVersion{}
					err := yaml.Unmarshal([]byte(content), csv)
					if err != nil {
						t.Fatalf("invalid CSV file content:\n%s", content)
					}
					_, deployment := findDeploymentByName(csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs, "rhmi-operator")
					if deployment == nil {
						t.Fatalf("can not find rhmi-operator deployment spec in csv file:\n%s", content)
					}
					_, container := findContainerByName(deployment.Spec.Template.Spec.Containers, "rhmi-operator")
					if container == nil {
						t.Fatalf("can not find rhmi-operator container spec in csv file:\n%s", content)
					}
					storageEnvVarChecked, alertEnvVarChecked := false, false
					for _, env := range container.Env {
						if env.Name == envVarNameUseClusterStorage && env.Value == "" {
							storageEnvVarChecked = true
						}
						if env.Name == envVarNameAlerEmailAddress && env.Value == integreatlyAlertEmailAddress {
							alertEnvVarChecked = true
						}
					}
					if !storageEnvVarChecked {
						t.Fatalf("%s env var should be empty in csv file:\n%s", envVarNameUseClusterStorage, content)
					}
					if !alertEnvVarChecked {
						t.Fatalf("%s env var should be set to %s in csv file:\n%s", envVarNameAlerEmailAddress, "integreatly-notifications@redhat.com", content)
					}
					_, installMode := findInstallMode(csv.Spec.InstallModes, olmapiv1alpha1.InstallModeTypeSingleNamespace)
					if !installMode.Supported {
						t.Fatalf("%s value should be true in csv file:\n%s", olmapiv1alpha1.InstallModeTypeSingleNamespace, content)
					}

				case customResourceDefinition:
					if found := len(p.Chunks()); found != 1 {
						t.Fatalf("expected 1 but found %d chunk changes for %s", found, customResourceDefinition)
					}
					if found := p.Chunks()[0].Type(); found != diff.Add {
						t.Fatalf("the frist and only chunk type should be Add but found %d for %s", found, customResourceDefinition)
					}
					if found := len(p.Chunks()[0].Content()); found <= 0 {
						t.Fatalf("expected %s to be largern then 0 but found %d", customResourceDefinition, found)
					}
				default:
					t.Fatalf("unexpected file %s", file.Path())
				}
			}

			// Verify the manage-tenents repo HEAD is pointing to master
			head, err := managedTenantsRepo.Head()
			if err != nil {
				t.Fatal(err)
			}

			if founded := head.Name(); founded != "refs/heads/master" {
				t.Fatalf("the managed-tenants repo HEAD doesn't point to the master branch\nexpected: refs/heads/master\nfounded: %s", founded)
			}
		})
	}
}

func TestOSDAddonRelease(t *testing.T) {

	basedir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		version               string
		expectedMergeRequests []string
	}{
		{
			version: "2.1.0-rc1",
			expectedMergeRequests: []string{
				fmt.Sprintf(mergeRequestTitleTemplate, "stage", "2.1.0-rc1"),
			},
		},
		{
			version: "2.1.0",
			expectedMergeRequests: []string{
				fmt.Sprintf(mergeRequestTitleTemplate, "stage", "2.1.0"),
				fmt.Sprintf(mergeRequestTitleTemplate, "edge", "2.1.0"),
				fmt.Sprintf(mergeRequestTitleTemplate, "stable", "2.1.0"),
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("test osd addon release for version %s", c.version), func(t *testing.T) {

			createdMergeRequests := []string{}

			flags := &osdAddonReleaseFlags{version: c.version}

			// Prepare the version
			version, err := utils.NewRHMIVersion(flags.version)
			if err != nil {
				t.Fatal(err)
			}

			// Prepare the integreatly-operator directory
			integreatlyOperatorDir := path.Join(basedir, fmt.Sprintf("testdata/osdAddonReleaseIntegreatlyOperator%s", version))

			// Prepare the managed-teneants repo and dir
			managedTenantsDir, managedTenantsRepo := prepareManagedTenants(t, basedir)

			// Mock the managed-tenants repo
			managedTenantsRepoMock := &gitRepositoryMock{
				repository: managedTenantsRepo,
				push:       func(o *git.PushOptions) error { return nil },
			}

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
					o *gitlab.CreateMergeRequestOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {

					// Push each merge request into the
					createdMergeRequests = append(createdMergeRequests, *o.Title)

					return &gitlab.MergeRequest{}, &gitlab.Response{}, nil
				},
			}

			// Create the osdAddonReleaseCmd object
			cmd := &osdAddonReleaseCmd{
				flags:                  flags,
				version:                version,
				gitlabMergeRequests:    gitlabMergeRequestMock,
				gitlabProjects:         gitlabProjectsMock,
				integreatlyOperatorDir: integreatlyOperatorDir,
				managedTenantsDir:      managedTenantsDir,
				managedTenantsRepo:     managedTenantsRepoMock,
			}

			// Run
			err = cmd.run()
			if err != nil {
				t.Fatalf("osdAddonReleaseCmd failed with error: %s", err)
			}

			// Verif that Merge Requests created
			if !reflect.DeepEqual(c.expectedMergeRequests, createdMergeRequests) {
				t.Fatalf(
					"the expected merge requests don't match the created merge requests\nexpected: %s\ncreated: %s",
					c.expectedMergeRequests, createdMergeRequests,
				)
			}
		})
	}
}
