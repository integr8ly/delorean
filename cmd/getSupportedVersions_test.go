package cmd

import (
	"context"
	"github.com/blang/semver"
	"github.com/integr8ly/delorean/pkg/types"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestGetSupportedVersionsCmd(t *testing.T) {
	cases := []struct {
		description   string
		olmType       string
		majorVersions int
		minorVersions int
		repo          string
		expectError   bool
	}{
		{
			description:   "Run command for RHOAM",
			olmType:       "managed-api-service",
			majorVersions: 1,
			minorVersions: 3,
			repo:          "https://gitlab.cee.redhat.com/service/managed-tenants.git",
			expectError:   false,
		},
		{
			description:   "Run command for RHMI",
			olmType:       "integreatly-operator",
			majorVersions: 1,
			minorVersions: 3,
			repo:          "https://gitlab.cee.redhat.com/service/managed-tenants.git",
			expectError:   false,
		},
		{
			description:   "Run command with expected error",
			olmType:       "managed-api-service",
			majorVersions: 1,
			minorVersions: 3,
			repo:          "https://gitlab.cee.redhat.com/bad-path/managed-tenants.git",
			expectError:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			cmd := &getSupportedVersionsCmd{
				olmType:                c.olmType,
				supportedMajorVersions: c.majorVersions,
				supportedMinorVersions: c.minorVersions,
				managedTenants:         c.repo,
			}

			result, err := cmd.run(context.TODO())
			if err != nil && !c.expectError {
				if strings.HasSuffix(err.Error(), "no such host") {
					t.Skipf("No access to repo: %s, Error: %s", c.repo, err)
				}
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && c.expectError {
				t.Fatal("error expected but got nil")
			}
			if result == nil && !c.expectError {
				t.Fatalf("No patch versions were found")
			}
		})
	}
}

func TestGetBundleFolders(t *testing.T) {
	basedir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		description string
		dir         string
		path        string
		expected    []string
	}{
		{
			description: "Return all bundle folders for managed-api-service",
			dir:         path.Join(basedir, "testdata/getSupportedVersions/managed-tenants"),
			path:        "addons/rhoams/bundles",
			expected:    []string{"1.4.0", "1.5.0", "1.6.0", "1.6.1", "1.7.0", "1.7.1", "1.7.2"},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			result, _ := getBundleFolders(c.dir, c.path)

			if len(result) != len(c.expected) {
				t.Fatalf("List do not match, expected: %v, result: %v", len(result), len(c.expected))
			}
		})
	}
}

func TestGetSemverValues(t *testing.T) {
	cases := []struct {
		description string
		bundles     []string
		expected    []semver.Version
	}{
		{
			description: "Creating a list of semver values",
			bundles:     []string{"1.4.0", "1.5.0", "1.6.0", "1.6.1", "1.7.0", "1.7.1", "1.7.2"},
			expected: []semver.Version{
				{Major: 1, Minor: 4, Patch: 0},
				{Major: 1, Minor: 5, Patch: 0},
				{Major: 1, Minor: 6, Patch: 0},
				{Major: 1, Minor: 6, Patch: 1},
				{Major: 1, Minor: 7, Patch: 0},
				{Major: 1, Minor: 7, Patch: 1},
				{Major: 1, Minor: 7, Patch: 2},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			result, err := getSemverValues(c.bundles)

			if err != nil {
				t.Fatalf("An unexpected error happened, %s", err)
			}

			if len(result) != len(c.expected) {
				t.Fatalf("Length of semver lists was not correct. Expected: %v Recived: %v", len(c.expected), len(result))
			}
		})
	}
}

func TestGetMajorVersions(t *testing.T) {
	cases := []struct {
		description       string
		supportedVersions int
		versions          []semver.Version
		expected          []int
	}{
		{
			description:       "Get the one major versions",
			supportedVersions: 1,
			versions: []semver.Version{
				{Major: 0, Minor: 7, Patch: 0},
				{Major: 0, Minor: 8, Patch: 0},
				{Major: 0, Minor: 9, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 1, Patch: 0},
				{Major: 1, Minor: 2, Patch: 0},
			},
			expected: []int{1},
		},
		{
			description:       "Get the two major versions",
			supportedVersions: 2,
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 3, Minor: 0, Patch: 0},
			},
			expected: []int{2, 3},
		},
		{
			description:       "Get the three major versions",
			supportedVersions: 3,
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 3, Minor: 0, Patch: 0},
			},
			expected: []int{1, 2, 3},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			majorVersions, err := getMajorVersions(c.versions, c.supportedVersions)

			if err != nil {
				t.Fatalf("Unexpected Error: %s", err)
			}
			result := compareIntList(majorVersions, c.expected)

			if !result {
				t.Fatalf("compared lists did not match, Expected: %v Recived: %v", c.expected, majorVersions)
			}

		})
	}
}

