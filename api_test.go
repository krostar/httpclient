package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_NewAPI(t *testing.T) {
	api := NewAPI(nil, url.URL{})
	test.Assert(check.Compare(t, api, &API{
		client:                           nil,
		serverAddress:                    url.URL{},
		defaultRequestHeaders:            make(http.Header),
		defaultResponseHandlers:          make(ResponseStatusHandlers),
		defaultResponseBodySizeReadLimit: 65536,
	}, gocmp.AllowUnexported(API{})))

	api = NewAPI(http.DefaultClient, url.URL{Scheme: "http", Host: "localhost"})
	test.Assert(check.Compare(t, api, &API{
		client:                           http.DefaultClient,
		serverAddress:                    url.URL{Scheme: "http", Host: "localhost"},
		defaultRequestHeaders:            make(http.Header),
		defaultResponseHandlers:          make(ResponseStatusHandlers),
		defaultResponseBodySizeReadLimit: 65536,
	}, gocmp.AllowUnexported(API{})))
}

func Test_API_Clone(t *testing.T) {
	original := &API{
		client: http.DefaultClient,
		serverAddress: url.URL{
			Scheme: "http",
			User:   url.UserPassword("foo", "bar"),
			Host:   "localhost",
			Path:   "/foo",
		},
		defaultRequestHeaders:            map[string][]string{"hello": {"world"}},
		defaultResponseHandlers:          map[int]ResponseHandler{503: func(*http.Response) error { return nil }},
		defaultResponseBodySizeReadLimit: 58968,
	}
	clone := original.Clone()

	test.Assert(check.Compare(t, original, clone, // object are deeply equal
		gocmp.AllowUnexported(API{}, url.Userinfo{}),
		gocmp.FilterPath(
			func(path gocmp.Path) bool { return path.GoString() == "{*httpclient.API}.defaultResponseHandlers[503]" },
			gocmp.Comparer(func(a, b ResponseHandler) bool { return a != nil && b != nil }),
		),
	))
	test.Assert(t, original != clone)                                                   // but pointers must be different
	test.Assert(t, &original.defaultRequestHeaders != &clone.defaultRequestHeaders)     // same for maps
	test.Assert(t, &original.defaultResponseHandlers != &clone.defaultResponseHandlers) // same for maps
	test.Assert(t, original.serverAddress.User != clone.serverAddress.User)             // same for url attributes that also are pointers
}

func Test_API_URL(t *testing.T) {
	t.Run("", func(t *testing.T) {
		api := NewAPI(http.DefaultClient, url.URL{
			Scheme: "http",
			Host:   "localhost",
		})
		endpointURL := api.URL("/foo/bar")

		test.Assert(t, endpointURL.String() == "http://localhost/foo/bar")
	})

	t.Run("every fields are cloned", func(t *testing.T) {
		serverURL, err := url.Parse("https://foo@localhost:8080")
		test.Require(t, err == nil)

		api := NewAPI(http.DefaultClient, *serverURL)
		endpointURL := api.URL("/foo/bar")

		test.Assert(t, serverURL.User != endpointURL.User)
		test.Assert(t, *serverURL.User == *endpointURL.User)
	})
}

