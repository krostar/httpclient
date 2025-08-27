// Package httpclienttest provides testing utilities for HTTP client code.
//
// Makes testing HTTP interactions easier, more reliable, and less dependent on external services.
// Follows common testing patterns like spies, stubs, and mocks for unit test verification.
package httpclienttest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/krostar/httpclient"
)

// Server provides a testing utility that creates temporary HTTP servers for validating
// HTTP client behavior. It combines request assertion with response generation,
// allowing tests to verify that clients make expected requests and handle responses correctly.
// The server automatically starts and stops a httptest.Server for each assertion.
type Server struct {
	do func(url.URL, httpclient.Doer, any) error
}

// NewServer creates a new Server with the provided request execution function.
// The do function should contain the code under test that makes HTTP requests.
// It receives the temporary server's URL, an HTTP client (typically httptest server's client),
// and an optional checkResponseFunc parameter that can be used for additional response validation.
// The do function should return an error if the HTTP request or response handling fails.
func NewServer(do func(serverAddress url.URL, serverDoer httpclient.Doer, checkResponseFunc any) error) *Server {
	return &Server{do: do}
}

// AssertRequest creates a temporary HTTP server, executes the configured request function,
// and validates both the incoming request and outgoing response. It performs three main steps:
//
//  1. creates a httptest.Server that validates incoming requests using requestExpectations
//  2. calls the configured do function with the server URL and client
//  3. uses writeResponse to generate the HTTP response sent back to the client
//
// The requestExpectations RequestMatcher validates that the incoming HTTP request
// meets all specified criteria (method, headers, body, etc.).
//
// The writeResponse function is responsible for writing the HTTP response, including
// status code, headers, and body. It should return an error if response writing fails.
//
// The checkResponseFunc parameter is passed through to the do function and can be used
// for additional response validation logic specific to the test case.
//
// Returns an error if request validation fails, response writing fails, or the
// do function returns an error.
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
