/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/spf13/cobra"
	"net/http"
	"strings"
)

var baseBranch string
var isDeletion bool

const MergeBlockerLabel = "tide/merge-blocker"

// mergeBlockerCmd represents the mergeBlocker command
var mergeBlockerCmd = &cobra.Command{
	Use:   "merge-blocker",
	Short: "Create or delete merge blockers",
	Long:  `A merge blocker can block all merges against a given branch. The merge-blocker command can be used to create or delete merge blockers`,
	Run: func(cmd *cobra.Command, args []string) {
		var token string
		var err error
		if token, err = requireGithubToken(); err != nil {
			fmt.Println("Error:", err)
			return
		}
		client := newGithubClient(token)
		if isDeletion {
			if _, err = closeMergeBlocker(cmd.Context(), client.Issues, integreatlyGHOrg, integreatlyOperatorRepo, baseBranch); err != nil {
				fmt.Println("Error:", err)
				return
			}
		} else {
			if _, err = createMergeBlocker(cmd.Context(), client.Issues, integreatlyGHOrg, integreatlyOperatorRepo, baseBranch); err != nil {
				fmt.Println("Error:", err)
				return
			}
		}
	},
}

func createMergeBlocker(ctx context.Context, client services.GithubIssuesService, org string, repo string, branch string) (*github.Issue, error) {
	existing, err := searchMergeBlockers(ctx, client, integreatlyGHOrg, integreatlyOperatorRepo, branch)
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
	created, resp, err := client.Create(ctx, org, repo, issue)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Unexpected status code when creating a new issue: %d", resp.StatusCode)
	}
	fmt.Println(fmt.Sprintf("Merge blocker issue created: %s", created.GetHTMLURL()))
	return created, nil
}

func closeMergeBlocker(ctx context.Context, client services.GithubIssuesService, org string, repo string, branch string) (*github.Issue, error) {
	existing, err := searchMergeBlockers(ctx, client, integreatlyGHOrg, integreatlyOperatorRepo, branch)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return existing, fmt.Errorf("No merge blocker issue for the given branch: %s", branch)
	}
	state := "closed"
	issue := &github.IssueRequest{
		Title:  existing.Title,
		Labels: &([]string{MergeBlockerLabel}),
		State:  &state,
	}
	updated, resp, err := client.Edit(ctx, org, repo, *existing.Number, issue)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code when close the issue: %d", resp.StatusCode)
	}
	fmt.Println(fmt.Sprintf("Merge blocker issue closed: %s", updated.GetHTMLURL()))
	return updated, nil
}

func searchMergeBlockers(ctx context.Context, client services.GithubIssuesService, org string, repo string, branch string) (*github.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{MergeBlockerLabel},
	}
	issues, resp, err := client.ListByRepo(ctx, org, repo, opts)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code when listing issues for repo: %d", resp.StatusCode)
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
	mergeBlockerCmd.Flags().StringVarP(&baseBranch, "branch", "b", "master", "name of the branch to block merge")
	mergeBlockerCmd.Flags().BoolVarP(&isDeletion, "delete", "d", false, "Delete the merge blocker instead of create")
}
