package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

// RequestBuilder provides a fluent interface for building HTTP requests.
// Configure method, URL, headers, body through method chaining.
//
// Not thread-safe. Each instance builds and executes a single request.
type RequestBuilder struct {
	builderError error

	client Doer
	method string
	url    url.URL
	header http.Header

	body          io.Reader
	bodyToMarshal any
	bodyMarshaler func(any) ([]byte, error)

	overrideFunc RequestOverrideFunc
}

// RequestOverrideFunc modifies an http.Request just before execution.
// Useful for authentication, signatures, or dynamic headers.
//
// Returns modified request and error. Errors cause request failure.
type RequestOverrideFunc func(req *http.Request) (*http.Request, error)

// NewRequest creates a RequestBuilder for the HTTP method and endpoint URL.
//
// Method should be valid (GET, POST, PUT, DELETE, etc.).
// Endpoint should be complete URL (containing scheme, host, endpoint, ...).
//
// Parse errors are captured and returned when Request() or Do() is called.
func NewRequest(method, endpoint string) *RequestBuilder {
	builder := &RequestBuilder{
		client: http.DefaultClient,
		method: method,
		header: make(http.Header),
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		builder.builderError = fmt.Errorf("unable to parse endpoint url %q: %v", endpoint, err)
	} else {
		builder.url = *endpointURL
	}

	return builder
}

// Client sets the HTTP client for request execution.
// Must implement Doer interface (http.Client does).
// Defaults to http.DefaultClient. Allows custom clients with timeouts, transports.
func (b *RequestBuilder) Client(client Doer) *RequestBuilder {
	b.client = client
	return b
}

// SetHeader sets HTTP header values, replacing existing ones.
// Header names are canonicalized.
func (b *RequestBuilder) SetHeader(key, value string, values ...string) *RequestBuilder {
	key = textproto.CanonicalMIMEHeaderKey(key)

	b.header[key] = append([]string{value}, values...)

	return b
}

// SetHeaders merges multiple headers, replacing existing values for same keys.
// Other headers remain unchanged.
//
// Equivalent to calling SetHeader for each provided header.
func (b *RequestBuilder) SetHeaders(header http.Header) *RequestBuilder {
	for key, values := range header {
		b.header[key] = values
	}
	return b
}

// AddHeader appends values to header, preserving existing ones.
// Header names are canonicalized.
// Creates header if missing, appends if exists.
func (b *RequestBuilder) AddHeader(key, value string, values ...string) *RequestBuilder {
	key = textproto.CanonicalMIMEHeaderKey(key)
	b.header[key] = append(b.header[key], append([]string{value}, values...)...)
	return b
}

// AddHeaders appends all provided header values to existing ones.
// Use SetHeaders() to replace entirely.
func (b *RequestBuilder) AddHeaders(header http.Header) *RequestBuilder {
	for key, values := range header {
		b.header[key] = append(b.header[key], values...)
	}
	return b
}

// SetQueryParam sets query parameter values, replacing existing ones.
// Multiple values can be provided for the same parameter.
func (b *RequestBuilder) SetQueryParam(key, value string, values ...string) *RequestBuilder {
	query := b.url.Query()

	query[key] = append([]string{value}, values...)
	b.url.RawQuery = query.Encode()
	return b
}

// SetQueryParams merges query parameters, replacing existing ones.
// Does not append; replaces entirely.
func (b *RequestBuilder) SetQueryParams(params url.Values) *RequestBuilder {
	query := b.url.Query()
	for key, values := range params {
		query[key] = values
	}

	b.url.RawQuery = query.Encode()
	return b
}

// AddQueryParam appends values to query parameter, preserving existing ones.
// Creates parameter if missing, appends if exists.
func (b *RequestBuilder) AddQueryParam(key, value string, values ...string) *RequestBuilder {
	query := b.url.Query()
	for _, value := range append([]string{value}, values...) {
		query.Add(key, value)
	}

	b.url.RawQuery = query.Encode()
	return b
}

// AddQueryParams appends all provided parameter values to existing ones.
// Use SetQueryParams() to replace entirely.
func (b *RequestBuilder) AddQueryParams(params url.Values) *RequestBuilder {
	query := b.url.Query()

	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	b.url.RawQuery = query.Encode()
	return b
}

// PathReplacer replaces pattern occurrences in URL path with replacement.
// Keeps URLs readable and searchable.
// Example: NewRequest("PUT", "/users/{userID}/email").PathReplacer("{userID}", userID).
func (b *RequestBuilder) PathReplacer(pattern, replaceWith string) *RequestBuilder {
	b.url.Path = strings.ReplaceAll(b.url.Path, pattern, replaceWith)
	return b
}

// SendForm sets request body to form values as application/x-www-form-urlencoded.
// Sets appropriate Content-Type header.
//
// Used for HTML forms and form-encoded API endpoints.
func (b *RequestBuilder) SendForm(values url.Values) *RequestBuilder {
	b.body = strings.NewReader(values.Encode())
	b.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	return b
}

// SendJSON sets object as JSON request body and Content-Type header.
//
// Marshaling is lazy - happens during execution, not when called.
// Object must be JSON-serializable. Marshal errors are returned during execution.
func (b *RequestBuilder) SendJSON(obj any) *RequestBuilder {
	b.bodyToMarshal = obj
	b.bodyMarshaler = json.Marshal
	b.SetHeader("Content-Type", "application/json")
	return b
}

// Send sets io.Reader as request body with Content-Type: application/octet-stream.
//
// Useful for binary data, file uploads, or custom content.
func (b *RequestBuilder) Send(body io.Reader) *RequestBuilder {
	b.body = body
	b.SetHeader("Content-Type", "application/octet-stream")
	return b
}

// SetOverrideFunc sets function called before request execution.
// Receives built request, returns modified request and error.
// Useful for authentication, signing, dynamic modifications.
func (b *RequestBuilder) SetOverrideFunc(overrideFunc RequestOverrideFunc) *RequestBuilder {
	b.overrideFunc = overrideFunc
	return b
}

// Request builds and returns the http.Request.
func (b *RequestBuilder) Request(ctx context.Context) (*http.Request, error) {
	if b.builderError != nil {
		return nil, b.builderError
	}

	if b.bodyToMarshal != nil {
		if b.body != nil {
			return nil, errors.New("body to marshal is set but body is already set")
		}

		if b.bodyMarshaler == nil {
			return nil, errors.New("body to marshal is set but body marshaller is unset")
		}

		raw, err := b.bodyMarshaler(b.bodyToMarshal)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal body: %w", err)
		}

		b.body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, b.method, b.url.String(), b.body)
	if err != nil {
		return nil, fmt.Errorf("unable to create request %s %s: %w", b.method, b.url.String(), err)
	}

	for header, value := range b.header {
		req.Header[header] = value
	}

	if b.overrideFunc != nil {
		if req, err = b.overrideFunc(req); err != nil {
			return nil, fmt.Errorf("unable to override request: %w", err)
		}
	}

	return req, nil
}

// Do builds, executes request and returns ResponseBuilder.
func (b *RequestBuilder) Do(ctx context.Context) *ResponseBuilder {
	responseBuilder := newResponse()

	req, err := b.Request(ctx)
	if err != nil {
		responseBuilder.builderError = fmt.Errorf("unable to create request: %w", err)
		return responseBuilder
	}

	resp, err := b.client.Do(req)
	if err != nil {
		responseBuilder.builderError = fmt.Errorf("unable to execute %s %s request: %w", req.Method, req.URL.String(), err)
		return responseBuilder
	}

	responseBuilder.resp = resp

	return responseBuilder
}
