package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_RequestBuilder_Client(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.client != nil)

	req = req.Client(nil)
	test.Assert(t, req.client == nil)

	req = req.Client(http.DefaultClient)
	test.Assert(t, req.client != nil)
}

func Test_RequestBuilder_SetHeader(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.header != nil)

	req = req.SetHeader("foobar", "foo", "bar")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"foo", "bar"}}))

	req = req.SetHeader("foo", "bar")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"foo", "bar"}, "Foo": {"bar"}}))

	req = req.SetHeader("foobar", "bar", "foo")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"bar", "foo"}, "Foo": {"bar"}}))
}

func Test_RequestBuilder_SetHeaders(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.header != nil)

	req = req.SetHeaders(http.Header{"foobar": {"foo", "bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"foo", "bar"}}))

	req = req.SetHeaders(http.Header{"foo": {"bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.SetHeaders(http.Header{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}}))
}

func Test_RequestBuilder_AddHeader(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.header != nil)

	req = req.AddHeader("foobar", "foo", "bar")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"foo", "bar"}}))

	req = req.AddHeader("foo", "bar")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"foo", "bar"}, "Foo": {"bar"}}))

	req = req.AddHeader("foobar", "bar", "foo")
	test.Assert(check.Compare(t, req.header, http.Header{"Foobar": {"foo", "bar", "bar", "foo"}, "Foo": {"bar"}}))
}

func Test_RequestBuilder_AddHeaders(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.header != nil)

	req = req.AddHeaders(http.Header{"foobar": {"foo", "bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"foo", "bar"}}))

	req = req.AddHeaders(http.Header{"foo": {"bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.AddHeaders(http.Header{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	test.Assert(check.Compare(t, req.header, http.Header{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}}))
}

func Test_RequestBuilder_SetQueryParam(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.SetQueryParam("foobar", "foo", "bar")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}}))

	req = req.SetQueryParam("foo", "bar")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.SetQueryParam("foobar", "bar", "foo")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"bar", "foo"}, "foo": {"bar"}}))
}

func Test_RequestBuilder_SetQueryParams(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.SetQueryParams(url.Values{"foobar": {"foo", "bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}}))

	req = req.SetQueryParams(url.Values{"foo": {"bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.SetQueryParams(url.Values{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}}))
}

func Test_RequestBuilder_AddQueryParam(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.AddQueryParam("foobar", "foo", "bar")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}}))

	req = req.AddQueryParam("foo", "bar")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.AddQueryParam("foobar", "bar", "foo")
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}}))
}

