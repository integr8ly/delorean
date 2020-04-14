package cmd

import (
	"context"
	"errors"
	"github.com/google/go-github/v30/github"
	"github.com/integr8ly/delorean/pkg/quay"
	"github.com/integr8ly/delorean/pkg/services"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type mockGitService struct {
	getRefFunc    func(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
	createRefFunc func(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
}

func (m *mockGitService) GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error) {
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
	changeFunc func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (*quay.Tag, *http.Response, error)
}

func (m *mockTagsService) List(ctx context.Context, repository string, options *quay.ListTagsOptions) (*quay.TagList, *http.Response, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, repository, options)
	}
	panic("implement me")
}

func (m mockTagsService) Change(ctx context.Context, repository string, tag string, input *quay.ChangTag) (*quay.Tag, *http.Response, error) {
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
	sha := "testsha"
	testTagName := "test"
	testTagDigest := "testdigest"
	labelKey := commitIdLabelFilter
	cases := []struct {
		desc              string
		ghClient          services.GitService
		quayClient        *quay.Client
		tagReleaseOptions *tagReleaseOptions
		expectError       bool
	}{
		{
			desc: "success",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference *github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return &github.Reference{
							Object: &github.GitObject{
								SHA: &sha,
							},
						}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &sha,
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
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (tag2 *quay.Tag, response *http.Response, err error) {
						return &quay.Tag{
							Name:           &testTagName,
							ManifestDigest: &testTagDigest,
						}, nil, nil
					},
				},
				Manifests: &mockManifestService{listLabelsFunc: func(ctx context.Context, repository string, manifestRef string, options *quay.ListManifestLabelsOptions) (list *quay.ManifestLabelsList, response *http.Response, err error) {
					return &quay.ManifestLabelsList{Labels: []quay.ManifestLabel{quay.ManifestLabel{
						Key:   &labelKey,
						Value: &sha,
					}}}, nil, nil
				}},
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false},
			expectError:       false,
		},
		{
			desc: "should fail if git tag exists with a different commit SHA",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference *github.Reference, response *github.Response, err error) {
					sha2 := "anothersha"
					if strings.Index(ref, "refs/heads/") > -1 {
						return &github.Reference{
							Object: &github.GitObject{
								SHA: &sha,
							},
						}, nil, nil
					} else {
						return &github.Reference{
							Object: &github.GitObject{
								SHA: &sha2,
							},
						}, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &sha,
						},
					}, nil, nil
				},
			},
			quayClient: &quay.Client{
				BaseURL: baseUrl,
			},
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false},
			expectError:       true,
		},
		{
			desc: "should fail if image doesn't exist with the expected git commit",
			ghClient: &mockGitService{
				getRefFunc: func(ctx context.Context, owner string, repo string, ref string) (reference *github.Reference, response *github.Response, err error) {
					if strings.Index(ref, "refs/heads/") > -1 {
						return &github.Reference{
							Object: &github.GitObject{
								SHA: &sha,
							},
						}, nil, nil
					} else {
						return nil, nil, nil
					}
				},
				createRefFunc: func(ctx context.Context, owner string, repo string, ref *github.Reference) (reference *github.Reference, response *github.Response, err error) {
					return &github.Reference{
						Object: &github.GitObject{
							SHA: &sha,
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
					changeFunc: func(ctx context.Context, repository string, tag string, input *quay.ChangTag) (tag2 *quay.Tag, response *http.Response, err error) {
						return &quay.Tag{
							Name:           &testTagName,
							ManifestDigest: &testTagDigest,
						}, nil, nil
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
			tagReleaseOptions: &tagReleaseOptions{releaseVersion: "2.0.0-rc1", branch: "master", wait: false},
			expectError:       true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			gitRepo := &githubRepoInfo{repo: DefaultIntegreatlyOperatorRepo, owner: DefaultIntegreatlyGithubOrg}
			err := DoTagRelease(context.TODO(), c.ghClient, gitRepo, c.quayClient, DefaultIntegreatlyOperatorQuayRepo, c.tagReleaseOptions)
			if c.expectError && err == nil {
				t.Errorf("error should not be nil")
			} else if !c.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRetry_ok(t *testing.T) {
	i := 0
	err := Retry(1*time.Millisecond, 3*time.Millisecond, func() error {
		i = i + 1
		return nil
	})
	if err != nil {
		t.Errorf("error should be nil")
	}
	if i != 1 {
		t.Errorf("function should be only called once")
	}
}

func TestRetry_timeout(t *testing.T) {
	i := 0
	err := Retry(1*time.Millisecond, 3*time.Millisecond, func() error {
		i = i + 1
		return errors.New("error")
	})
	if err == nil {
		t.Errorf("error should not be nil")
	}
	if i < 2 {
		t.Errorf("function should be called at least twice")
	}
	if err.Error() != "timeout" {
		t.Errorf("it should be timed out")
	}
}
