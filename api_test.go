package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_NewAPI(t *testing.T) {
	api := NewAPI(nil, url.URL{})
	assert.Check(t, cmp.DeepEqual(api, &API{
		client:                           nil,
		serverAddress:                    url.URL{},
		defaultRequestHeaders:            nil,
		defaultResponseHandlers:          make(ResponseStatusHandlers),
		defaultResponseBodySizeReadLimit: 65536,
	}, gocmp.AllowUnexported(API{})))

	api = NewAPI(http.DefaultClient, url.URL{Scheme: "http", Host: "localhost"})
	assert.Check(t, cmp.DeepEqual(api, &API{
		client:                           http.DefaultClient,
		serverAddress:                    url.URL{Scheme: "http", Host: "localhost"},
		defaultRequestHeaders:            nil,
		defaultResponseHandlers:          make(ResponseStatusHandlers),
		defaultResponseBodySizeReadLimit: 65536,
	}, gocmp.AllowUnexported(API{})))
}

func Test_API_URL(t *testing.T) {
	t.Run("", func(t *testing.T) {
		api := NewAPI(http.DefaultClient, url.URL{
			Scheme: "http",
			Host:   "localhost",
		})
		endpointURL := api.URL("/foo/bar")

		assert.Equal(t, "http://localhost/foo/bar", endpointURL.String())
	})

	t.Run("every fields are cloned", func(t *testing.T) {
		serverURL, err := url.Parse("https://foo@localhost:8080")
		assert.NilError(t, err)

		api := NewAPI(http.DefaultClient, *serverURL)
		endpointURL := api.URL("/foo/bar")

		assert.Check(t, serverURL.User != endpointURL.User)
		assert.Check(t, *serverURL.User == *endpointURL.User)
	})
}

func Test_API_Method(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Check(t, r.URL.Path == "/foo/bar")
		rw.Header().Set("hello", r.Header.Get("hello"))
		rw.Header().Set("method", r.Method)
		rw.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()
	httpServerURL, err := url.Parse(httpServer.URL)
	assert.NilError(t, err)

	api := NewAPI(httpServer.Client(), *httpServerURL).WithRequestHeaders(http.Header{"hello": []string{"world"}})

	for httpMethod, apiMethod := range map[string]func(string) *RequestBuilder{
		http.MethodHead:   api.Head,
		http.MethodGet:    api.Get,
		http.MethodPost:   api.Post,
		http.MethodPut:    api.Put,
		http.MethodPatch:  api.Patch,
		http.MethodDelete: api.Delete,
		// http.MethodConnect do not have an API method, yet ?
		// http.MethodOptions do not have an API method, yet ?
		// http.MethodTrace   do not have an API method, yet ?
	} {
		httpMethod, apiMethod := httpMethod, apiMethod

		t.Run(httpMethod, func(t *testing.T) {
			t.Run("run with defaults", func(t *testing.T) {
				assert.NilError(t, api.
					Do(context.Background(), apiMethod("/foo/bar")).
					OnStatus(http.StatusOK, func(resp *http.Response) error {
						assert.Check(t, resp.Header.Get("hello") == "world", "header 'hello' should be set by default")
						assert.Check(t, resp.Header.Get("method") == httpMethod)
						return nil
					}).
					Error(),
				)
			})

			t.Run("make sure all api defaults are set in builder and not after building", func(t *testing.T) {
				t.Run("client should be overridable", func(t *testing.T) {
					defer func() {
						panicked := recover()
						assert.Check(t, panicked != nil, "client should have been overridden with nil, creating a panic")
					}()
					assert.NilError(t, api.Execute(context.Background(), apiMethod("/foo/bar").Client(nil)))
				})

				t.Run("headers defaults should be overridable", func(t *testing.T) {
					assert.NilError(t, api.
						Do(context.Background(), apiMethod("/foo/bar").SetHeader("hello", "notworld")).
						OnStatus(http.StatusOK, func(resp *http.Response) error {
							assert.Check(t, resp.Header.Get("hello") == "notworld", "header 'hello' should have been overridden")
							return nil
						}).
						Error(),
					)
				})
			})
		})
	}
}

func Test_API_Do(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("hello", r.Header.Get("hello"))
		rw.Header().Set("method", r.Method)
		if r.URL.Path == "/teapot" {
			rw.WriteHeader(http.StatusTeapot)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	assert.NilError(t, err)

	api := NewAPI(httpServer.Client(), *httpServerURL).
		WithRequestHeaders(http.Header{"hello": []string{"world"}}).
		WithResponseHandler(http.StatusTeapot, func(*http.Response) error { return nil }).
		WithResponseBodySizeReadLimit(121212)

	t.Run("should not set defaults in request", func(t *testing.T) {
		t.Run("client", func(t *testing.T) {
			defer func() {
				panicked := recover()
				assert.Check(t, panicked != nil, "default client is non-nil, but request chose a nil client, creating a panic")
			}()
			_ = api.Do(context.Background(), NewRequest(http.MethodGet, httpServerURL.String()).Client(nil))
		})

		t.Run("header", func(t *testing.T) {
			assert.NilError(t, api.
				Do(context.Background(), NewRequest(http.MethodGet, httpServerURL.String()).SetHeader("hello", "notworld")).
				OnStatus(http.StatusOK, func(resp *http.Response) error {
					assert.Check(t, resp.Header.Get("hello") == "notworld")
					return nil
				}).
				Error(),
			)
		})
	})

	t.Run("should set defaults in response", func(t *testing.T) {
		t.Run("max body read size", func(t *testing.T) {
			assert.Equal(t, int64(121212), api.Do(context.Background(), NewRequest(http.MethodGet, httpServerURL.String())).bodySizeReadLimit)
		})

		t.Run("status not handled by default", func(t *testing.T) {
			assert.ErrorContains(t, api.Do(context.Background(), api.Get("/")).Error(), "unhandled request status")
		})

		t.Run("status handled by default", func(t *testing.T) {
			assert.NilError(t, api.Do(context.Background(), api.Get("/teapot")).Error())
		})
	})
}

func Test_API_Execute(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusOK) }))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	assert.NilError(t, err)

	t.Run("request and response are handled with default", func(t *testing.T) {
		api := NewAPI(httpServer.Client(), *httpServerURL).WithResponseHandler(http.StatusOK, func(*http.Response) error { return nil })

		assert.NilError(t, api.Execute(context.Background(), api.Get("/")))
	})

	t.Run("request and response are handled with default", func(t *testing.T) {
		api := NewAPI(httpServer.Client(), *httpServerURL)
		assert.ErrorContains(t, api.Execute(context.Background(), api.Get("/")), "unhandled request status")
	})
}
