package cmd

import (
	"github.com/blang/semver"
	"github.com/integr8ly/delorean/pkg/utils"
	"regexp"
	"testing"
)

func getCurrentVersionMock(version string) func(string, string, *regexp.Regexp) (*semver.Version, error) {
	return func(a string, b string, r *regexp.Regexp) (*semver.Version, error) {
		v, _ := semver.Parse(version)
		return &v, nil
	}
}

func TestImageVersion(t *testing.T) {
	tests := []struct {
		name                  string
		flags                 *verifyImageVersionFlags
		getCurrentVersionFunc func(string, string, *regexp.Regexp) (*semver.Version, error)
		wantErr               bool
	}{
		{
			name: "test new version is ahead of the current version - major update",
			flags: &verifyImageVersionFlags{
				opDir:     "",
				imageType: "envoyProxy",
				newImage:  "envoyProxy:2.0.0",
			},
			wantErr:               false,
			getCurrentVersionFunc: getCurrentVersionMock("1.0.0"),
		},
		{
			name: "test new version is ahead of the current version - minor update",
			flags: &verifyImageVersionFlags{
				opDir:     "",
				imageType: "envoyProxy",
				newImage:  "envoyProxy:1.1.0",
			},
			wantErr:               false,
			getCurrentVersionFunc: getCurrentVersionMock("1.0.0"),
		},
		{
			name: "test new version is ahead of the current version - patch update",
			flags: &verifyImageVersionFlags{
				opDir:     "",
				imageType: "envoyProxy",
				newImage:  "envoyProxy:1.0.1",
			},
			wantErr:               false,
			getCurrentVersionFunc: getCurrentVersionMock("1.0.0"),
		},
		{
			name: "test new version is same of the current version (part 1)",
			flags: &verifyImageVersionFlags{
				opDir:     "",
				imageType: "envoyProxy",
				newImage:  "envoyProxy:1.0.0",
			},
			wantErr:               true,
			getCurrentVersionFunc: getCurrentVersionMock("1.0.0"),
		},
		{
			name: "test error returned on passing invalid image",
			flags: &verifyImageVersionFlags{
				opDir:     "",
				imageType: "invalidImage",
				newImage:  "invalidImage:2.0.0",
			},
			wantErr:               true,
			getCurrentVersionFunc: getCurrentVersionMock("1.0.0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			utils.ImageSubs["envoyProxy"].GetCurrentVersion = tt.getCurrentVersionFunc

			if err := verifyImageVersion(tt.flags); (err != nil) != tt.wantErr {
				t.Errorf("verifyImageVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
