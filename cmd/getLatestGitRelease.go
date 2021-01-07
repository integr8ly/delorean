package cmd

import (
	"fmt"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/spf13/cobra"
)

type GetLatestReleaseCmdFlags struct {
	repo    string
	owner   string
	service services.GithubReleaseService
}

func init() {
	f := &GetLatestReleaseCmdFlags{}

	cmd := &cobra.Command{
		Use:   "get-latest-release",
		Short: "Get the latest release from a git repo",
		Run: func(cmd *cobra.Command, args []string) {
			err, releaseVerison := getLatestGitRelease(NewGetLatestReleaseCmd(f.repo, f.owner))
			if err != nil {
				handleError(err)
				return
			}
			fmt.Println(releaseVerison)
		},
	}

	ewsCmd.AddCommand(cmd)
	cmd.Flags().StringVarP(&f.repo, "repo", "r", "", "Git repo from which to get latest release version")
	cmd.Flags().StringVarP(&f.owner, "owner", "o", "", "Git owner from which to get latest release version")
}

func NewGetLatestReleaseCmd(repo string, owner string) *GetLatestReleaseCmdFlags {
	return &GetLatestReleaseCmdFlags{
		repo:    repo,
		owner:   owner,
		service: &services.DefaultGithubReleaseService{},
	}
}

func getLatestGitRelease(flags *GetLatestReleaseCmdFlags) (error, string) {
	repo := flags.repo
	owner := flags.owner

	return flags.service.GetLatestRelease(owner, repo)
}
