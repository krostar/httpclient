package httpclienttest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	gocmp "github.com/google/go-cmp/cmp"
	"go.uber.org/multierr"
	"golang.org/x/exp/slices"

	"github.com/krostar/httpclient"
)

// RequestMatcher defines a way to check whenever a request match against some pre-defined rules.
type RequestMatcher interface {
	MatchRequest(req *http.Request) error
}

// RequestMatcherBuilder stores assertions and implements RequestMatcher.
type RequestMatcherBuilder struct {
	assertions []func(*http.Request) error
}

// NewRequestMatcherBuilder creates a new empty RequestMatcherBuilder.
func NewRequestMatcherBuilder() *RequestMatcherBuilder {
	return new(RequestMatcherBuilder)
}

// Method asserts that the provided method matches request.Method.
func (b *RequestMatcherBuilder) Method(method string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.Method != method {
			return fmt.Errorf("request method %q != %q", req.Method, method)
		}
		return nil
	})
	return b
}

// URLHost asserts that the provided host matches request.URL.Host.
func (b *RequestMatcherBuilder) URLHost(host string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.URL.Host != host {
			return fmt.Errorf("request url host %q != %q", req.URL.Host, host)
		}
		return nil
	})
	return b
}

// URLPath asserts that the provided path matches request.URL.Path.
func (b *RequestMatcherBuilder) URLPath(path string) *RequestMatcherBuilder {
	b.assertions = append(b.assertions, func(req *http.Request) error {
		if req.URL.Path != path {
			return fmt.Errorf("request url path %q != %q", req.URL.Path, path)
		}
		return nil
	})
	return b
}

// URLQueryParamsContains asserts that the provided url values are contained in request.URL.Query().
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

		return multierr.Combine(errs...)
	})
	return b
}

// HeadersContains asserts that the provided headers are contained in request.Header.
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

		return multierr.Combine(errs...)
	})
	return b
}

// BodyForm asserts that the provided url values are contained in request.PostForm.
// Strict parameters define whenever the request.PostForm should be exactly the provided url values or more values can exists.
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

		return multierr.Combine(errs...)
	})
	return b
}

// BodyJSON asserts that request's body is a JSON can be bound to getDest()'s output and is the same as compareWith.
// Strict parameters define whenever the body can contain unknown fields.
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

// MatchRequest implements RequestMatcher and asserts all built assertions.
func (b *RequestMatcherBuilder) MatchRequest(req *http.Request) error {
	var errs []error
	for _, assertion := range b.assertions {
		errs = append(errs, assertion(req))
	}
	return multierr.Combine(errs...)
}
