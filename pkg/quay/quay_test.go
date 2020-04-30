package quay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
)

const (
	// baseURLPath is a non-empty Client.BaseURL path to use during tests,
	// to ensure relative URLs are used for all endpoints. See issue #752.
	baseURLPath = "/api/v1"
)

type values map[string]string

// setup sets up a test HTTP server along with a quay.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() (client *Client, mux *http.ServeMux, serverURL string, teardown func()) {
	// mux is the HTTP request multiplexer used with the test server.
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative.
	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)

	// client is the Quay client being tested and is
	// configured to use test server.
	client = NewClient(nil)
	url, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = url

	return client, mux, server.URL, server.Close
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testFormValues(t *testing.T, r *http.Request, values values) {
	t.Helper()
	want := url.Values{}
	for k, v := range values {
		want.Set(k, v)
	}

	r.ParseForm()
	if got := r.Form; !reflect.DeepEqual(got, want) {
		t.Errorf("Request parameters: %v, want %v", got, want)
	}
}

func String(s string) *string {
	return &s
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil)

	if got, want := c.BaseURL.String(), baseURL; got != want {
		t.Errorf("NewClient BaseURL is %v, want %v", got, want)
	}

	c2 := NewClient(nil)
	if c.client == c2.client {
		t.Error("NewClient returned same http.Clients, but they should differ")
	}
}

func TestNewRequest(t *testing.T) {
	c := NewClient(nil)
	n := "test"
	inURL, outURL := "foo", baseURL+"foo"
	inBody, outBody := &Tag{Name: &n}, `{"name":"test"}`+"\n"
	req, _ := c.NewRequest("GET", inURL, inBody)

	// test that relative URL was expanded
	if got, want := req.URL.String(), outURL; got != want {
		t.Errorf("NewRequest(%q) URL is %v, want %v", inURL, got, want)
	}

	// test that body was JSON encoded
	body, _ := ioutil.ReadAll(req.Body)
	if got, want := string(body), outBody; got != want {
		t.Errorf("NewRequest(%q) Body is %v, want %v", inBody, got, want)
	}
}

func TestDo(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	type foo struct {
		A string
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := client.NewRequest("GET", ".", nil)
	body := new(foo)
	client.Do(context.Background(), req, body)

	want := &foo{"a"}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("Response body = %v, want %v", body, want)
	}
}

func TestDo_nilContext(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	req, _ := client.NewRequest("GET", ".", nil)
	_, err := client.Do(nil, req, nil)

	if !reflect.DeepEqual(err, errors.New("context must be non-nil")) {
		t.Errorf("Expected context must be non-nil error")
	}
}

func TestDo_httpError(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})

	req, _ := client.NewRequest("GET", ".", nil)
	resp, err := client.Do(context.Background(), req, nil)

	if err == nil {
		t.Fatal("Expected HTTP 400 error, got no error.")
	}
	if resp.StatusCode != 400 {
		t.Errorf("Expected HTTP 400 error, got %d status code.", resp.StatusCode)
	}
}

func TestTagsService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repository/testorg/testrepo/tag", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"onlyActiveTags": "true",
			"page":           "1",
			"limit":          "100",
			"specificTag":    "master",
		})
		fmt.Fprint(w, `{"tags":[{"name":"master"}]}`)
	})

	opt := &ListTagsOptions{
		OnlyActiveTags: true,
		Page:           1,
		Limit:          100,
		SpecificTag:    "master",
	}
	tags, _, err := client.Tags.List(context.Background(), "testorg/testrepo", opt)
	if err != nil {
		t.Errorf("Tags.List returned error: %v", err)
	}

	want := &TagList{
		Tags: []Tag{
			{
				Name: String("master"),
			},
		},
	}
	if !reflect.DeepEqual(tags, want) {
		t.Errorf("Tags.List returned %+v, want %+v", tags, want)
	}
}

func TestTagsService_Change(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &ChangTag{
		ManifestDigest: "testdiagest",
		Expiration:     0,
	}
	mux.HandleFunc("/repository/testorg/testrepo/tag/master", func(w http.ResponseWriter, r *http.Request) {
		v := new(ChangTag)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PUT")
		if !reflect.DeepEqual(v, input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"name":"master"}`)
	})

	_, err := client.Tags.Change(context.Background(), "testorg/testrepo", "master", input)
	if err != nil {
		t.Errorf("Tags.Change returned error: %v", err)
	}
}

func TestManifestsService_ListLabels(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repository/testorg/testrepo/manifest/testmanifestid/labels", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"filter": "filter.test",
		})
		fmt.Fprint(w, `{"labels":[{"id":"filter.test","key":"testkey","value":"testvalue"}]}`)
	})

	opt := &ListManifestLabelsOptions{
		Filter: "filter.test",
	}
	labels, _, err := client.Manifests.ListLabels(context.Background(), "testorg/testrepo", "testmanifestid", opt)
	if err != nil {
		t.Errorf("Manifests.ListLabels returned error: %v", err)
	}

	want := &ManifestLabelsList{
		Labels: []ManifestLabel{
			{
				Id:    String("filter.test"),
				Key:   String("testkey"),
				Value: String("testvalue"),
			},
		},
	}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("Manifests.ListLabels returned %+v, want %+v", labels, want)
	}

}
