package httpclient

import (
	"context"
	"net/http"
	"net/url"
)

// API stores configuration and defaults for HTTP requests to a specific service.
// It provides common attributes (headers, response handlers) applied to all requests.
//
// API instances are safe for concurrent use as they create new builders for each request.
type API struct {
	client        Doer
	serverAddress url.URL

	defaultRequestHeaders            http.Header
	defaultRequestOverrideFunc       RequestOverrideFunc
	defaultResponseHandlers          ResponseStatusHandlers
	defaultResponseBodySizeReadLimit int64
}

// NewAPI creates an API instance with the provided client and server address.
//
// The client must implement the Doer interface (http.Client does).
// Endpoint paths are appended to the server address.
func NewAPI(client Doer, serverAddress url.URL) *API {
	const defaultResponseBodySizeReadLimit = 1 << 16 // 64KB

	return &API{
		client:                           client,
		serverAddress:                    serverAddress,
		defaultRequestHeaders:            make(http.Header),
		defaultResponseHandlers:          make(map[int]ResponseHandler),
		defaultResponseBodySizeReadLimit: defaultResponseBodySizeReadLimit,
	}
}

// Clone creates a deep copy with independent configuration.
//
// Headers, response handlers, and settings are copied.
// The HTTP client is shared.
func (api *API) Clone() *API {
	clone := &API{
		client:                           api.client,
		serverAddress:                    *api.URL(""),
		defaultRequestHeaders:            make(http.Header),
		defaultResponseHandlers:          make(map[int]ResponseHandler),
		defaultResponseBodySizeReadLimit: api.defaultResponseBodySizeReadLimit,
	}

	for key, value := range api.defaultRequestHeaders {
		clone.defaultRequestHeaders[key] = value
	}

	for key, value := range api.defaultResponseHandlers {
		clone.defaultResponseHandlers[key] = value
	}

	return clone
}

// WithRequestOverrideFunc sets a function called for every request to modify
// the final http.Request before execution.
//
// Useful for authentication, signing, or consistent request modifications.
func (api *API) WithRequestOverrideFunc(overrideFunc RequestOverrideFunc) *API {
	api.defaultRequestOverrideFunc = overrideFunc
	return api
}

// WithRequestHeaders adds headers to all requests.
// Request-specific headers take precedence over API-level headers.
//
// Provided headers are merged into existing headers, not replaced.
// Common uses: authentication, user agents, ...
func (api *API) WithRequestHeaders(headers http.Header) *API {
	for key, value := range headers {
		api.defaultRequestHeaders[key] = value
	}
	return api
}

// WithResponseHandler sets a default handler for the specified status code.
// Called automatically unless overridden by request-specific handlers.
//
// Useful for consistent error handling (e.g., 401 as authentication error).
func (api *API) WithResponseHandler(status int, handler ResponseHandler) *API {
	api.defaultResponseHandlers[status] = handler
	return api
}

// WithResponseBodySizeReadLimit sets maximum bytes to read from response bodies.
// Security feature preventing memory exhaustion attacks.
//
// Behavior:
//   - Larger Content-Length causes failure
//   - Unknown Content-Length stops at limit
//   - 0 uses Content-Length as limit
//   - Negative disables limit (use with caution)
func (api *API) WithResponseBodySizeReadLimit(bodySizeReadLimit int64) *API {
	api.defaultResponseBodySizeReadLimit = bodySizeReadLimit
	return api
}

// URL constructs absolute URL by combining endpoint with server address.
func (api *API) URL(endpoint string) *url.URL {
	var user *url.Userinfo

	if api.serverAddress.User != nil {
		u := *api.serverAddress.User
		user = &u
	}

	u := api.serverAddress
	u.User = user
	u.Path += endpoint

	return &u
}

// Head creates a HEAD request builder with default headers and settings.
func (api *API) Head(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodHead, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Get creates a GET request builder with default headers and settings.
func (api *API) Get(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodGet, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Post creates a POST request builder with default headers and settings.
func (api *API) Post(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPost, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Put creates a PUT request builder with default headers and settings.
func (api *API) Put(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPut, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Patch creates a PATCH request builder with default headers and settings.
func (api *API) Patch(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPatch, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Delete creates a DELETE request builder with default headers and settings.
func (api *API) Delete(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodDelete, api.URL(endpoint).String()).
		Client(api.client).
		SetHeaders(api.defaultRequestHeaders).
		SetOverrideFunc(api.defaultRequestOverrideFunc)
}

// Do executes the request and returns a response builder with API defaults.
//
// Applies default response handlers and body size limits.
// Use for custom response handling beyond Execute().
func (api *API) Do(ctx context.Context, req *RequestBuilder) *ResponseBuilder {
	resp := req.Do(ctx)
	resp = resp.BodySizeReadLimit(api.defaultResponseBodySizeReadLimit)

	for httpStatus, responseHandler := range api.defaultResponseHandlers {
		resp = resp.OnStatus(httpStatus, responseHandler)
	}

	return resp
}

// Execute performs the request using API defaults and returns any error.
//
// Convenience method for simple success/failure handling.
// Use Do() for complex response processing.
func (api *API) Execute(ctx context.Context, req *RequestBuilder) error {
	return api.Do(ctx, req).Error()
}
