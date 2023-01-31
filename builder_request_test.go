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

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_RequestBuilder_Client(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.client != nil)

	req = req.Client(nil)
	assert.Check(t, req.client == nil)

	req = req.Client(http.DefaultClient)
	assert.Check(t, req.client != nil)
}

func Test_RequestBuilder_SetHeader(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.header != nil)

	req = req.SetHeader("foobar", "foo", "bar")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"foo", "bar"}})

	req = req.SetHeader("foo", "bar")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"foo", "bar"}, "Foo": {"bar"}})

	req = req.SetHeader("foobar", "bar", "foo")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"bar", "foo"}, "Foo": {"bar"}})
}

func Test_RequestBuilder_SetHeaders(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.header != nil)

	req = req.SetHeaders(http.Header{"foobar": {"foo", "bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"foo", "bar"}})

	req = req.SetHeaders(http.Header{"foo": {"bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.SetHeaders(http.Header{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}})
}

func Test_RequestBuilder_AddHeader(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.header != nil)

	req = req.AddHeader("foobar", "foo", "bar")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"foo", "bar"}})

	req = req.AddHeader("foo", "bar")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"foo", "bar"}, "Foo": {"bar"}})

	req = req.AddHeader("foobar", "bar", "foo")
	assert.DeepEqual(t, req.header, http.Header{"Foobar": {"foo", "bar", "bar", "foo"}, "Foo": {"bar"}})
}

func Test_RequestBuilder_AddHeaders(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.header != nil)

	req = req.AddHeaders(http.Header{"foobar": {"foo", "bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"foo", "bar"}})

	req = req.AddHeaders(http.Header{"foo": {"bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.AddHeaders(http.Header{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	assert.DeepEqual(t, req.header, http.Header{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}})
}

func Test_RequestBuilder_SetQueryParam(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.SetQueryParam("foobar", "foo", "bar")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}})

	req = req.SetQueryParam("foo", "bar")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.SetQueryParam("foobar", "bar", "foo")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"bar", "foo"}, "foo": {"bar"}})
}

func Test_RequestBuilder_SetQueryParams(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.SetQueryParams(url.Values{"foobar": {"foo", "bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}})

	req = req.SetQueryParams(url.Values{"foo": {"bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.SetQueryParams(url.Values{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}})
}

func Test_RequestBuilder_AddQueryParam(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.AddQueryParam("foobar", "foo", "bar")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}})

	req = req.AddQueryParam("foo", "bar")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.AddQueryParam("foobar", "bar", "foo")
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}})
}

func Test_RequestBuilder_AddQueryParams(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.url == url.URL{Scheme: "http", Host: "localhost"})

	req = req.AddQueryParams(url.Values{"foobar": {"foo", "bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}})

	req = req.AddQueryParams(url.Values{"foo": {"bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar"}, "foo": {"bar"}})

	req = req.AddQueryParams(url.Values{"foobar": {"bar", "foo"}, "bar": {"bar"}})
	assert.DeepEqual(t, req.url.Query(), url.Values{"foobar": {"foo", "bar", "bar", "foo"}, "foo": {"bar"}, "bar": {"bar"}})
}

func Test_RequestBuilder_PathReplacer(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost/localhost")
	req = req.PathReplacer("localhost", "hostlocal")
	assert.Equal(t, req.url.Host, "localhost")
	assert.Equal(t, req.url.Path, "/hostlocal")

	req = NewRequest(http.MethodGet, "http://localhost/{userID}/{userID}/{ userID }")
	req = req.PathReplacer("{userID}", "42")
	assert.Equal(t, req.url.Path, "/42/42/{ userID }")

	req = NewRequest(http.MethodGet, "http://localhost/{foo}/{bar}/{foobar}")
	req = req.PathReplacer("{foo}", "42")
	req = req.PathReplacer("{foo}", "32")
	req = req.PathReplacer("{bar}", "22")
	req = req.PathReplacer("{yolo}", "YOLO")
	assert.Equal(t, req.url.Path, "/42/22/{foobar}")
}

func Test_RequestBuilder_SendForm(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.bodyMarshaler == nil)
	assert.Check(t, req.bodyToMarshal == nil)
	assert.Check(t, req.body == nil)
	assert.Check(t, req.header.Get("Content-Type") == "")

	req = req.SendForm(url.Values{"hello": {"world"}})
	assert.Check(t, req.bodyMarshaler == nil)
	assert.Check(t, req.bodyToMarshal == nil)
	assert.Check(t, req.body != nil)
	rawQuery, err := io.ReadAll(req.body)
	assert.NilError(t, err)
	query, err := url.ParseQuery(string(rawQuery))
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(query, url.Values{"hello": {"world"}}))
	assert.Check(t, req.header.Get("Content-Type") == "application/x-www-form-urlencoded")
}

