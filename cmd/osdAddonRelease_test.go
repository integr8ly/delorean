package cmd

import (
	"testing"
)

func TestReleaseVersion(t *testing.T) {

	cases := []struct {
		description string
		version     string
		verify      func(t *testing.T, version string, v *ReleaseVersion, err error)
	}{
		{
			description: "Verify release version",
			version:     "2.0.0",
			verify: func(t *testing.T, version string, v *ReleaseVersion, err error) {
				if err != nil {
					t.Fatalf("expected to parse %s but it fails with: %s", version, err)
				}

				if v.IsPreRrelease() {
					t.Fatalf("expected %s to not be a prerelease version", version)
				}

				if s := v.String(); version != s {
					t.Fatalf("expected %s when stringify the version but found %s", version, s)
				}
			},
		},
		{
			description: "Verify pre release version",
			version:     "2.0.0-ER1",
			verify: func(t *testing.T, version string, v *ReleaseVersion, err error) {
				if err != nil {
					t.Fatalf("expected to parse %s but it fails with: %s", version, err)
				}

				if !v.IsPreRrelease() {
					t.Fatalf("expected %s to be a prerelease version", version)
				}

				if s := v.String(); version != s {
					t.Fatalf("expected %s when stringify the version but found %s", version, s)
				}
			},
		},
		{
			description: "When the version is empty it should fails",
			version:     "",
			verify: func(t *testing.T, _ string, _ *ReleaseVersion, err error) {
				if err == nil {
					t.Fatalf("expected to fail when parsing an empty version")
				}
			},
		},
		{
			description: "When the version is wrong it should fails",
			version:     "2.0.0-er1-two",
			verify: func(t *testing.T, version string, _ *ReleaseVersion, err error) {
				if err == nil {
					t.Fatalf("expected to fail when parsing the wrong version %s", version)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			v, err := NewReleaseVersion(c.version)
			c.verify(t, c.version, v, err)
		})
	}
}
