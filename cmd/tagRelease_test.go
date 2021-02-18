package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/quay"
	"github.com/integr8ly/delorean/pkg/services"
	"github.com/integr8ly/delorean/pkg/types"
)

type mockGitService struct {
	getRefFunc    func(ctx context.Context, owner string, repo string, ref string) ([]*github.Reference, *github.Response, error)
	createRefFunc func(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
}

func (m *mockGitService) GetRefs(ctx context.Context, owner string, repo string, ref string) ([]*github.Reference, *github.Response, error) {
	if m.getRefFunc != nil {
		return m.getRefFunc(ctx, owner, releaseVersion, ref)
	}
	panic("implement me")
}

func (m *mockGitService) CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error) {
	if m.createRefFunc != nil {
		return m.createRefFunc(ctx, owner, repo, ref)
	}
	panic("implement me")
}

type mockTagsService struct {
	listFunc   func(ctx context.Context, repository string, options *quay.ListTagsOptions) (*quay.TagList, *http.Response, error)
	changeFunc func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (*http.Response, error)
}

func (m *mockTagsService) List(ctx context.Context, repository string, options *quay.ListTagsOptions) (*quay.TagList, *http.Response, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, repository, options)
	}
	panic("implement me")
}

func (m mockTagsService) Change(ctx context.Context, repository string, tag string, input *quay.ChangTag) (*http.Response, error) {
	if m.changeFunc != nil {
		return m.changeFunc(ctx, repository, tag, input)
	}
	panic("implement me")
}

type mockManifestService struct {
	listLabelsFunc func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (*quay.ManifestLabelsList, *http.Response, error)
}

func (m *mockManifestService) ListLabels(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (*quay.ManifestLabelsList, *http.Response, error) {
	if m.listLabelsFunc != nil {
		return m.listLabelsFunc(ctx, repository, manifestRef, options)
	}
	panic("implement me")
}

func TestDoTagRelease(t *testing.T) {
	baseUrl, _ := url.Parse("https://example.com")

	masterRef := "refs/heads/master"
	masterSha := "masterSha"
	tagRefRC1 := "refs/tags/2.0.0-rc1"
	tagShaRC1 := "tagShaRC1"
	tagRefRC2 := "refs/tags/2.0.0-rc2"
	tagShaRC2 := "tagShaRC2"
	tagRefRC3 := "refs/tags/2.0.0-rc3"
	tagShaRC3 := "tagShaRC3"
	tagRefFinal := "refs/tags/2.0.0"

	testTagName := "test"
	testTagDigest := "testdigest"
	labelKey := commitIDLabelFilter
	quayRepos := fmt.Sprintf("%s,%s,%s:latest-staging", DefaultIntegreatlyOperatorQuayRepo, DefaultIntegreatlyOperatorTestQuayRepo, DefaultIntegreatlyOperatorTestQuayRepo)
	cases := []struct {
		desc              string
		ghClient          services.GitService
		quayClient        *quay.Client
		tagReleaseOptions *tagReleaseOptions
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						requestTag := options.SpecificTag
						if requestTag == "master" {
							return &quay.TagList{
								Tags: []quay.Tag{
									quay.Tag{
										Name:           &requestTag,
										ManifestDigest: &testTagDigest,
									},
								},
							}, nil, nil
						} else {
							return &quay.TagList{
								Tags: []quay.Tag{},
							}, nil, nil
						}
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &masterSha,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						requestTag := options.SpecificTag
						if requestTag == "2.0.0-rc3" {
							return &quay.TagList{
								Tags: []quay.Tag{
									quay.Tag{
										Name:           &requestTag,
										ManifestDigest: &testTagDigest,
									},
								},
							}, nil, nil
						} else {
							return &quay.TagList{
								Tags: []quay.Tag{},
							}, nil, nil
						}
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &tagShaRC3,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0", branch: "master", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						requestTag := options.SpecificTag
						if requestTag == "master" {
							return &quay.TagList{
								Tags: []quay.Tag{
									quay.Tag{
										Name:           &requestTag,
										ManifestDigest: &testTagDigest,
									},
								},
							}, nil, nil
						} else {
							return &quay.TagList{
								Tags: []quay.Tag{},
							}, nil, nil
						}
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &masterSha,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0", branch: "master", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
		{
			desc: "success for osde2e tag release",
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						requestTag := options.SpecificTag
						if requestTag == "osde2e-master" {
							return &quay.TagList{
								Tags: []quay.Tag{
									quay.Tag{
										Name:           &requestTag,
										ManifestDigest: &testTagDigest,
									},
								},
							}, nil, nil
						} else {
							return &quay.TagList{
								Tags: []quay.Tag{},
							}, nil, nil
						}
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &masterSha,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0", branch: "master", wait: false, quayRepos: "integreatly/integreatly-operator-test-harness:osde2e-rhmi", olmType: types.OlmTypeRhmi, sourceTag: "osde2e-master"},
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						requestTag := options.SpecificTag
						if requestTag == "2.0" {
							return &quay.TagList{
								Tags: []quay.Tag{
									quay.Tag{
										Name:           &requestTag,
										ManifestDigest: &testTagDigest,
									},
								},
							}, nil, nil
						} else {
							return &quay.TagList{
								Tags: []quay.Tag{},
							}, nil, nil
						}
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &masterSha,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.1-rc1", branch: "release-v2.0", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
		{
			desc: "should fail if git tag exists with a different commit SHA",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					sha2 := "anothersha"
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
							Object: &github.GitObject{
								SHA: &masterSha,
							},
						}}, nil, nil
					} else {
						return []*github.Reference{{
							Object: &github.GitObject{
								SHA: &sha2,
							},
						}}, nil, nil
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
			expectError:       true,
		},
		{
			desc: "should fail if image doesn't exist with the expected git commit",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference []*github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return []*github.Reference{{
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
			quayClient: &quay.Client{
				BaseURL: baseUrl,
				Tags: &mockTagsService{
					listFunc: func(ctx context.Context, repository string, options *quay.ListTagsOptions) (list *quay.TagList, response *http.Response, err error) {
						return &quay.TagList{
							Tags: []quay.Tag{
								quay.Tag{
									Name:           &testTagName,
									ManifestDigest: &testTagDigest,
								},
							},
						}, nil, nil
					},
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (response *http.Response, err error) {
						return nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					sha2 := "anothersha"
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &sha2,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false, quayRepos: quayRepos, olmType: types.OlmTypeRhmi},
			expectError:       true,
		},
		{
			desc: "success but no image tags should be created",
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
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false, quayRepos: "", olmType: types.OlmTypeRhmi},
			expectError:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gitRepo := &githubRepoInfo{repo: DefaultIntegreatlyOperatorRepo, owner: DefaultIntegreatlyGithubOrg}
			err := DoTagRelease(context.TODO(), c.ghClient, gitRepo, c.quayClient, c.tagReleaseOptions)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
