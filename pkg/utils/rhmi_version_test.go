package utils

import (
	"testing"
)

func TestReleaseVersion(t *testing.T) {

	cases := []struct {
		description string
		version     string
		branchName  string
		tagName     string
		preRelease  bool
		expectError bool
	}{
		{
			description: "Verify release version",
			version:     "2.0.0",
			branchName:  "release-v2.0.0",
			tagName:     "v2.0.0",
			preRelease:  false,
			expectError: false,
		},
		{
			description: "Verify pre release version",
			version:     "2.0.0-ER1",
			branchName:  "release-v2.0.0",
			tagName:     "v2.0.0-ER1",
			preRelease:  true,
			expectError: false,
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
				if actual, wanted := v.IsPreRrelease(), c.preRelease; actual != wanted {
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
			}
		})
	}
}
