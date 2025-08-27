package httpclient

import (
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

func Test_ResponseBuilder_BodySizeReadLimit(t *testing.T) {
	resp := newResponse()
	test.Assert(t, resp.bodySizeReadLimit == 0)
	resp = resp.BodySizeReadLimit(42)
	test.Assert(t, resp.bodySizeReadLimit == 42)
}

func Test_ResponseBuilder_OnStatus(t *testing.T) {
	called := make(map[int]int)
	resp := newResponse()

	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusTeapot]
		return !exists
	}())

	resp = resp.OnStatus(http.StatusTeapot, func(*http.Response) error {
		called[http.StatusTeapot]++
		return nil
	})

	resp = resp.OnStatus(http.StatusOK, func(*http.Response) error {
		called[http.StatusOK]++
		return nil
	})

	test.Assert(t, resp.statusHandler[http.StatusOK](nil) == nil)
	test.Assert(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
	test.Assert(check.Compare(t, called, map[int]int{http.StatusTeapot: 1, http.StatusOK: 1}))
}

func Test_ResponseBuilder_OnStatuses(t *testing.T) {
	var called int

	resp := newResponse()

	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusTeapot]
		return !exists
	}())

	resp = resp.OnStatuses([]int{http.StatusOK, http.StatusTeapot}, func(*http.Response) error {
		called++
		return nil
	})

	test.Assert(t, resp.statusHandler[http.StatusOK](nil) == nil)
	test.Assert(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
	test.Assert(t, called == 2)
}

func Test_ResponseBuilder_ErrorOnStatus(t *testing.T) {
	anError := errors.New("an error")
	resp := newResponse()

	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusBadRequest]
		return !exists
	}())

	resp = resp.ErrorOnStatus(http.StatusBadRequest, anError)

	test.Assert(t, errors.Is(resp.statusHandler[http.StatusBadRequest](nil), anError))
}

func Test_ResponseBuilder_SuccessOnStatus(t *testing.T) {
	resp := newResponse()

	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	test.Assert(t, func() bool {
		_, exists := resp.statusHandler[http.StatusTeapot]
		return !exists
	}())

	resp = resp.SuccessOnStatus(http.StatusOK, http.StatusTeapot)
	test.Assert(t, resp.statusHandler[http.StatusOK](nil) == nil)
	test.Assert(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
}

func Test_ResponseBuilder_ReceiveJSON(t *testing.T) {
	type responseBody struct {
		Hello string `json:"hello"`
	}

	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
		test.Assert(t, json.NewEncoder(rw).Encode(&responseBody{Hello: "hi!"}) == nil)
	}))
	defer httpServer.Close()

	httpServerURL, err := url.Parse(httpServer.URL)
	test.Require(t, err == nil)

	t.Run("ok", func(t *testing.T) {
		var body responseBody

		resp := NewRequest(http.MethodPost, httpServerURL.String()).Do(t.Context())
		test.Assert(t, func() bool {
			_, exists := resp.statusHandler[http.StatusTeapot]
			return !exists
		}())

		resp = resp.ReceiveJSON(http.StatusTeapot, &body)
		test.Assert(t, resp.statusHandler[http.StatusTeapot](resp.resp) == nil)
		test.Assert(check.Compare(t, body, responseBody{Hello: "hi!"}))
	})

	t.Run("ko", func(t *testing.T) {
		var body struct {
			Hello int `json:"hello"`
		}

		resp := NewRequest(http.MethodPost, httpServerURL.String()).Do(t.Context())
		resp = resp.ReceiveJSON(http.StatusTeapot, &body)
		err := resp.statusHandler[http.StatusTeapot](resp.resp)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to parse JSON response body"))
	})
}