func Test_RequestBuilder_AddQueryParams(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.AddQueryParams(url.Values{"foobar": {"foo", "bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}}))

	req = req.AddQueryParams(url.Values{"foo": {"bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}}))

	req = req.AddQueryParams(url.Values{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	test.Assert(check.Compare(t, req.url.Query(), url.Values{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}}))
}

func Test_RequestBuilder_PathReplacer(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost/localhost")
	req = req.PathReplacer("localhost", "hostlocal")
	test.Assert(t, req.url.Host == "localhost")
	test.Assert(t, req.url.Path == "/hostlocal")

	req = NewRequest(http.MethodGet, "http://localhost/{userID}/{userID}/{ userID }")
	req = req.PathReplacer("{userID}", "42")
	test.Assert(t, req.url.Path == "/42/42/{ userID }")

	req = NewRequest(http.MethodGet, "http://localhost/{foo}/{bar}/{foobar}")
	req = req.PathReplacer("{foo}", "42")
	req = req.PathReplacer("{foo}", "32")
	req = req.PathReplacer("{bar}", "22")
	req = req.PathReplacer("{yolo}", "YOLO")
	test.Assert(t, req.url.Path == "/42/22/{foobar}")
}

func Test_RequestBuilder_SendForm(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.bodyMarshaler == nil)
	test.Assert(t, req.bodyToMarshal == nil)
	test.Assert(t, req.body == nil)
	test.Assert(t, req.header.Get("Content-Type") == "")

	req = req.SendForm(url.Values{"hello": {"world"}})
	test.Assert(t, req.bodyMarshaler == nil)
	test.Assert(t, req.bodyToMarshal == nil)
	test.Assert(t, req.body != nil)
	rawQuery, err := io.ReadAll(req.body)
	test.Require(t, err == nil)
	query, err := url.ParseQuery(string(rawQuery))
	test.Require(t, err == nil)
	test.Assert(check.Compare(t, query, url.Values{"hello": {"world"}}))
	test.Assert(t, req.header.Get("Content-Type") == "application/x-www-form-urlencoded")
}

func Test_RequestBuilder_SendJSON(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.bodyMarshaler == nil)
	test.Assert(t, req.bodyToMarshal == nil)
	test.Assert(t, req.body == nil)
	test.Assert(t, req.header.Get("Content-Type") == "")

	type input struct {
		Say string `json:"say"`
		To  string `json:"to"`
	}

	req = req.SendJSON(input{Say: "Hello", To: "world"})
	test.Assert(t, req.body == nil)
	test.Assert(t, req.bodyMarshaler != nil)
	test.Assert(t, req.bodyToMarshal != nil)
	rawBody, err := req.bodyMarshaler(req.bodyToMarshal)
	test.Require(t, err == nil)

	var parsedBody input
	test.Assert(t, json.Unmarshal(rawBody, &parsedBody) == nil)
	test.Assert(check.Compare(t, parsedBody, input{Say: "Hello", To: "world"}))
	test.Assert(t, req.header.Get("Content-Type") == "application/json")
}

func Test_RequestBuilder_Send(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	test.Assert(t, req.bodyMarshaler == nil)
	test.Assert(t, req.bodyToMarshal == nil)
	test.Assert(t, req.body == nil)
	test.Assert(t, req.header.Get("Content-Type") == "")

	req = req.Send(strings.NewReader("hello world!"))
	test.Assert(t, req.bodyMarshaler == nil)
	test.Assert(t, req.bodyToMarshal == nil)
	test.Assert(t, req.body != nil)
	rawBody, err := io.ReadAll(req.body)
	test.Require(t, err == nil)
	test.Assert(t, string(rawBody) == "hello world!")
	test.Assert(t, req.header.Get("Content-Type") == "application/octet-stream")
}

func Test_RequestBuilder_SetOverrideFunc(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost/localhost")
	test.Assert(t, req.overrideFunc == nil)
	req = req.SetOverrideFunc(func(*http.Request) (*http.Request, error) {
		return nil, errors.New("boom")
	})
	test.Assert(t, req.overrideFunc != nil)
}

func Test_RequestBuilder_Request(t *testing.T) {
	type ctxKey string

	t.Run("ok", func(t *testing.T) {
		t.Run("without body", func(t *testing.T) {
			requestBuilder := NewRequest(http.MethodGet, "http://localhost")
			requestBuilder = requestBuilder.AddHeader("hello", "world")
			requestBuilder = requestBuilder.SetOverrideFunc(func(req *http.Request) (*http.Request, error) {
				ctx := context.WithValue(t.Context(), ctxKey("key"), "value")
				return req.WithContext(ctx), nil
			})
			requestBuilt, err := requestBuilder.Request(t.Context())
			test.Require(t, err == nil)
			expected := newHTTPRequestForTesting(t, http.MethodGet, "http://localhost", nil,
				func(_ *testing.T, request *http.Request) {
					request.Header.Add("Hello", "world")
				},
			)
			test.Assert(t, compareHTTPRequests(requestBuilt, expected) == nil)
			test.Assert(t, requestBuilt.Context().Value(ctxKey("key")).(string) == "value")
		})

		t.Run("with body", func(t *testing.T) {
			t.Run("need to serialize", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder = requestBuilder.SendJSON("42")
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Require(t, err == nil)

				expected := newHTTPRequestForTesting(t, http.MethodPost, "http://localhost", strings.NewReader(`"42"`),
					func(_ *testing.T, request *http.Request) {
						request.Header.Add("Content-Type", "application/json")
					},
				)
				test.Assert(t, compareHTTPRequests(requestBuilt, expected) == nil)
			})

			t.Run("no need to serialize", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder = requestBuilder.Send(strings.NewReader("hello world"))
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Require(t, err == nil)

				expected := newHTTPRequestForTesting(t, http.MethodPost, "http://localhost", strings.NewReader(`hello world`),
					func(_ *testing.T, request *http.Request) {
						request.Header.Add("Content-Type", "application/octet-stream")
					},
				)
				test.Assert(t, compareHTTPRequests(requestBuilt, expected) == nil)
			})
		})
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("with body to serialize", func(t *testing.T) {
			t.Run("invalid url", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "\\:/\\")
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to parse endpoint url"))
				test.Assert(t, requestBuilt == nil)
			})

			t.Run("with body", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.body = strings.NewReader("hello world")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Assert(t, err != nil && strings.Contains(err.Error(), "body to marshal is set but body is already set"))
				test.Assert(t, requestBuilt == nil)
			})

			t.Run("without body serializer", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilder.bodyMarshaler = nil
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Assert(t, err != nil && strings.Contains(err.Error(), "body to marshal is set but body marshaller is unset"))
				test.Assert(t, requestBuilt == nil)
			})

			t.Run("unable to marshal body", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilder.bodyMarshaler = func(any) ([]byte, error) {
					return nil, errors.New("boom")
				}
				requestBuilt, err := requestBuilder.Request(t.Context())
				test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to marshal body: boom"))
				test.Assert(t, requestBuilt == nil)
			})
		})

		t.Run("unable to create request", func(t *testing.T) {
			requestBuilder := NewRequest(`\`, "http://localhost")
			requestBuilt, err := requestBuilder.Request(t.Context())
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create request \\ http://localhost: net/http: invalid method"))
			test.Assert(t, requestBuilt == nil)
		})

		t.Run("unable to override request", func(t *testing.T) {
			anError := errors.New("boom")
			requestBuilder := NewRequest(http.MethodPost, "http://localhost")
			requestBuilder = requestBuilder.SetOverrideFunc(func(*http.Request) (*http.Request, error) {
				return nil, anError
			})
			requestBuilt, err := requestBuilder.Request(t.Context())
			test.Assert(t, errors.Is(err, anError))
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to override request: boom"))
			test.Assert(t, requestBuilt == nil)
		})
	})
}

func Test_RequestBuilder_Do(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		}))
		defer httpServer.Close()

		httpServerURL, err := url.Parse(httpServer.URL)
		test.Require(t, err == nil)

		httpClientSpy := &doerSpy{doer: httpServer.Client()}

		requestBuilder := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpClientSpy)
		responseBuilder := requestBuilder.Do(t.Context())
		test.Assert(t, responseBuilder != nil)
		test.Assert(t, responseBuilder.builderError == nil)
		test.Assert(t, responseBuilder.resp.StatusCode == http.StatusTeapot)

		test.Assert(t, len(httpClientSpy.calls) == 1)
		requestBuilt, err := requestBuilder.Request(t.Context())
		test.Require(t, err == nil)
		test.Assert(t, compareHTTPRequests(requestBuilt, httpClientSpy.calls[0].req) == nil)
		test.Assert(t, responseBuilder.resp.StatusCode == httpClientSpy.calls[0].resp.StatusCode)
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("unable to create the request", func(t *testing.T) {
			responseBuilder := NewRequest(`\`, "http://localhost").Do(t.Context())
			test.Assert(t, responseBuilder != nil)
			test.Assert(t, responseBuilder.builderError != nil && strings.Contains(responseBuilder.builderError.Error(), "unable to create request"))
			test.Assert(t, responseBuilder.resp == nil)
		})

		t.Run("unable to perform the request using client", func(t *testing.T) {
			httpClientStub := &doerFail{err: errors.New("boom")}

			responseBuilder := NewRequest(http.MethodGet, "http://localhost").Client(httpClientStub).Do(t.Context())
			test.Assert(t, responseBuilder != nil)
			test.Assert(t, responseBuilder.builderError != nil && strings.Contains(responseBuilder.builderError.Error(), "unable to execute GET http://localhost request: boom"))
			test.Assert(t, responseBuilder.resp == nil)
		})
	})
}
