package cmd

import (
	"context"
	"fmt"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type tagReleaseRepoOptions struct {
	releaseVersion string
	branch         string
	olmType        string
	sourceTag      string
}

var tagReleaseRepoCmdOpts = &tagReleaseRepoOptions{}

var tagReleaseRepoCmd = &cobra.Command{
	Use:   "tag-release-repo",
	Short: "Tag the integreatly repo",
	Long:  `Change a release tag using the given release version for the HEAD of the given branch.`,
	Run: func(cmd *cobra.Command, args []string) {
		var ghToken string
		var err error
		if ghToken, err = requireValue(GithubTokenKey); err != nil {
			handleError(err)
		}

		ghClient := newGithubClient(ghToken)
		repoInfo := &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo}
		tagReleaseRepoCmdOpts.releaseVersion = releaseVersion
		tagReleaseRepoCmdOpts.olmType = olmType
		if err = DoTagReleaseRepo(cmd.Context(), ghClient.Git, repoInfo, tagReleaseRepoCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func DoTagReleaseRepo(ctx context.Context, ghClient services.GitService, gitRepoInfo *githubRepoInfo, cmdOpts *tagReleaseRepoOptions) error {
	rv, err := utils.NewVersion(cmdOpts.releaseVersion, cmdOpts.olmType)
	if err != nil {
		return err
	}
	fmt.Println("Fetch git ref:", fmt.Sprintf("refs/heads/%s", cmdOpts.branch))
	headRef, err := getGitRef(ctx, ghClient, gitRepoInfo, fmt.Sprintf("refs/heads/%s", cmdOpts.branch), false)
	if err != nil {
		return err
	}

	fmt.Println("Create git tag:", rv.TagName())
	if headRef == nil {
		return fmt.Errorf("can not find git ref: refs/heads/%s", cmdOpts.branch)
	}
	tagRef, err := createGitTag(ctx, ghClient, gitRepoInfo, rv.TagName(), headRef.GetObject().GetSHA())
	if err != nil {
		return err
	}
	fmt.Println("Git tag", rv.TagName(), "created:", tagRef.GetURL())

	return nil
}

func createGitTag(ctx context.Context, client services.GitService, gitRepoInfo *githubRepoInfo, tag string, sha string) (*github.Reference, error) {
	tagRefVal := fmt.Sprintf("refs/tags/%s", tag)
	tagRef, err := getGitRef(ctx, client, gitRepoInfo, tagRefVal, false)
	if err != nil {
		return nil, err
	}
	if tagRef != nil {
		if tagRef.GetObject().GetSHA() != sha {
			return nil, fmt.Errorf("tag %s is already created but pointing to a different commit. Please delete it first", tag)
		}
		return tagRef, nil
	}
	tagRef = &github.Reference{
		Ref: &tagRefVal,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}
	created, _, err := client.CreateRef(ctx, gitRepoInfo.owner, gitRepoInfo.repo, tagRef)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func init() {
	releaseCmd.AddCommand(tagReleaseRepoCmd)
	tagReleaseRepoCmd.Flags().StringVarP(&tagReleaseRepoCmdOpts.branch, "branch", "b", "master", "Branch to create the tag")
	tagReleaseRepoCmd.Flags().StringVar(&tagReleaseRepoCmdOpts.sourceTag, "sourceTag", "", "OSD Source Tag passed through pipeline.")
}
