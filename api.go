package httpclient

import (
	"context"
	"net/http"
	"net/url"
)

// API stores attributes common to multiple requests definition / responses handing.
// It helps to build requests and responses with less duplication.
type API struct {
	client        Doer
	serverAddress url.URL

	defaultRequestHeaders            http.Header
	defaultResponseHandlers          ResponseStatusHandlers
	defaultResponseBodySizeReadLimit int64
}

// NewAPI creates an API object that will use the provided client to perform all requests with.
// The provided address will be the base url used for each request.
func NewAPI(client Doer, serverAddress url.URL) *API {
	return &API{
		client:                           client,
		serverAddress:                    serverAddress,
		defaultRequestHeaders:            make(http.Header),
		defaultResponseHandlers:          make(map[int]ResponseHandler),
		defaultResponseBodySizeReadLimit: 1 << 16, //nolint:gomnd // 64ko
	}
}

// Clone returns a deep clone of the original API.
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

// WithRequestHeaders sets headers that will be sent to each request.
func (api *API) WithRequestHeaders(headers http.Header) *API {
	for key, value := range headers {
		api.defaultRequestHeaders[key] = value
	}
	return api
}

// WithResponseHandler sets a response handler that will be used by default (unless override) for the provided status.
func (api *API) WithResponseHandler(status int, handler ResponseHandler) *API {
	api.defaultResponseHandlers[status] = handler
	return api
}

// WithResponseBodySizeReadLimit sets the maximum sized read for any API response.
// A value of 64ko is set by default. See ResponseBuilder.BodySizeReadLimit for more details on the provided value.
func (api *API) WithResponseBodySizeReadLimit(bodySizeReadLimit int64) *API {
	api.defaultResponseBodySizeReadLimit = bodySizeReadLimit
	return api
}

// URL returns the absolute URL to query the server.
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

// Head creates a HEAD request builder.
func (api *API) Head(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodHead, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Get creates a GET request builder.
func (api *API) Get(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodGet, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Post creates a POST request builder.
func (api *API) Post(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPost, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Put creates a PUT request builder.
func (api *API) Put(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPut, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Patch creates a PATCH request builder.
func (api *API) Patch(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodPatch, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Delete creates a DELETE request builder.
func (api *API) Delete(endpoint string) *RequestBuilder {
	return NewRequest(http.MethodDelete, api.URL(endpoint).String()).Client(api.client).SetHeaders(api.defaultRequestHeaders)
}

// Do performs the requests and returns a response builder.
// It differs from NewRequest().Do() by adding defaults to the request / response.
func (api *API) Do(ctx context.Context, req *RequestBuilder) *ResponseBuilder {
	resp := req.Do(ctx)
	resp = resp.BodySizeReadLimit(api.defaultResponseBodySizeReadLimit)

	for httpStatus, responseHandler := range api.defaultResponseHandlers {
		resp = resp.OnStatus(httpStatus, responseHandler)
	}

	return resp
}

// Execute calls the server, handle the response with default response handlers, and returns the associated error.
// To customize the response handling, use api.Do().
func (api *API) Execute(ctx context.Context, req *RequestBuilder) error {
	return api.Do(ctx, req).Error()
}