func Test_API_Method(t *testing.T) {
	type ctxKey string

	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		test.Assert(t, r.URL.Path == "/foo/bar")
		rw.Header().Set("Hello", r.Header.Get("Hello"))
		rw.Header().Set("Method", r.Method)
		rw.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	test.Require(t, err == nil && httpServerURL != nil)

	api := NewAPI(httpServer.Client(), *httpServerURL).
		WithRequestHeaders(http.Header{"hello": []string{"world"}}).
		WithRequestOverrideFunc(func(req *http.Request) (*http.Request, error) {
			return req.WithContext(context.WithValue(req.Context(), ctxKey("key"), "value")), nil
		})

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
		t.Run(httpMethod, func(t *testing.T) {
			t.Run("run with defaults", func(t *testing.T) {
				test.Assert(t, api.
					Do(t.Context(), apiMethod("/foo/bar")).
					OnStatus(http.StatusOK, func(resp *http.Response) error {
						test.Assert(t, resp.Header.Get("Hello") == "world", "header 'hello' should be set by default")
						test.Assert(t, resp.Header.Get("Method") == httpMethod)
						test.Assert(t, resp.Request.Context().Value(ctxKey("key")).(string) == "value")
						return nil
					}).
					Error() == nil)
			})

			t.Run("make sure all api defaults are set in builder and not after building", func(t *testing.T) {
				t.Run("client should be overridable", func(t *testing.T) {
					test.Assert(check.Panics(t, func() {
						_ = api.Execute(t.Context(), apiMethod("/foo/bar").Client(nil))
					}, nil))
				})

				t.Run("headers defaults should be overridable", func(t *testing.T) {
					test.Assert(t, api.
						Do(t.Context(), apiMethod("/foo/bar").SetHeader("hello", "notworld")).
						OnStatus(http.StatusOK, func(resp *http.Response) error {
							test.Assert(t, resp.Header.Get("Hello") == "notworld", "header 'hello' should have been overridden")
							return nil
						}).
						Error() == nil)
				})

				t.Run("request override func should be overridable", func(t *testing.T) {
					test.Assert(t, api.
						Clone().
						Do(t.Context(), apiMethod("/foo/bar").
							SetOverrideFunc(func(req *http.Request) (*http.Request, error) {
								return req.WithContext(context.WithValue(req.Context(), ctxKey("key"), "notvalue")), nil
							}),
						).
						OnStatus(http.StatusOK, func(resp *http.Response) error {
							test.Assert(t, resp.Request.Context().Value(ctxKey("key")).(string) == "notvalue")
							return nil
						}).
						Error() == nil)
				})
			})
		})
	}
}

func Test_API_Do(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Hello", r.Header.Get("Hello"))
		rw.Header().Set("Method", r.Method)

		if r.URL.Path == "/teapot" {
			rw.WriteHeader(http.StatusTeapot)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	test.Require(t, err == nil && httpServerURL != nil)

	api := NewAPI(httpServer.Client(), *httpServerURL).
		WithRequestHeaders(http.Header{"hello": []string{"world"}}).
		WithResponseHandler(http.StatusTeapot, func(*http.Response) error { return nil }).
		WithResponseBodySizeReadLimit(121212)

	t.Run("should not set defaults in request", func(t *testing.T) {
		t.Run("client", func(t *testing.T) {
			test.Assert(check.Panics(t, func() {
				_ = api.Do(t.Context(), NewRequest(http.MethodGet, httpServerURL.String()).Client(nil))
			}, nil))
		})

		t.Run("header", func(t *testing.T) {
			test.Assert(t, api.
				Do(t.Context(), NewRequest(http.MethodGet, httpServerURL.String()).SetHeader("hello", "notworld")).
				OnStatus(http.StatusOK, func(resp *http.Response) error {
					test.Assert(t, resp.Header.Get("Hello") == "notworld")
					return nil
				}).
				Error() == nil)
		})
	})

	t.Run("should set defaults in response", func(t *testing.T) {
		t.Run("max body read size", func(t *testing.T) {
			test.Assert(t, api.Do(t.Context(), NewRequest(http.MethodGet, httpServerURL.String())).bodySizeReadLimit == int64(121212))
		})

		t.Run("status not handled by default", func(t *testing.T) {
			err := api.Do(t.Context(), api.Get("/")).Error()
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status"))
		})

		t.Run("status handled by default", func(t *testing.T) {
			test.Assert(t, api.Do(t.Context(), api.Get("/teapot")).Error() == nil)
		})
	})
}

func Test_API_Execute(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusOK) }))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	test.Require(t, err == nil)

	t.Run("request and response are handled with default", func(t *testing.T) {
		api := NewAPI(httpServer.Client(), *httpServerURL).WithResponseHandler(http.StatusOK, func(*http.Response) error { return nil })

		test.Assert(t, api.Execute(t.Context(), api.Get("/")) == nil)
	})

	t.Run("request and response are handled with default", func(t *testing.T) {
		api := NewAPI(httpServer.Client(), *httpServerURL)
		err := api.Execute(t.Context(), api.Get("/"))
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status"))
	})
}