func Test_RequestBuilder_SendJSON(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.bodyMarshaler == nil)
	assert.Check(t, req.bodyToMarshal == nil)
	assert.Check(t, req.body == nil)
	assert.Check(t, req.header.Get("Content-Type") == "")

	type input struct {
		Say string `json:"say"`
		To  string `json:"to"`
	}

	req = req.SendJSON(input{Say: "Hello", To: "world"})
	assert.Check(t, req.body == nil)
	assert.Assert(t, req.bodyMarshaler != nil)
	assert.Assert(t, req.bodyToMarshal != nil)
	rawBody, err := req.bodyMarshaler(req.bodyToMarshal)
	assert.NilError(t, err)

	var parsedBody input
	assert.NilError(t, json.Unmarshal(rawBody, &parsedBody))
	assert.Check(t, cmp.DeepEqual(parsedBody, input{Say: "Hello", To: "world"}))
	assert.Check(t, req.header.Get("Content-Type") == "application/json")
}

func Test_RequestBuilder_Send(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost")
	assert.Check(t, req.bodyMarshaler == nil)
	assert.Check(t, req.bodyToMarshal == nil)
	assert.Check(t, req.body == nil)
	assert.Check(t, req.header.Get("Content-Type") == "")

	req = req.Send(strings.NewReader("hello world!"))
	assert.Check(t, req.bodyMarshaler == nil)
	assert.Check(t, req.bodyToMarshal == nil)
	assert.Check(t, req.body != nil)
	rawBody, err := io.ReadAll(req.body)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal("hello world!", string(rawBody)))
	assert.Check(t, req.header.Get("Content-Type") == "application/octet-stream")
}

func Test_RequestBuilder_SetOverrideFunc(t *testing.T) {
	req := NewRequest(http.MethodGet, "http://localhost/localhost")
	assert.Check(t, req.overrideFunc == nil)
	req = req.SetOverrideFunc(func(req *http.Request) (*http.Request, error) {
		return nil, errors.New("boom")
	})
	assert.Check(t, req.overrideFunc != nil)
}

