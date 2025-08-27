package httpclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type (
	// ResponseBuilder provides fluent interface for handling HTTP responses.
	// Define status code processing, parse bodies, apply security constraints.
	//
	// Not thread-safe. Each instance handles a single response.
	ResponseBuilder struct {
		builderError error

		resp              *http.Response
		bodySizeReadLimit int64
		statusHandler     ResponseStatusHandlers
	}

	// ResponseStatusHandlers maps status codes to response handlers.
	ResponseStatusHandlers map[int]ResponseHandler

	// ResponseHandler handles HTTP responses for specific status codes.
	ResponseHandler func(*http.Response) error
)

func newResponse() *ResponseBuilder {
	return &ResponseBuilder{statusHandler: make(ResponseStatusHandlers)}
}

// BodySizeReadLimit sets maximum bytes to read from response body.
// Security feature preventing memory exhaustion attacks.
//
// Behavior:
//   - Positive: Max bytes. Larger Content-Length fails immediately.
//   - Zero: Uses Content-Length as limit.
//   - Negative: Disables limit (use with caution).
//
// Without Content-Length header, reading stops at limit.
func (b *ResponseBuilder) BodySizeReadLimit(bodySizeReadLimit int64) *ResponseBuilder {
	b.bodySizeReadLimit = bodySizeReadLimit
	return b
}

// OnStatus sets custom handler for specific HTTP status code.
// Handler called when response matches status.
func (b *ResponseBuilder) OnStatus(status int, handler ResponseHandler) *ResponseBuilder {
	b.statusHandler[status] = handler
	return b
}

// OnStatuses sets single handler for multiple status codes.
// Convenience method for shared handling logic.
func (b *ResponseBuilder) OnStatuses(statuses []int, handler ResponseHandler) *ResponseBuilder {
	for _, status := range statuses {
		b.OnStatus(status, handler)
	}
	return b
}

// SuccessOnStatus marks status codes as successful (no error).
// Convenience method for success codes without special processing.
//
// Equivalent to OnStatus with handler returning nil.
func (b *ResponseBuilder) SuccessOnStatus(statuses ...int) *ResponseBuilder {
	return b.OnStatuses(statuses, func(*http.Response) error { return nil })
}

// ErrorOnStatus sets error for specific status code.
// Useful for mapping status codes to domain-specific sentinel errors.
func (b *ResponseBuilder) ErrorOnStatus(status int, err error) *ResponseBuilder {
	return b.OnStatus(status, func(*http.Response) error { return err })
}

// ReceiveJSON parses response body as JSON for specified status code.
// Stores result in provided destination.
//
// Does not validate Content-Type header. Destination must be pointer.
func (b *ResponseBuilder) ReceiveJSON(status int, dest any) *ResponseBuilder {
	return b.OnStatus(status, func(resp *http.Response) error {
		if err := json.NewDecoder(resp.Body).Decode(&dest); err != nil {
			return fmt.Errorf("%s: unable to parse JSON response body: %w", b.formatResponseError(resp), err)
		}
		return nil
	})
}

// Error processes response with configured handlers and returns any error.
//
// Call last in chain to finalize processing:
//  1. Apply body size limits
//  2. Call status handler
//  3. Return error if no handler configured
//
// Unhandled status codes return error with request details and base64 body.
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

	return fmt.Errorf("%s: unhandled status%s", b.formatResponseError(b.resp), errSuffix)
}

// formatResponseError creates standardized error message.
// Includes method, URL, and status code for context.
func (*ResponseBuilder) formatResponseError(resp *http.Response) string {
	return fmt.Sprintf("request %s %s failed with status %d", resp.Request.Method, resp.Request.URL.String(), resp.StatusCode)
}
