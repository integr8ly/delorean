package cmd

import (
	"context"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/utils"
	"github.com/spf13/cobra"
)

type mergeReleaseOptions struct {
	releaseVersion string
	baseBranch     string
}

var mergeReleaseCmdOpts = &mergeReleaseOptions{}

// mergeReleaseCmd represents the mergeRelease command
var mergeReleaseCmd = &cobra.Command{
	Use:   "merge-release",
	Short: "Merge release PR for the given release version",
	Long:  `Merge release PR for the given release version`,
	Run: func(cmd *cobra.Command, args []string) {
		var token string
		var err error
		if token, err = requireValue(GithubTokenKey); err != nil {
			handleError(err)
		}
		client := newGithubClient(token)
		repoInfo := &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo}
		mergeReleaseCmdOpts.releaseVersion = releaseVersion
		if err = DoMergeRelease(cmd.Context(), client.PullRequests, repoInfo, mergeReleaseCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func DoMergeRelease(ctx context.Context, client services.PullRequestsService, repoInfo *githubRepoInfo, cmdOpts *mergeReleaseOptions) error {
	rv, err := utils.NewRHMIVersion(cmdOpts.releaseVersion)
	if err != nil {
		return err
	}
	opts := &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", repoInfo.owner, rv.ReleaseBranchName()),
		Base: cmdOpts.baseBranch,
	}
	fmt.Println("Try to find the release PR from", opts.Head, "against base branch", opts.Base)
	pr, err := findPRForRelease(ctx, client, repoInfo, opts)
	if err != nil {
		return err
	}
	fmt.Println("Release PR found:", pr.GetHTMLURL(), ". Merging.")
	msg := fmt.Sprintf("merge for release %s", cmdOpts.releaseVersion)
	_, err = mergePR(ctx, client, repoInfo, pr, msg)
	if err != nil {
		return err
	}
	fmt.Println("Release PR merged.")
	return nil
}

func findPRForRelease(ctx context.Context, client services.PullRequestsService, repoInfo *githubRepoInfo, opts *github.PullRequestListOptions) (*github.PullRequest, error) {
	prs, _, err := client.List(ctx, repoInfo.owner, repoInfo.repo, opts)
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, fmt.Errorf("no open pull request found with options: %+v", opts)
	}
	if len(prs) > 1 {
		return nil, fmt.Errorf("more than 1 pull requests found with options: %+v. Please close some of them", opts)
	}
	pr, _, err := client.Get(ctx, repoInfo.owner, repoInfo.repo, *prs[0].Number)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func mergePR(ctx context.Context, client services.PullRequestsService, repoIno *githubRepoInfo, pr *github.PullRequest, msg string) (*string, error) {
	if *pr.Merged {
		fmt.Println("Pull request is already merged:", pr.GetHTMLURL())
		return pr.MergeCommitSHA, nil
	}
	if *pr.State == "closed" {
		return nil, fmt.Errorf("pull request is closed but not merged: %s", pr.GetHTMLURL())
	}
	if !*pr.Mergeable {
		return nil, fmt.Errorf("pull request is not mergeable. Please fix the issue first. Link: %s", pr.GetHTMLURL())
	}
	result, _, err := client.Merge(ctx, repoIno.owner, repoIno.repo, *pr.Number, msg, &github.PullRequestOptions{})
	if err != nil {
		return nil, err
	}
	if !*result.Merged {
		return nil, fmt.Errorf("something went wrong and the merge has failed. Please try to merge the PR manually. Link: %s", pr.GetHTMLURL())
	}
	return result.SHA, nil
}

func init() {
	releaseCmd.AddCommand(mergeReleaseCmd)
	mergeReleaseCmd.Flags().StringVarP(&mergeReleaseCmdOpts.baseBranch, "branch", "b", "master", "Base branch for the PR to merge")
}
