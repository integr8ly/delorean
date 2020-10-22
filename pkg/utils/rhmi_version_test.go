package utils

import (
	"testing"
)

func TestReleaseVersion(t *testing.T) {

	cases := []struct {
		description    string
		version        string
		branchName     string
		tagName        string
		preRelease     bool
		isPatchRelease bool
		branchImageTag string
		expectError    bool
	}{
		{
			description:    "Verify release version",
			version:        "2.0.0",
			branchName:     "release-v2.0",
			tagName:        "v2.0.0",
			preRelease:     false,
			expectError:    false,
			isPatchRelease: false,
			branchImageTag: "master",
		},
		{
			description:    "Verify pre release version",
			version:        "2.0.0-ER1",
			branchName:     "release-v2.0",
			tagName:        "v2.0.0-ER1",
			preRelease:     true,
			expectError:    false,
			isPatchRelease: false,
			branchImageTag: "master",
		},
		{
			description:    "Verify patch release version",
			version:        "2.0.1",
			branchName:     "release-v2.0",
			tagName:        "v2.0.1",
			preRelease:     false,
			expectError:    false,
			isPatchRelease: true,
			branchImageTag: "2.0",
		},
		{
			description: "When the version is empty it should fails",
			version:     "",
			expectError: true,
		},
		{
			description: "When the version is wrong it should fails",
			version:     "2.0.0-er1-two",
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			v, err := NewRHMIVersion(c.version)

			if c.expectError && err == nil {
				t.Fatalf("error should not be nil")
			} else if !c.expectError {
				if err != nil {
					t.Fatalf("expected to parse %s but it fails with: %v", c.version, err)
				}
				if actual, wanted := v.IsPreRelease(), c.preRelease; actual != wanted {
					t.Fatalf("expected %s to not be a prerelease version", c.version)
				}

				if actual, wanted := v.String(), c.version; actual != wanted {
					t.Fatalf("expected %s when stringify the version but found %s", wanted, actual)
				}

				if actual, wanted := v.ReleaseBranchName(), c.branchName; actual != wanted {
					t.Fatalf("expected %s when build branch name but found %s", wanted, actual)
				}

				if actual, wanted := v.TagName(), c.tagName; actual != wanted {
					t.Fatalf("expected %s when build tag name but found %s", wanted, actual)
				}

				if actual, wanted := v.IsPatchRelease(), c.isPatchRelease; actual != wanted {
					t.Fatalf("expected %t when check if patch release but found %t", wanted, actual)
				}

				if actual, wanted := v.ReleaseBranchImageTag(), c.branchImageTag; actual != wanted {
					t.Fatalf("expected %s when build branchImageTag but found %s", wanted, actual)
				}
			}
		})
	}
}

func TestRHMIVersion_InitialPointReleaseTag(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		olmType string
	}{
		{
			name:    "test same point value",
			version: "2.1.0",
			want:    "v2.1.0",
		},
		{
			name:    "test same point value with build",
			version: "2.1.0-er1",
			want:    "v2.1.0",
		},
		{
			name:    "test point value with build",
			version: "2.1.1-er1",
			want:    "v2.1.0",
		},
		{
			name:    "test point value",
			version: "2.1.1",
			want:    "v2.1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _ := NewRHMIVersion(tt.version)
			if got := v.InitialPointReleaseTag(); got != tt.want {
				t.Errorf("RHMIVersion.InitialPointReleaseTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMIVersion_MajorMinor(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		olmType string
	}{
		{
			name:    "test same point value",
			version: "2.1.0",
			want:    "2.1",
		},
		{
			name:    "test same point value with build",
			version: "2.1.0-er1",
			want:    "2.1",
		},
		{
			name:    "test point value with build",
			version: "2.1.1-er1",
			want:    "2.1",
		},
		{
			name:    "test point value",
			version: "2.1.1",
			want:    "2.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _ := NewRHMIVersion(tt.version)
			if got := v.MajorMinor(); got != tt.want {
				t.Errorf("RHMIVersion.MajorMinor() = %v, want %v", got, tt.want)
			}
		})
	}
}
