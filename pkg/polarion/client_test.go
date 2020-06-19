package polarion

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

const projectID = "RedHatManagedIntegration"

const (
	// baseURLPath is a non-empty Client.BaseURL path to use during tests,
	// to ensure relative URLs are used for all endpoints. See issue #752.
	baseURLPath = "/polarion/ws/services"
)

// setup sets up a test HTTP server along with a quay.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup() (endpoint *url.URL, mux *http.ServeMux, teardown func()) {
	// mux is the HTTP request multiplexer used with the test server.
	mux = http.NewServeMux()

	// We want to ensure that tests catch mistakes where the endpoint URL is
	// specified as absolute rather than relative.
	h := http.NewServeMux()
	h.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	h.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		http.Error(w, "Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(h)

	url, _ := url.Parse(server.URL + baseURLPath)

	return url, mux, server.Close
}

func testRequest(t *testing.T, r *http.Request, expected string) {
	t.Helper()

	if got := r.Method; got != "POST" {
		t.Errorf("the request method should be 'POST' but got '%s'", got)
	}

	got, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("failed to parse the request body with error: %s", err)
	}

	if string(got) != expected {
		t.Errorf("the request body should match the expected body:\n  got: %s\n  expected: %s", string(got), expected)
	}
}

func TestLogIn(t *testing.T) {
	cases := []struct {
		description string
		test        func(t *testing.T)
	}{{
		description: "should login",
		test: func(t *testing.T) {
			endpoint, mux, teardown := setup()
			defer teardown()

			mux.HandleFunc("/"+string(sessionService), func(w http.ResponseWriter, r *http.Request) {
				testRequest(t, r, `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ses="http://ws.polarion.com/SessionWebService-impl"><soapenv:Header></soapenv:Header><soapenv:Body><ses:logIn><ses:userName>test</ses:userName><ses:password>passw</ses:password></ses:logIn></soapenv:Body></soapenv:Envelope>`)
				fmt.Fprint(w, `<Envelope><Header><sessionID>fakesessionid</sessionID></Header></Envelope>`)
			})

			polarion := NewClient(endpoint.String(), false)

			r, err := polarion.LogIn("test", "passw")
			if err != nil {
				t.Fatalf("LogIn failed with error: %s", err)
			}

			if expected := "fakesessionid"; r.Header.SessionID != expected {
				t.Fatalf("the response sessionID should be '%s' but got '%s'", expected, r.Header.SessionID)
			}
		},
	}, {
		description: "should fail to login",
		test: func(t *testing.T) {
			endpoint, mux, teardown := setup()
			defer teardown()

			mux.HandleFunc("/"+string(sessionService), func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<Envelope><Body><Fault><faultcode>some error</faultcode></Fault></Body></Envelope>`)
			})

			polarion := NewClient(endpoint.String(), false)

			_, err := polarion.LogIn("test", "passw")
			if err == nil {
				t.Fatalf("LogIn should fails with error")
			}
		},
	},
	}

	for _, c := range cases {
		t.Run(c.description, c.test)
	}
}
