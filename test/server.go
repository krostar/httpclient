package httpclienttest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/krostar/httpclient"
)

// Server runs a server on which one can assert requests.
type Server struct {
	do func(url.URL, httpclient.Doer, any) error
}

// NewServer creates a server.
// Provided do argument is a fonction that should perform a request.
func NewServer(do func(serverAddress url.URL, serverDoer httpclient.Doer, checkResponseFunc any) error) *Server {
	return &Server{do: do}
}

// AssertRequest performs the request and asserts.
func (srv *Server) AssertRequest(requestExpectations RequestMatcher, writeResponse func(http.ResponseWriter) error, checkResponseFunc any) error {
	cerr := make(chan error, 1)
	defer close(cerr)

	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if err := requestExpectations.MatchRequest(r); err != nil {
			cerr <- fmt.Errorf("request does not match: %v", err)
			return
		}

		if err := writeResponse(rw); err != nil {
			cerr <- fmt.Errorf("unable to write response: %v", err)
			return
		}

		cerr <- nil
	}))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	if err != nil {
		return fmt.Errorf("unable to parse url %s: %v", httpServer.URL, err)
	}

	if err := srv.do(*httpServerURL, httpServer.Client(), checkResponseFunc); err != nil {
		return fmt.Errorf("doer execution failed: %v", err)
	}

	return <-cerr
}
