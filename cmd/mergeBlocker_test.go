package cmd

import (
	"context"
	"errors"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"net/http"
	"testing"
)

func convertLabels(labelStr []string) []*github.Label {
	o := []*github.Label{}
	for _, l := range labelStr {
		o = append(o, &github.Label{
			Name: &l,
		})
	}
	return o
}

func toIssue(req *github.IssueRequest) *github.Issue {
	url := "http://testurl"
	return &github.Issue{
		Title:   req.Title,
		State:   req.State,
		Labels:  convertLabels(*req.Labels),
		HTMLURL: &url,
	}
}

type mockGithubIssuesService struct {
	ListByRepoFunc func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	CreateFunc     func(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
	EditFunc       func(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
}

func (m mockGithubIssuesService) ListByRepo(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	if m.ListByRepoFunc != nil {
		return m.ListByRepoFunc(ctx, owner, repo, opts)
	}
	panic("ListByRepoFunc is not defined")
}

func (m mockGithubIssuesService) Create(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, owner, repo, issue)
	}
	panic("CreateFunc is not defined")
}

func (m mockGithubIssuesService) Edit(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	if m.EditFunc != nil {
		return m.EditFunc(ctx, owner, repo, number, issue)
	}
	panic("EditFunc is not defined")
}

func responseWithCode(code int) *github.Response {
	return &github.Response{Response: &http.Response{
		StatusCode: code,
	}}
}

func TestSearchMergeBlockers(t *testing.T) {
	cases := []struct {
		description string
		client      services.GithubIssuesService
		branch      string
		verify      func(t *testing.T, issue *github.Issue, err error)
	}{
		{
			description: "test matching issue found",
			branch:      "master",
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					title := "release blocker|branch:master"
					return []*github.Issue{
						&github.Issue{
							Title: &title,
						},
					}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc:   nil,
			},
			verify: func(t *testing.T, issue *github.Issue, err error) {
				if err != nil {
					t.Fatal("error found:", err)
				} else if issue == nil {
					t.Fatal("issue is not found")
				}
			},
		},
		{
			description: "test matching issue not found",
			branch:      "master",
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					title := "master|branch:release-v2.1"
					return []*github.Issue{
						&github.Issue{
							Title: &title,
						},
					}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc:   nil,
			},
			verify: func(t *testing.T, issue *github.Issue, err error) {
				if err != nil {
					t.Fatal("error found:", err)
				} else if issue != nil {
					t.Fatal("issue should be nil, but found:", issue)
				}
			},
		},
		{
			description: "test error",
			branch:      "master",
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					return nil, nil, errors.New("unexpected error")
				},
				CreateFunc: nil,
				EditFunc:   nil,
			},
			verify: func(t *testing.T, issue *github.Issue, err error) {
				if err == nil {
					t.Fatal("error should not be nil")
				} else if issue != nil {
					t.Fatal("issue should be nil, but found:", issue)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			issue, err := searchMergeBlockers(context.TODO(), c.client, &githubRepoInfo{owner: DefaultIntegreatlyGithubOrg, repo: DefaultIntegreatlyOperatorRepo}, c.branch)
			c.verify(t, issue, err)
		})
	}
}

func TestDoMergeBlocker(t *testing.T) {
	cases := []struct {
		description string
		client      services.GithubIssuesService
		opts        *mergeBlockerCmdOptions
		expectError bool
	}{
		{
			description: "test create ok",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: false,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					return []*github.Issue{}, responseWithCode(200), nil
				},
				CreateFunc: func(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (issue2 *github.Issue, response *github.Response, err error) {
					return toIssue(issue), responseWithCode(201), nil
				},
				EditFunc: nil,
			},
			expectError: false,
		},
		{
			description: "test issue already exists",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: false,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					title := "release blocker|branch:master"
					url := "http://test"
					return []*github.Issue{
						&github.Issue{
							Title:   &title,
							HTMLURL: &url,
						},
					}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc:   nil,
			},
			expectError: false,
		},
		{
			description: "test create error",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: false,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					return []*github.Issue{}, responseWithCode(200), nil
				},
				CreateFunc: func(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (issue2 *github.Issue, response *github.Response, err error) {
					return nil, nil, errors.New("Unexpected error")
				},
				EditFunc: nil,
			},
			expectError: true,
		},
		{
			description: "test close ok",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: true,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					title := "release blocker|branch:master"
					url := "http://test"
					num := 1
					return []*github.Issue{
						&github.Issue{
							Title:   &title,
							HTMLURL: &url,
							Number:  &num,
						},
					}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc: func(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (issue2 *github.Issue, response *github.Response, err error) {
					return toIssue(issue), responseWithCode(200), nil
				},
			},
			expectError: false,
		},
		{
			description: "test issue not exists",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: true,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					return []*github.Issue{}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc:   nil,
			},
			expectError: true,
		},
		{
			description: "test close error",
			opts: &mergeBlockerCmdOptions{
				baseBranch: "master",
				isDeletion: true,
			},
			client: &mockGithubIssuesService{
				ListByRepoFunc: func(ctx context.Context, owner string, repo string, opts *github.IssueListByRepoOptions) (issues []*github.Issue, response *github.Response, err error) {
					title := "release blocker|branch:master"
					url := "http://test"
					num := 1
					return []*github.Issue{
						&github.Issue{
							Title:   &title,
							HTMLURL: &url,
							Number:  &num,
						},
					}, responseWithCode(200), nil
				},
				CreateFunc: nil,
				EditFunc: func(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (issue2 *github.Issue, response *github.Response, err error) {
					return nil, nil, errors.New("Unexpected error")
				},
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			err := DoMergeBlocker(context.TODO(), c.client, &githubRepoInfo{owner: DefaultIntegreatlyGithubOrg, repo: DefaultIntegreatlyOperatorRepo}, c.opts)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
