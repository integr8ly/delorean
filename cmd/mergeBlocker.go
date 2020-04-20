package cmd

import (
	"context"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/spf13/cobra"
	"strings"
)

type mergeBlockerCmdOptions struct {
	baseBranch string
	isDeletion bool
}

var mergeBlockerCmdOpts = &mergeBlockerCmdOptions{}

const MergeBlockerLabel = "tide/merge-blocker"

// mergeBlockerCmd represents the mergeBlocker command
var mergeBlockerCmd = &cobra.Command{
	Use:   "merge-blocker",
	Short: "Change or delete merge blockers",
	Long:  `A merge blocker can block all merges against a given branch. The merge-blocker command can be used to create or delete merge blockers`,
	Run: func(cmd *cobra.Command, args []string) {
		var token string
		var err error
		if token, err = requireToken(GithubTokenKey); err != nil {
			handleError(err)
		}
		client := newGithubClient(token)
		repoInfo := &githubRepoInfo{owner: integreatlyGHOrg, repo: integreatlyOperatorRepo}
		if err = DoMergeBlocker(cmd.Context(), client.Issues, repoInfo, mergeBlockerCmdOpts); err != nil {
			handleError(err)
		}
	},
}

func DoMergeBlocker(ctx context.Context, client services.GithubIssuesService, repoInfo *githubRepoInfo, cmdOpts *mergeBlockerCmdOptions) error {
	if cmdOpts.isDeletion {
		if _, err := closeMergeBlocker(ctx, client, repoInfo, cmdOpts.baseBranch); err != nil {
			return err
		}
	} else {
		if _, err := createMergeBlocker(ctx, client, repoInfo, cmdOpts.baseBranch); err != nil {
			return err
		}
	}
	return nil
}

func createMergeBlocker(ctx context.Context, client services.GithubIssuesService, repoInfo *githubRepoInfo, branch string) (*github.Issue, error) {
	existing, err := searchMergeBlockers(ctx, client, repoInfo, branch)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		fmt.Println(fmt.Sprintf("Merge blocker issue is already created: %s", existing.GetHTMLURL()))
		return existing, nil
	}
	title := fmt.Sprintf("Merge Blocker|branch:%s", branch)
	state := "open"
	issue := &github.IssueRequest{
		Title:  &title,
		Labels: &([]string{MergeBlockerLabel}),
		State:  &state,
	}
	created, _, err := client.Create(ctx, repoInfo.owner, repoInfo.repo, issue)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("Merge blocker issue created: %s", created.GetHTMLURL()))
	return created, nil
}

func closeMergeBlocker(ctx context.Context, client services.GithubIssuesService, repoInfo *githubRepoInfo, branch string) (*github.Issue, error) {
	existing, err := searchMergeBlockers(ctx, client, repoInfo, branch)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return existing, fmt.Errorf("no merge blocker issue for the given branch: %s", branch)
	}
	state := "closed"
	issue := &github.IssueRequest{
		Title:  existing.Title,
		Labels: &([]string{MergeBlockerLabel}),
		State:  &state,
	}
	updated, _, err := client.Edit(ctx, repoInfo.owner, repoInfo.repo, *existing.Number, issue)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("Merge blocker issue closed: %s", updated.GetHTMLURL()))
	return updated, nil
}

func searchMergeBlockers(ctx context.Context, client services.GithubIssuesService, repoInfo *githubRepoInfo, branch string) (*github.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{MergeBlockerLabel},
	}
	issues, _, err := client.ListByRepo(ctx, repoInfo.owner, repoInfo.repo, opts)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if strings.Index(*issue.Title, fmt.Sprintf("branch:%s", branch)) > -1 {
			return issue, nil
		}
	}
	return nil, nil
}

func init() {
	releaseCmd.AddCommand(mergeBlockerCmd)
	mergeBlockerCmd.Flags().StringVarP(&mergeBlockerCmdOpts.baseBranch, "branch", "b", "master", "name of the branch to block merge")
	mergeBlockerCmd.Flags().BoolVarP(&mergeBlockerCmdOpts.isDeletion, "delete", "d", false, "Delete the merge blocker instead of create")
}
