package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type tagRepositoryFlags struct {
	organization   string
	repository     string
	branch         string
	skipPreRelease bool
}

func init() {

	flags := &tagRepositoryFlags{}

	cmd := &cobra.Command{
		Use:   "tag-repository",
		Short: "Tag the passed repository with the given release on the given branch",
		Run: func(cmd *cobra.Command, args []string) {

			ghToken, err := requireValue(GithubTokenKey)
			if err != nil {
				handleError(err)
			}
			ghClient := newGithubClient(ghToken)

			version := releaseVersion
			if version == "" {
				handleError(errors.New("version is not defined"))
			}

			if err := runTagRepository(cmd.Context(), ghClient.Git, version, flags); err != nil {
				handleError(err)
			}
		},
	}

	cmd.Flags().StringVar(&flags.organization, "organization", "", "GitHub organization where the repository reside")
	cmd.Flags().StringVar(&flags.repository, "repository", "", "Repository in the GitHub organization on which to create the tag")
	cmd.Flags().StringVar(&flags.branch, "branch", "master", "Branch to create the tag")
	cmd.Flags().BoolVar(&flags.skipPreRelease, "skip-pre-release", false, "Don't tag te repository if the version is a pre-release")
	cmd.MarkFlagRequired("organization")
	cmd.MarkFlagRequired("repository")

	releaseCmd.AddCommand(cmd)
}

func runTagRepository(ctx context.Context, ghClient services.GitService, version string, flags *tagRepositoryFlags) error {
	v, err := utils.NewRHMIVersion(version)
	if err != nil {
		return err
	}

	if flags.skipPreRelease && v.IsPreRelease() {
		fmt.Println("Skip pre-release version:", v)
		return nil
	}

	repo := &githubRepoInfo{owner: flags.organization, repo: flags.repository}

	branchRefName := plumbing.NewBranchReferenceName(flags.branch)
	fmt.Println("Fetch git ref:", branchRefName)
	headRef, err := getGitRef(ctx, ghClient, repo, branchRefName.String(), false)
	if err != nil {
		return err
	}
	if headRef == nil {
		return fmt.Errorf("can not find git ref: %s", branchRefName)
	}

	fmt.Println("Create git tag:", v.TagName())
	tagRef, err := createGitTag(ctx, ghClient, repo, v.TagName(), headRef.GetObject().GetSHA())
	if err != nil {
		return err
	}
	fmt.Println("Git tag", v.TagName(), "created:", tagRef.GetURL())

	return nil
}
