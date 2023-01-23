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

// NewRequest returns a new request builder.
func NewRequest(method, endpoint string) *RequestBuilder {
	builder := &RequestBuilder{
		client: http.DefaultClient,
		method: method,
		header: make(http.Header),
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		builder.builderError = fmt.Errorf("unable to parse endpoint url %q: %v", endpointURL, err)
	} else {
		builder.url = *endpointURL
	}

	return builder
}

// RequestBuilder stores the different attributes set by the builder methods.
type RequestBuilder struct {
	builderError error

	client Doer
	method string
	url    url.URL
	header http.Header

	body          io.Reader
	bodyToMarshal any
	bodyMarshaler func(any) ([]byte, error)
}

// Client overrides the default http client with the provided one.
func (b *RequestBuilder) Client(client Doer) *RequestBuilder {
	b.client = client
	return b
}

// SetHeader replaces the value of the request header with the provided value.
func (b *RequestBuilder) SetHeader(key, value string, values ...string) *RequestBuilder {
	key = textproto.CanonicalMIMEHeaderKey(key)
	b.header[key] = append([]string{value}, values...)
	return b
}

// SetHeaders replaces the value of the request with the provided value.
// It does not replace all the request header with provided headers (it is equivalent of calling SetHeader for each provided headers).
func (b *RequestBuilder) SetHeaders(header http.Header) *RequestBuilder {
	for key, values := range header {
		b.header[key] = values
	}
	return b
}

// AddHeader appends the provided value to the provided header.
func (b *RequestBuilder) AddHeader(key, value string, values ...string) *RequestBuilder {
	key = textproto.CanonicalMIMEHeaderKey(key)
	b.header[key] = append(b.header[key], append([]string{value}, values...)...)
	return b
}

// AddHeaders appends the provided value to the provided header.
func (b *RequestBuilder) AddHeaders(header http.Header) *RequestBuilder {
	for key, values := range header {
		b.header[key] = append(b.header[key], values...)
	}
	return b
}

// SetQueryParam replaces the provided value to the provided query parameter.
func (b *RequestBuilder) SetQueryParam(key, value string, values ...string) *RequestBuilder {
	query := b.url.Query()
	query[key] = append([]string{value}, values...)
	b.url.RawQuery = query.Encode()
	return b
}

// SetQueryParams replaces the provided value to the provided query parameters.
// It does not replace all the request query parameters with provided query parameters (it is equivalent of calling SetQueryParam for each provided query parameter).
func (b *RequestBuilder) SetQueryParams(params url.Values) *RequestBuilder {
	query := b.url.Query()
	for key, values := range params {
		query[key] = values
	}

	b.url.RawQuery = query.Encode()
	return b
}

// AddQueryParam sets / appends the provided value to the provided query parameter.
func (b *RequestBuilder) AddQueryParam(key, value string, values ...string) *RequestBuilder {
	query := b.url.Query()
	for _, value := range append([]string{value}, values...) {
		query.Add(key, value)
	}

	b.url.RawQuery = query.Encode()
	return b
}

// AddQueryParams sets / appends the provided value to the provided query parameters.
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

// PathReplacer replaces any matching occurrences of the provided pattern inside the url path, with the provided replacement.
// It is useful to keep the url provided to NewRequest readable and searchable.
// Example: NewRequest("PUT", "/users/{userID}/email").PathReplacer({"{userID}", userID).
func (b *RequestBuilder) PathReplacer(pattern, replaceWith string) *RequestBuilder {
	b.url.Path = strings.ReplaceAll(b.url.Path, pattern, replaceWith)
	return b
}

// SendForm sets the provided values as url-encoded form values to the request body, with Content-Type header.
func (b *RequestBuilder) SendForm(values url.Values) *RequestBuilder {
	b.body = strings.NewReader(values.Encode())
	b.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	return b
}

// SendJSON sets the provided object, marshaled in JSON, to the request body, with Content-Type header.
func (b *RequestBuilder) SendJSON(obj any) *RequestBuilder {
	b.bodyToMarshal = obj
	b.bodyMarshaler = json.Marshal
	b.SetHeader("Content-Type", "application/json")
	return b
}

// Send sets the provided body to be used as the request body, with Content-Type octet-stream.
func (b *RequestBuilder) Send(body io.Reader) *RequestBuilder {
	b.body = body
	b.SetHeader("Content-Type", "application/octet-stream")
	return b
}

// Request builds the request.
func (b *RequestBuilder) Request(ctx context.Context) (*http.Request, error) {
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

	return req, nil
}

// Do builds the request using Request(), executes it and returns a builder to handle the response.
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
