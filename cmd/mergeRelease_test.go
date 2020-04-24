package cmd

import (
	"context"
	"errors"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"net/http"
	"testing"
)

type mockPullRequestsService struct {
	GetFunc    func(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListFunc   func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	MergeFunc  func(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
	CreateFunc func(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
}

func (m mockPullRequestsService) List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, owner, repo, opts)
	}
	panic("implement me")
}

func (m mockPullRequestsService) Merge(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error) {
	if m.MergeFunc != nil {
		return m.MergeFunc(ctx, owner, repo, number, commitMessage, options)
	}
	panic("implement me")
}

func (m mockPullRequestsService) Get(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, owner, repo, number)
	}
	panic("implement me")
}

func (m mockPullRequestsService) Create(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, owner, repo, pull)
	}
	panic("implement me")
}

func TestDoMergeRelease(t *testing.T) {
	cases := []struct {
		description string
		client      services.PullRequestsService
		opts        *mergeReleaseOptions
		expectError bool
	}{
		{
			description: "success",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: func(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (result *github.PullRequestMergeResult, response *github.Response, err error) {
					sha := "mergesha"
					merged := true
					return &github.PullRequestMergeResult{
						SHA:    &sha,
						Merged: &merged,
					}, responseWithCode(http.StatusOK), nil
				},
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					mergeable := true
					state := "open"
					merged := false
					return &github.PullRequest{
						HTMLURL:   &url,
						Number:    &num,
						Mergeable: &mergeable,
						State:     &state,
						Merged:    &merged,
					}, responseWithCode(http.StatusOK), nil
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: false,
		},
		{
			description: "test listing PR returns error",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					return nil, nil, errors.New("test error")
				},
				MergeFunc: nil,
				GetFunc:   nil,
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: true,
		},
		{
			description: "test get PR returns error",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: nil,
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					return nil, nil, errors.New("get error")
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: true,
		},
		{
			description: "test PR is merged",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: nil,
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					mergeable := true
					state := "closed"
					merged := true
					return &github.PullRequest{
						HTMLURL:   &url,
						Number:    &num,
						Mergeable: &mergeable,
						State:     &state,
						Merged:    &merged,
					}, responseWithCode(http.StatusOK), nil
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: false,
		},
		{
			description: "test PR is closed",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: nil,
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					mergeable := true
					state := "closed"
					merged := false
					return &github.PullRequest{
						HTMLURL:   &url,
						Number:    &num,
						Mergeable: &mergeable,
						State:     &state,
						Merged:    &merged,
					}, responseWithCode(http.StatusOK), nil
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: true,
		},
		{
			description: "test PR is not mergeable",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: nil,
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					mergeable := false
					state := "open"
					merged := false
					return &github.PullRequest{
						HTMLURL:   &url,
						Number:    &num,
						Mergeable: &mergeable,
						State:     &state,
						Merged:    &merged,
					}, responseWithCode(http.StatusOK), nil
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: true,
		},
		{
			description: "test merging PR returns error",
			client: &mockPullRequestsService{
				ListFunc: func(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) (requests []*github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					return []*github.PullRequest{{
						HTMLURL: &url,
						Number:  &num,
					}}, responseWithCode(http.StatusOK), nil
				},
				MergeFunc: func(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (result *github.PullRequestMergeResult, response *github.Response, err error) {
					return nil, nil, errors.New("merge error")
				},
				GetFunc: func(ctx context.Context, owner string, repo string, number int) (request *github.PullRequest, response *github.Response, err error) {
					url := "http://test"
					num := 1
					mergeable := true
					state := "open"
					merged := false
					return &github.PullRequest{
						HTMLURL:   &url,
						Number:    &num,
						Mergeable: &mergeable,
						State:     &state,
						Merged:    &merged,
					}, responseWithCode(http.StatusOK), nil
				},
			},
			opts:        &mergeReleaseOptions{baseBranch: "master", releaseVersion: "2.0.0-rc1"},
			expectError: true,
		},
	}

	for _, c := range cases {
		repo := &githubRepoInfo{owner: DefaultIntegreatlyOperatorRepo, repo: DefaultIntegreatlyOperatorRepo}
		t.Run(c.description, func(t *testing.T) {
			err := DoMergeRelease(context.TODO(), c.client, repo, c.opts)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
