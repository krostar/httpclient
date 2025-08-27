package httpclienttest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	gocmp "github.com/google/go-cmp/cmp"

	"github.com/krostar/httpclient"
)

// RequestMatcher defines an interface for validating HTTP requests against predefined criteria.
// Implementations should return nil if the request matches all expected conditions,
// or a descriptive error if any condition fails. This interface is commonly used
// with DoerStub to ensure that stubbed responses are only returned for expected requests.
type RequestMatcher interface {
	MatchRequest(req *http.Request) error
}

// RequestMatcherBuilder provides a fluent interface for building complex HTTP request
// assertions. It implements the RequestMatcher interface and allows chaining multiple
// validation criteria such as HTTP method, URL components, headers, and body content.
// Each method adds a new assertion that will be checked when MatchRequest is called.
type RequestMatcherBuilder struct {
	assertions []func(*http.Request) error
}

// NewRequestMatcherBuilder creates a new empty RequestMatcherBuilder with no assertions.
// Use the builder's methods to add specific validation criteria for HTTP requests.
func NewRequestMatcherBuilder() *RequestMatcherBuilder {
	return new(RequestMatcherBuilder)
}

// Method adds an assertion that the HTTP request method must exactly match the provided method.
// The comparison is case-sensitive (e.g., "GET", "POST", "PUT").
func (b *RequestMatcherBuilder) Method(method string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.Method != method {
			return fmt.Errorf("request method %q != %q", req.Method, method)
		}
		return nil
	})

	return b
}

// URLHost adds an assertion that the request URL's host component must exactly match
// the provided host string. This includes the hostname and optional port (e.g., "example.com:8080").
func (b *RequestMatcherBuilder) URLHost(host string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.URL.Host != host {
			return fmt.Errorf("request url host %q != %q", req.URL.Host, host)
		}
		return nil
	})

	return b
}

// URLPath adds an assertion that the request URL's path component must exactly match
// the provided path string. The path should include the leading slash (e.g., "/api/users").
func (b *RequestMatcherBuilder) URLPath(path string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.URL.Path != path {
			return fmt.Errorf("request url path %q != %q", req.URL.Path, path)
		}
		return nil
	})

	return b
}

// URLQueryParamsContains adds an assertion that the request URL's query parameters
// must contain all the provided key-value pairs. Additional query parameters in the
// request are allowed. The values for each key must match exactly, including order.
func (b *RequestMatcherBuilder) URLQueryParamsContains(params url.Values) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		reqQueryParams := req.URL.Query()

		var errs []error

		for key, values := range params {
			reqQueryParamValues, ok := reqQueryParams[key]
			if !ok {
				errs = append(errs, fmt.Errorf("expected url query param key %s to be set", key))
				continue
			}

			if !slices.Equal(values, reqQueryParamValues) {
				errs = append(errs, fmt.Errorf("expected url query param key %s to be %s but is %s", key, values, reqQueryParamValues))
				continue
			}
		}

		return errors.Join(errs...)
	})

	return b
}

// HeadersContains adds an assertion that the request headers must contain all the
// provided header key-value pairs. Additional headers in the request are allowed.
// Header names are case-insensitive, but values must match exactly including order.
func (b *RequestMatcherBuilder) HeadersContains(headers http.Header) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		reqHeaders := req.Header

		var errs []error

		for key, values := range headers {
			reqHeaderValues, ok := reqHeaders[key]
			if !ok {
				errs = append(errs, fmt.Errorf("expected header key %s to be set", key))
				continue
			}

			if !slices.Equal(values, reqHeaderValues) {
				errs = append(errs, fmt.Errorf("expected header key %s to be %s but is %s", key, values, reqHeaderValues))
				continue
			}
		}

		return errors.Join(errs...)
	})

	return b
}

// BodyForm adds an assertion that the request body contains form data matching the
// provided url.Values. The request's Content-Type should be "application/x-www-form-urlencoded".
// If strict is true, the form data must match exactly with no additional fields.
// If strict is false, additional fields in the request are allowed.
func (b *RequestMatcherBuilder) BodyForm(compareWith url.Values, strict bool) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if err := httpclient.ParsePostForm(req); err != nil {
			return fmt.Errorf("unable to parse post form: %v", err)
		}

		reqPostForm := make(url.Values)
		for k, v := range req.PostForm {
			reqPostForm[k] = v
		}

		var errs []error

		for key, expectedValues := range compareWith {
			values, found := reqPostForm[key]
			if !found {
				errs = append(errs, fmt.Errorf("key %s is expected to exist but is not found", key))
				continue
			}

			delete(reqPostForm, key)

			if !slices.Equal(expectedValues, values) {
				errs = append(errs, fmt.Errorf("key %s values differ %s %s", key, expectedValues, values))
				continue
			}
		}

		if strict && len(reqPostForm) > 0 {
			for key, values := range reqPostForm {
				errs = append(errs, fmt.Errorf("remaining key found in form: %q: {%q}", key, values))
			}
		}

		return errors.Join(errs...)
	})

	return b
}

// BodyJSON adds an assertion that the request body contains JSON data that can be
// unmarshalled into the type returned by getDest() and matches compareWith exactly.
// The getDest function should return a new instance of the expected type for unmarshalling.
// If strict is true, the JSON must not contain any fields not present in the target type.
// If strict is false, additional JSON fields are ignored during unmarshalling.
func (b *RequestMatcherBuilder) BodyJSON(compareWith any, getDest func() any, strict bool) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		buf := new(bytes.Buffer)
		tee := io.TeeReader(req.Body, buf)
		req.Body = io.NopCloser(buf)

		decoder := json.NewDecoder(tee)
		if strict {
			decoder.DisallowUnknownFields()
		}

		dest := getDest()
		if err := decoder.Decode(dest); err != nil {
			return fmt.Errorf("unable to parse json: %v", err)
		}

		if diff := gocmp.Diff(dest, compareWith); diff != "" {
			return fmt.Errorf("json does not match: %s", diff)
		}

		return nil
	})

	return b
}

// MatchRequest implements the RequestMatcher interface by running all configured
// assertions against the provided HTTP request. It returns nil if all assertions
// pass, or a combined error containing details of all failed assertions.
func (b *RequestMatcherBuilder) MatchRequest(req *http.Request) error {
	var errs []error
	for _, assertion := range b.assertions {
		errs = append(errs, assertion(req))
	}
	return errors.Join(errs...)
}
