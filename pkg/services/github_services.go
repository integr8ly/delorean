package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v30/github"
)

type GithubIssuesService interface {
	ListByRepo(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	Create(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
	Edit(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
}

type PullRequestsService interface {
	List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	Get(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
	Merge(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
	Create(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
}

type GitService interface {
	GetRefs(ctx context.Context, owner string, repo string, ref string) ([]*github.Reference, *github.Response, error)
	CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
}

type GithubReleaseService interface {
	GetLatestRelease(owner string, repo string, client *github.Client) (error, string)
}

type DefaultGithubReleaseService struct{}

func (s *DefaultGithubReleaseService) GetLatestRelease(owner string, repo string, client *github.Client) (error, string) {
	releases, _, err := client.Repositories.ListReleases(context.TODO(), owner, repo, &github.ListOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("Error attempting to get release from github, owner: %s, repo: %s. Error: %v", repo, owner, err.Error())), ""
	}

	if len(releases) > 0 {
		return nil, *releases[0].Name
	}
	return errors.New(fmt.Sprintf("No release found on github, owner: %s, repo: %s", repo, owner)), ""
}
