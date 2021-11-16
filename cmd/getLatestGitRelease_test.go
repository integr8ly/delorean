package cmd

import (
	"errors"
	"github.com/google/go-github/v30/github"
	"testing"
)

type MockGithubReleaseService struct{}

func (s *MockGithubReleaseService) GetLatestRelease(owner string, repo string, client *github.Client) (error, string) {
	if owner == "invalid" {
		return errors.New("Invalid Repo"), ""
	} else {
		return nil, "1.1.1"
	}
}

func TestGetLatestGitRelease(t *testing.T) {

	tests := []struct {
		name    string
		flags   *GetLatestReleaseCmdFlags
		wantErr bool
	}{
		{
			name: "test getting latest release from valid repo",
			flags: &GetLatestReleaseCmdFlags{
				repo:    "ratelimit",
				owner:   "envoyproxy",
				service: &MockGithubReleaseService{},
			},
			wantErr: false,
		},
		{
			name: "test getting invalid release",
			flags: &GetLatestReleaseCmdFlags{
				repo:    "invalid",
				owner:   "invalid",
				service: &MockGithubReleaseService{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := github.NewClient(nil)
			if err, v := getLatestGitRelease(tt.flags, client); ((err != nil) != tt.wantErr) || ((err == nil) && v == "") {
				t.Errorf("getLatestGitRelease() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