func TestGetMinorVersions(t *testing.T) {
	cases := []struct {
		description       string
		supportedVersions int
		majorVersions     []int
		versions          []semver.Version
		expected          map[int][]int
	}{
		{
			description:       "Check two major and minor version",
			supportedVersions: 2,
			majorVersions:     []int{2, 3},
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 3, Minor: 0, Patch: 0},
				{Major: 3, Minor: 1, Patch: 0},
				{Major: 3, Minor: 2, Patch: 0},
			},
			expected: map[int][]int{
				2: {0},
				3: {1, 2},
			},
		},
		{
			description:       "Check three major and minor version",
			supportedVersions: 3,
			majorVersions:     []int{1, 2, 3},
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 3, Minor: 0, Patch: 0},
				{Major: 3, Minor: 1, Patch: 0},
				{Major: 3, Minor: 2, Patch: 0},
			},
			expected: map[int][]int{
				1: {0},
				2: {0},
				3: {0, 1, 2},
			},
		},
		{
			description:       "Check Minor versions with random patch",
			supportedVersions: 3,
			majorVersions:     []int{1},
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 4, Patch: 0},
				{Major: 1, Minor: 5, Patch: 0},
				{Major: 1, Minor: 6, Patch: 0},
				{Major: 1, Minor: 6, Patch: 1},
				{Major: 1, Minor: 7, Patch: 0},
			},
			expected: map[int][]int{
				1: {5, 6, 7},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			minorVersion, err := getMinorVersions(c.versions, c.majorVersions, c.supportedVersions)

			if err != nil {
				t.Fatalf("Unexpected Error: %s", err)
			}
			result := compareMinorVersionResult(minorVersion, c.expected)
			if !result {
				t.Fatalf("Wrong minor versions returned. Expected: %v, Recived: %v", c.expected, minorVersion)
			}

		})
	}
}

func TestGetPatchVersions(t *testing.T) {
	cases := []struct {
		description       string
		versions          []semver.Version
		supportedVersions map[int][]int
		expected          []string
	}{
		{
			description: "Three Patch versions for One major and minor",
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
				{Major: 1, Minor: 0, Patch: 2},
			},
			supportedVersions: map[int][]int{
				1: {0},
			},
			expected: []string{"1.0.0", "1.0.1", "1.0.2"},
		},
		{
			description: "Six Patch versions for two major and three minor",
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
				{Major: 1, Minor: 0, Patch: 2},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 2, Minor: 1, Patch: 0},
				{Major: 2, Minor: 2, Patch: 0},
			},
			supportedVersions: map[int][]int{
				1: {0},
				2: {0, 1, 2},
			},
			expected: []string{"1.0.0", "1.0.1", "1.0.2", "2.0.0", "2.1.0", "2.2.0"},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			patchVersions, err := getPatchVersions(c.versions, c.supportedVersions)
			if err != nil {
				t.Fatalf("Unexpected Error: %s", err)
			}

			result := compareStringList(patchVersions, c.expected)

			if !result {
				t.Fatalf("Patch Versions do not match. Expected: %s, Recived: %s", c.expected, patchVersions)
			}
		})
	}
}

func compareMinorVersionResult(versions map[int][]int, expected map[int][]int) bool {
	if len(versions) != len(expected) {
		return false
	}

	for i := range versions {
		found := false
		if compareIntList(versions[i], expected[i]) {
			found = true
		}
		if !found {
			return false
		}
	}
	return true
}

