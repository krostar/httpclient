package httpclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type (
	// ResponseStatusHandlers maps response http status to response handler.
	ResponseStatusHandlers map[int]ResponseHandler

	// ResponseHandler defines the signature of the function called to handle a response.
	ResponseHandler func(*http.Response) error

	// ResponseBuilder stores the different attributes set by the builder methods.
	ResponseBuilder struct {
		builderError error

		resp              *http.Response
		bodySizeReadLimit int64
		statusHandler     ResponseStatusHandlers
	}
)

func newResponse() *ResponseBuilder {
	return &ResponseBuilder{statusHandler: make(ResponseStatusHandlers)}
}

// BodySizeReadLimit limits the maximum amount of octets to be read in the response.
// Server responses will be considered invalid if the read limit is less than the response content-length.
// Zero value is equivalent of setting bodySizeReadLimit = resp.ContentLength.
// Negative value disables checks and limitations.
func (b *ResponseBuilder) BodySizeReadLimit(bodySizeReadLimit int64) *ResponseBuilder {
	b.bodySizeReadLimit = bodySizeReadLimit
	return b
}

// OnStatus sets the provided handler to be called if the response http status is the provided status.
func (b *ResponseBuilder) OnStatus(status int, handler ResponseHandler) *ResponseBuilder {
	b.statusHandler[status] = handler
	return b
}

// OnStatuses sets the provided handler to be called if the response http status is any of the provided statuses.
func (b *ResponseBuilder) OnStatuses(statuses []int, handler ResponseHandler) *ResponseBuilder {
	for _, status := range statuses {
		b.OnStatus(status, handler)
	}
	return b
}

// SuccessOnStatus sets the provided statuses handler to return no errors if the response http status is the provided statuses.
func (b *ResponseBuilder) SuccessOnStatus(statuses ...int) *ResponseBuilder {
	return b.OnStatuses(statuses, func(*http.Response) error { return nil })
}

// ErrorOnStatus sets the provided err to be returned if the response http status is the provided status.
func (b *ResponseBuilder) ErrorOnStatus(status int, err error) *ResponseBuilder {
	return b.OnStatus(status, func(*http.Response) error { return err })
}

// ReceiveJSON parses the response body as JSON (without caring about ContentType header), and sets the result in the provided destination.
func (b *ResponseBuilder) ReceiveJSON(status int, dest any) *ResponseBuilder {
	return b.OnStatus(status, func(resp *http.Response) error {
		if err := json.NewDecoder(resp.Body).Decode(&dest); err != nil {
			return fmt.Errorf("%s: unable to parse JSON response body: %w", b.formatResponseError(resp), err)
		}
		return nil
	})
}

// Error applies all the configured attributes on the request's response.
func (b *ResponseBuilder) Error() error {
	if b.resp != nil && b.resp.Body != nil {
		body := b.resp.Body
		defer func() { _ = body.Close() }()
	}

	if b.builderError != nil {
		return b.builderError
	}

	if b.bodySizeReadLimit >= 0 {
		readLimit := b.bodySizeReadLimit

		switch {
		case b.resp.ContentLength < 0:
		case readLimit == 0:
			readLimit = b.resp.ContentLength
		case readLimit > b.resp.ContentLength:
			readLimit = b.resp.ContentLength
		case readLimit < b.resp.ContentLength:
			return fmt.Errorf("%s: content length %d is above read limit %d", b.formatResponseError(b.resp), b.resp.ContentLength, readLimit)
		}

		b.resp.Body = io.NopCloser(io.LimitReader(b.resp.Body, readLimit))
	}

	if statusHandler, exists := b.statusHandler[b.resp.StatusCode]; exists {
		return statusHandler(b.resp)
	}

	var errSuffix string
	if body, _ := io.ReadAll(b.resp.Body); len(body) > 0 {
		errSuffix += " with b64 body " + base64.StdEncoding.EncodeToString(body)
	}
	return fmt.Errorf("%s: unhandled request status%s", b.formatResponseError(b.resp), errSuffix)
}

func (*ResponseBuilder) formatResponseError(resp *http.Response) string {
	return fmt.Sprintf("request %s %s failed with status %d", resp.Request.Method, resp.Request.URL.String(), resp.StatusCode)
}