func Test_RequestBuilder_Request(t *testing.T) {
	type ctxKey string

	t.Run("ok", func(t *testing.T) {
		t.Run("without body", func(t *testing.T) {
			requestBuilder := NewRequest(http.MethodGet, "http://localhost")
			requestBuilder = requestBuilder.AddHeader("hello", "world")
			requestBuilder = requestBuilder.SetOverrideFunc(func(req *http.Request) (*http.Request, error) {
				ctx := context.WithValue(context.Background(), ctxKey("key"), "value")
				return req.WithContext(ctx), nil
			})
			requestBuilt, err := requestBuilder.Request(context.Background())
			assert.NilError(t, err)
			assert.Check(t, compareHTTPRequestFunc(requestBuilt,
				newHTTPRequestForTesting(t, http.MethodGet, "http://localhost", nil,
					func(t *testing.T, request *http.Request) {
						request.Header.Add("hello", "world")
					},
				)),
			)
			assert.Check(t, requestBuilt.Context().Value(ctxKey("key")).(string) == "value")
		})

		t.Run("with body", func(t *testing.T) {
			t.Run("need to serialize", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder = requestBuilder.SendJSON("42")
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.NilError(t, err)

				assert.Check(t, compareHTTPRequestFunc(requestBuilt,
					newHTTPRequestForTesting(t, http.MethodPost, "http://localhost", strings.NewReader(`"42"`),
						func(t *testing.T, request *http.Request) {
							request.Header.Add("Content-Type", "application/json")
						},
					),
				))
			})

			t.Run("no need to serialize", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder = requestBuilder.Send(strings.NewReader("hello world"))
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.NilError(t, err)

				assert.Check(t, compareHTTPRequestFunc(requestBuilt,
					newHTTPRequestForTesting(t, http.MethodPost, "http://localhost", strings.NewReader(`hello world`),
						func(t *testing.T, request *http.Request) {
							request.Header.Add("Content-Type", "application/octet-stream")
						},
					),
				))
			})
		})
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("with body to serialize", func(t *testing.T) {
			t.Run("invalid url", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "\\:/\\")
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.ErrorContains(t, err, "unable to parse endpoint url")
				assert.Check(t, requestBuilt == nil)
			})

			t.Run("with body", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.body = strings.NewReader("hello world")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.ErrorContains(t, err, "body to marshal is set but body is already set")
				assert.Check(t, requestBuilt == nil)
			})

			t.Run("without body serializer", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilder.bodyMarshaler = nil
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.ErrorContains(t, err, "body to marshal is set but body marshaller is unset")
				assert.Check(t, requestBuilt == nil)
			})

			t.Run("unable to marshal body", func(t *testing.T) {
				requestBuilder := NewRequest(http.MethodPost, "http://localhost")
				requestBuilder.bodyToMarshal = `"hello world"`
				requestBuilder.bodyMarshaler = func(any) ([]byte, error) {
					return nil, errors.New("boom")
				}
				requestBuilt, err := requestBuilder.Request(context.Background())
				assert.ErrorContains(t, err, "unable to marshal body: boom")
				assert.Check(t, requestBuilt == nil)
			})
		})

		t.Run("unable to create request", func(t *testing.T) {
			requestBuilder := NewRequest(`\`, "http://localhost")
			requestBuilt, err := requestBuilder.Request(context.Background())
			assert.ErrorContains(t, err, "unable to create request \\ http://localhost: net/http: invalid method")
			assert.Check(t, requestBuilt == nil)
		})

		t.Run("unable to override request", func(t *testing.T) {
			anError := errors.New("boom")
			requestBuilder := NewRequest(http.MethodPost, "http://localhost")
			requestBuilder = requestBuilder.SetOverrideFunc(func(*http.Request) (*http.Request, error) {
				return nil, anError
			})
			requestBuilt, err := requestBuilder.Request(context.Background())
			assert.ErrorIs(t, err, anError)
			assert.ErrorContains(t, err, "unable to override request: boom")
			assert.Check(t, requestBuilt == nil)
		})
	})
}

func Test_RequestBuilder_Do(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		}))
		defer httpServer.Close()
		httpServerURL, err := url.Parse(httpServer.URL)
		assert.NilError(t, err)

		httpClientSpy := &doerSpy{doer: httpServer.Client()}

		requestBuilder := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpClientSpy)
		responseBuilder := requestBuilder.Do(context.Background())
		assert.Assert(t, responseBuilder != nil)
		assert.NilError(t, responseBuilder.builderError)
		assert.Check(t, responseBuilder.resp.StatusCode == http.StatusTeapot)

		assert.Assert(t, len(httpClientSpy.calls) == 1)
		requestBuilt, err := requestBuilder.Request(context.Background())
		assert.NilError(t, err)
		assert.Check(t, compareHTTPRequestFunc(requestBuilt, httpClientSpy.calls[0].req))
		assert.Check(t, compareHTTPResponseFunc(responseBuilder.resp, httpClientSpy.calls[0].resp))
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("unable to create the request", func(t *testing.T) {
			responseBuilder := NewRequest(`\`, "http://localhost").Do(context.Background())
			assert.Assert(t, responseBuilder != nil)
			assert.ErrorContains(t, responseBuilder.builderError, "unable to create request")
			assert.Check(t, responseBuilder.resp == nil)
		})

		t.Run("unable to perform the request using client", func(t *testing.T) {
			httpClientStub := &doerFail{err: errors.New("boom")}

			responseBuilder := NewRequest(http.MethodGet, "http://localhost").Client(httpClientStub).Do(context.Background())
			assert.Assert(t, responseBuilder != nil)
			assert.ErrorContains(t, responseBuilder.builderError, "unable to execute GET http://localhost request: boom")
			assert.Check(t, responseBuilder.resp == nil)
		})
	})
}
