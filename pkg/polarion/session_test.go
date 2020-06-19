package polarion

import (
	"fmt"
	"net/http"
	"testing"
)

func setupSession() (session *Session, mux *http.ServeMux, teardown func(), err error) {
	endpoint, mux, teardow := setup()

	mux.HandleFunc("/"+string(sessionService), func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<Envelope><Header><sessionID>fakesessionid</sessionID></Header></Envelope>`)
	})

	session, err = NewSession("test", "passw", endpoint.String(), false)
	if err != nil {
		return nil, nil, func() {}, err
	}

	return session, mux, teardow, nil
}

func TestNewSession(t *testing.T) {
	_, _, teardown, err := setupSession()
	if err != nil {
		t.Fatalf("failed to create a new session with error: %s", err)
	}
	defer teardown()
}