func Test_ResponseBuilder_Error(t *testing.T) {
	httpServer, httpServerURL := newHTTPServerForTesting(t, func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rw.WriteHeader(http.StatusTeapot)
		case http.MethodPost:
			rw.WriteHeader(http.StatusTeapot)
			_, err := rw.Write([]byte(`"hello world!"`))
			test.Require(t, err == nil)
		}
	})

	t.Run("body is closed every time when a response exists", func(t *testing.T) {
		responseBuilder := NewRequest(http.MethodGet, httpServerURL.String()).
			Client(httpServer.Client()).
			Do(t.Context()).
			SuccessOnStatus(http.StatusTeapot)
		spy := &spyReadCloser{readCloser: responseBuilder.resp.Body}
		responseBuilder.resp.Body = spy
		test.Assert(t, responseBuilder.Error() == nil)
		test.Assert(t, spy.closeCallCount == 1)
	})

	t.Run("is case of building error, return without executing the query", func(t *testing.T) {
		var called bool

		resp := NewRequest(http.MethodGet, httpServerURL.String()).
			Client(httpServer.Client()).
			Do(t.Context()).
			OnStatus(http.StatusTeapot, func(*http.Response) error {
				called = true
				return nil
			})
		resp.builderError = errors.New("boom")
		err := resp.Error()
		test.Assert(t, err != nil && strings.Contains(err.Error(), "boom"))
		test.Assert(t, !called)
	})

	t.Run("body read limit is not negative", func(t *testing.T) {
		t.Run("0 means ContentLength", func(t *testing.T) {
			responseBuilder := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(t.Context())

			// simulate a faked Content-Length to check we only read part of the body
			responseBuilder.resp.ContentLength = 2

			test.Assert(t, responseBuilder.
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					test.Require(t, err == nil)
					test.Assert(t, int64(len(body)) == resp.ContentLength)
					test.Assert(t, len(body) == 2)
					return nil
				}).
				BodySizeReadLimit(20).
				Error() == nil)
		})

		t.Run("negative ContentLength", func(t *testing.T) {
			responseBuilder := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(t.Context())

			// simulate a faked Content-Length to check we only read part of the body
			responseBuilder.resp.ContentLength = -1

			test.Assert(t, responseBuilder.
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					test.Require(t, err == nil)
					test.Assert(t, resp.ContentLength == int64(-1))
					test.Assert(t, len(body) == 3)
					return nil
				}).
				BodySizeReadLimit(3).
				Error() == nil)
		})

		t.Run("inferior to content length", func(t *testing.T) {
			err := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(t.Context()).
				BodySizeReadLimit(2).
				Error()
			test.Assert(t, err != nil && strings.Contains(err.Error(), "content length 14 is above read limit 2"))
		})

		t.Run("superior to content length", func(t *testing.T) {
			test.Assert(t, NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(t.Context()).
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					test.Require(t, err == nil)
					t.Log(resp.ContentLength, len(body))
					// read only content-length
					return nil
				}).
				BodySizeReadLimit(20).
				Error() == nil)
		})
	})

	t.Run("a handler exists for the response status code", func(t *testing.T) {
		var called bool

		resp := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpServer.Client()).Do(t.Context())
		resp.OnStatus(http.StatusTeapot, func(*http.Response) error {
			called = true
			return nil
		})

		test.Assert(t, resp.Error() == nil)
		test.Assert(t, called)
	})

	t.Run("no handler exists for the response status code", func(t *testing.T) {
		t.Run("response contains a body", func(t *testing.T) {
			var called bool

			resp := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(t.Context()).
				OnStatus(http.StatusOK, func(*http.Response) error {
					called = true
					return nil
				})

			err := resp.Error()
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status with b64 body ImhlbGxvIHdvcmxkISI="))
			test.Assert(t, !called)
		})

		t.Run("response does not contain a body", func(t *testing.T) {
			resp := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpServer.Client()).Do(t.Context())
			err := resp.Error()
			test.Assert(t, err != nil && strings.Contains(err.Error(), "failed with status 418: unhandled status"))
			test.Assert(t, !strings.Contains(err.Error(), "with b64 body"))
		})
	})
}

type spyReadCloser struct {
	readCloser     io.ReadCloser
	closeCallCount uint
}

func (s *spyReadCloser) Read(p []byte) (int, error) {
	return s.readCloser.Read(p)
}

func (s *spyReadCloser) Close() error {
	defer func() { s.closeCallCount++ }()
	return s.readCloser.Close()
}
