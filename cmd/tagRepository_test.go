package cmd

import (
	"context"
	"errors"
	"github.com/integr8ly/delorean/pkg/types"
	"strings"
	"testing"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
)

func TestTagRepository(t *testing.T) {

	masterRef := "refs/heads/master"
	masterSha := "masterSha"

	cases := []struct {
		desc        string
		ghClient    services.GitService
		version     string
		flags       *tagRepositoryFlags
		expectError bool
	}{
		{
			desc: "successfully tag the repository",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					}
					return nil, nil, nil
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			version:     "2.0.0-rc1",
			flags:       &tagRepositoryFlags{branch: "master", organization: "test", repository: "test", skipPreRelease: false, olmType: types.OlmTypeRhmi},
			expectError: false,
		},
		{
			desc: "successfully tag repository with final version and skip-pre-release",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					}
					return nil, nil, nil
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			version:     "2.0.0",
			flags:       &tagRepositoryFlags{branch: "master", organization: "test", repository: "test", skipPreRelease: true, olmType: types.OlmTypeRhmi},
			expectError: false,
		},
		{
			desc: "skip tag repository with pre-release version and skip-pre-release",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					return nil, nil, errors.New("unexpected call")
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return nil, nil, errors.New("unexpected call")
				},
			},
			version:     "2.0.0-rc1",
			flags:       &tagRepositoryFlags{branch: "master", organization: "test", repository: "test", skipPreRelease: true, olmType: types.OlmTypeRhmi},
			expectError: false,
		},
		{
			desc: "fail with tag already exists",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					}
					return nil, nil, nil
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return nil, nil, errors.New("tag already exists")
				},
			},
			version:     "2.0.0-rc1",
			flags:       &tagRepositoryFlags{branch: "master", organization: "test", repository: "test", skipPreRelease: false, olmType: types.OlmTypeRhmi},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := runTagRepository(context.TODO(), c.ghClient, c.version, c.flags)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