func TestGetOlmTypePath(t *testing.T) {
	cases := []struct {
		description        string
		olmType            string
		expectedBundlePath string
		expectedFilePath   string
		expectedError      string
		hasError           bool
	}{
		{
			description:        "Get values for RHOAM",
			olmType:            types.OlmTypeRhoam,
			expectedBundlePath: "managed-api-service",
			expectedFilePath:   "addons/rhoams/metadata/production/addon.yaml",
			hasError:           false,
		},
		{
			description:        "Get values for RHMI",
			olmType:            types.OlmTypeRhmi,
			expectedBundlePath: "addons/integreatly-operator/bundles",
			expectedFilePath:   "addons/integreatly-operator/metadata/production/addon.yaml",

			hasError: false,
		},
		{
			description:   "Unsupported Type",
			olmType:       types.OlmTypeRhoam,
			expectedError: "Unsupported Olm Type",
			hasError:      true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			paths, err := getOlmTypePaths(c.olmType)
			if err != nil {
				if c.hasError && err.Error() != c.expectedError {
					t.Fatalf("Did not get expected error. Expected: %s, Recived: %s", c.expectedError, err)
				} else if !c.hasError {
					t.Fatalf("Unexpected Error, %s", err)
				}
			} else {
				if paths.bundleFolder != c.expectedBundlePath && !c.hasError {
					t.Fatalf("Wrong path returned. Expected: %s, Recived: %s", c.expectedBundlePath, paths.bundleFolder)
				}

				if paths.addonFilePath != c.expectedFilePath && !c.hasError {
					t.Fatalf("Wrong path returned. Expected: %s, Recived: %s", c.expectedFilePath, paths.addonFilePath)
				}
			}
		})
	}
}

func TestDownloadManagedTenants(t *testing.T) {
	cases := []struct {
		description string
		url         string
		expected    string
	}{
		{
			description: "Download managed tenants from service delivery",
			url:         "https://gitlab.cee.redhat.com/service/managed-tenants.git",
			expected:    "/tmp/managed-tenants",
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			repoPath, err := downloadManagedTenants(c.url)
			if err != nil {
				if strings.HasSuffix(err.Error(), "no such host") {
					t.Skipf("No access to repo: %s, Error: %s", c.url, err)
				}
				t.Fatalf("Unexpected Error: %s", err)
			}

			if !strings.HasPrefix(repoPath, c.expected) {
				t.Fatalf("Repo path not started with expected. Expected: %s, Recived: %s", c.expected, repoPath)
			}

		})
	}
}

func TestGetProductionVersion(t *testing.T) {
	basedir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		description string
		repoDir     string
		paths       olmPaths
		expected    semver.Version
	}{
		{
			description: "Get production version for rhoam",
			repoDir:     path.Join(basedir, "testdata/getSupportedVersions/managed-tenants"),
			paths: olmPaths{
				addonFilePath: "addons/rhoams/metadata/production/addon.yaml",
			},
			expected: semver.Version{Major: 1, Minor: 6, Patch: 1},
		},
		{
			description: "Get production version for rhmi",
			repoDir:     path.Join(basedir, "testdata/getSupportedVersions/managed-tenants"),
			paths: olmPaths{
				addonFilePath: "addons/integreatly-operator/metadata/production/addon.yaml",
			},
			expected: semver.Version{Major: 2, Minor: 8, Patch: 0},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			result, err := getProductionVersion(c.repoDir, c.paths)
			if err != nil {
				t.Fatalf("Unexpected Error: %s", err)
			}

			if !reflect.DeepEqual(result, c.expected) {
				t.Fatalf("Version do not match, Expected: %s, Recived: %s", c.expected, result)
			}
		})
	}
}

func TestTrimSemverVersions(t *testing.T) {
	cases := []struct {
		description    string
		versions       []semver.Version
		currentVersion semver.Version
		expected       []semver.Version
	}{
		{
			description: "Two minor version less in production",
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
				{Major: 1, Minor: 1, Patch: 0},
				{Major: 1, Minor: 2, Patch: 0},
			},
			currentVersion: semver.Version{Major: 1, Minor: 0},
			expected: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
			},
		},
		{
			description: "Two major version less in production",
			versions: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
				{Major: 1, Minor: 1, Patch: 0},
				{Major: 2, Minor: 0, Patch: 0},
				{Major: 3, Minor: 0, Patch: 0},
			},
			currentVersion: semver.Version{Major: 1, Minor: 0},
			expected: []semver.Version{
				{Major: 0, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 0},
				{Major: 1, Minor: 0, Patch: 1},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			result, err := trimSemverVersions(c.versions, c.currentVersion)
			if err != nil {
				t.Fatalf("Unexected Error: %s", err)
			}

			if !reflect.DeepEqual(c.expected, result) {
				t.Fatalf("Version do not match, Expected: %s, Recived %s", c.expected, result)
			}
		})
	}

}

func compareIntList(versions []int, expected []int) bool {
	if len(versions) != len(expected) {
		return false
	}
	for _, i := range versions {
		found := false
		for _, j := range expected {
			if i == j {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func compareStringList(versions []string, expected []string) bool {
	if len(versions) != len(expected) {
		return false
	}
	for _, i := range versions {
		found := false
		for _, j := range expected {
			if i == j {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
