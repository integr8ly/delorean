package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/types"
)

func TestDoTagReleaseRepo(t *testing.T) {

	masterRef := "refs/heads/master"
	masterSha := "masterSha"
	tagRefRC1 := "refs/tags/2.0.0-rc1"
	tagShaRC1 := "tagShaRC1"
	tagRefRC2 := "refs/tags/2.0.0-rc2"
	tagShaRC2 := "tagShaRC2"
	tagRefRC3 := "refs/tags/2.0.0-rc3"
	tagShaRC3 := "tagShaRC3"
	tagRefFinal := "refs/tags/2.0.0"

	cases := []struct {
		desc              string
		ghClient          services.GitService
		tagReleaseOptions *tagReleaseRepoOptions
		expectError       bool
	}{
		{
			desc: "success for minor release",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			tagReleaseOptions: &tagReleaseRepoOptions{releaseVersion: "2.0.0-rc1", branch: "master", olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
		{
			desc: "success for final minor release",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					} else if strings.Index(ref, "refs/tags/") > -1 {
						return []*github.Reference{
							{
								Ref: &tagRefRC2,
								Object: &github.GitObject{
									SHA: &tagShaRC2,
								},
							},
							{
								Ref: &tagRefRC3,
								Object: &github.GitObject{
									SHA: &tagShaRC3,
								},
							},
							{
								Ref: &tagRefRC1,
								Object: &github.GitObject{
									SHA: &tagShaRC1,
								},
							},
							{
								Ref: &tagRefFinal,
								Object: &github.GitObject{
									SHA: &masterSha,
								},
							},
						}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			tagReleaseOptions: &tagReleaseRepoOptions{releaseVersion: "2.0.0", branch: "master", olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
		{
			desc: "success for final minor release no existing rc",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &masterRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			tagReleaseOptions: &tagReleaseRepoOptions{releaseVersion: "2.0.0", branch: "master", olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
		{
			desc: "success for patch release",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					releaseRef := "refs/heads/release-v2.0"
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Ref: &releaseRef,
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &masterSha,
						},
					}, nil, nil
				},
			},
			tagReleaseOptions: &tagReleaseRepoOptions{releaseVersion: "2.0.1-rc1", branch: "release-v2.0", olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gitRepo := &githubRepoInfo{repo: DefaultIntegreatlyOperatorRepo, owner: DefaultIntegreatlyGithubOrg}
			err := DoTagReleaseRepo(context.TODO(), c.ghClient, gitRepo, c.tagReleaseOptions)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
