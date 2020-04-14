package quay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

const (
	baseURL = "https://quay.io/api/v1/"
)

// Tag represents an image tag on quay.io
type Tag struct {
	Name           *string `json:"name,omitempty"`
	ImageId        *string `json:"image_id,omitempty"`
	ManifestDigest *string `json:"manifest_digest,omitempty"`
	DockerImageId  *string `json:"docker_image_id,omitempty"`
}

func (t *Tag) String() string {
	return fmt.Sprintf("name:%s,image_id:%s,manifest_digest:%s,docker_image_id:%s", *t.Name, *t.ImageId, *t.ManifestDigest, *t.DockerImageId)
}

// TagList represents a list of image tags on quay.io
type TagList struct {
	Tags []Tag `json:"tags,omitempty"`
	Page *int  `json:"page,omitempty"`
}

// ListTagsOptions specifies the options to use when listing image tags
type ListTagsOptions struct {
	OnlyActiveTags bool   `url:"onlyActiveTags,omitempty"`
	Page           int    `url:"page,omitempty"`
	Limit          int    `url:"limit,omitempty"`
	SpecificTag    string `url:"specificTag,omitempty"`
}

// ChangTag specifies the options to change an image tag
type ChangTag struct {
	ManifestDigest string `json:"manifest_digest,omitempty"`
	Expiration     int64  `json:"expiration,omitempty"`
}

// TagsServiceManager manages the tags of a repository.
// See https://docs.quay.io/api/swagger/#!/tag/
type TagsServiceManager interface {
	List(ctx context.Context, repository string, options *ListTagsOptions) (*TagList, *http.Response, error)
	Change(ctx context.Context, repository string, tag string, input *ChangTag) (*Tag, *http.Response, error)
}

// ManifestLabel represents a label for an image
type ManifestLabel struct {
	Id    *string `json:"id,omitempty"`
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

// ManifestLabelsList represents a list of ManifestLabels
type ManifestLabelsList struct {
	Labels []ManifestLabel `json:"labels,omitempty"`
}

// ListManifestLabelsOptions specifies the options when list manifest labels
type ListManifestLabelsOptions struct {
	Filter string `url:"filter,omitempty"`
}

// ManifestsServiceManager manages the manifests of a repository
// See https://docs.quay.io/api/swagger/#!/manifest
type ManifestsServiceManager interface {
	ListLabels(ctx context.Context, repository string, manifestRef string, options *ListManifestLabelsOptions) (*ManifestLabelsList, *http.Response, error)
}

// Client represents the client to access quay.io
type Client struct {
	client    *http.Client
	BaseURL   *url.URL
	common    service
	Tags      TagsServiceManager
	Manifests ManifestsServiceManager
}

// NewRequest builds a new http request to send to quay.io
// The relative path should be specified without a preceding slash
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}
	if strings.HasPrefix(urlStr, "/") {
		return nil, fmt.Errorf("relative path must not have a preceding slash: %q", urlStr)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// Do sends the request to quay.io and parse the response.  The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("context must be non-nil")
	}
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		return nil, err
	}
	defer resp.Body.Close()

	err = checkResponse(resp)
	if err != nil {
		return resp, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			decErr := json.NewDecoder(resp.Body).Decode(v)
			if decErr == io.EOF {
				decErr = nil // ignore EOF errors caused by empty response body
			}
			if decErr != nil {
				err = decErr
			}
		}
	}
	return resp, err
}

func checkResponse(resp *http.Response) error {
	if c := resp.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	return fmt.Errorf("http error: url = %q; status = %d", resp.Request.URL, resp.StatusCode)
}

type service struct {
	client *Client
}

// TagsService implements TagsServiceManager
type TagsService service

// ManifestsService implements ManifestsServiceManager
type ManifestsService service

// NewClient builds a new quay.io client
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	baseURL, _ := url.Parse(baseURL)
	c := &Client{
		client:  httpClient,
		BaseURL: baseURL,
	}
	c.common.client = c
	c.Tags = (*TagsService)(&c.common)
	c.Manifests = (*ManifestsService)(&c.common)
	return c
}

// List the image tags of a repo.
// See https://docs.quay.io/api/swagger/#!/tag/listRepoTags
func (t *TagsService) List(ctx context.Context, repository string, options *ListTagsOptions) (*TagList, *http.Response, error) {
	u := fmt.Sprintf("repository/%v/tag", repository)
	u, err := addOptions(u, options)
	if err != nil {
		return nil, nil, err
	}
	req, err := t.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	tags := new(TagList)
	resp, err := t.client.Do(ctx, req, tags)
	if err != nil {
		return nil, nil, err
	}
	return tags, resp, err
}

// Change an image tag on the repo.
// See https://docs.quay.io/api/swagger/#!/tag/changeTag
func (t *TagsService) Change(ctx context.Context, repository string, tag string, input *ChangTag) (*Tag, *http.Response, error) {
	u := fmt.Sprintf("repository/%v/tag/%v", repository, tag)
	req, err := t.client.NewRequest("PUT", u, input)
	if err != nil {
		return nil, nil, err
	}
	i := new(Tag)
	resp, err := t.client.Do(ctx, req, i)
	if err != nil {
		return nil, nil, err
	}
	return i, resp, err
}

// Get labels for the given manifest of an image
// See https://docs.quay.io/api/swagger/#!/manifest/listManifestLabels
func (m *ManifestsService) ListLabels(ctx context.Context, repository string, manifestRef string, options *ListManifestLabelsOptions) (*ManifestLabelsList, *http.Response, error) {
	u := fmt.Sprintf("repository/%v/manifest/%v/labels", repository, manifestRef)
	u, err := addOptions(u, options)
	if err != nil {
		return nil, nil, err
	}
	req, err := m.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}
	labels := new(ManifestLabelsList)
	resp, err := m.client.Do(ctx, req, labels)
	if err != nil {
		return nil, nil, err
	}
	return labels, resp, err
}

func addOptions(s string, opts interface{}) (string, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}
