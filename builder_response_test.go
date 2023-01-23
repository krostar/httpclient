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

func Test_ResponseBuilder_BodySizeReadLimit(t *testing.T) {
	resp := newResponse()
	assert.Check(t, resp.bodySizeReadLimit == 0)
	resp = resp.BodySizeReadLimit(42)
	assert.Check(t, resp.bodySizeReadLimit == 42)
}

func Test_ResponseBuilder_OnStatus(t *testing.T) {
	called := make(map[int]int)
	resp := newResponse()

	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	assert.Check(t, func() bool {
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

	assert.Check(t, resp.statusHandler[http.StatusOK](nil) == nil)
	assert.Check(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
	assert.Check(t, cmp.DeepEqual(called, map[int]int{http.StatusTeapot: 1, http.StatusOK: 1}))
}

func Test_ResponseBuilder_OnStatuses(t *testing.T) {
	var called int

	resp := newResponse()

	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusTeapot]
		return !exists
	}())

	resp = resp.OnStatuses([]int{http.StatusOK, http.StatusTeapot}, func(*http.Response) error {
		called++
		return nil
	})

	assert.Check(t, resp.statusHandler[http.StatusOK](nil) == nil)
	assert.Check(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
	assert.Check(t, cmp.Equal(called, 2))
}

func Test_ResponseBuilder_ErrorOnStatus(t *testing.T) {
	anError := errors.New("an error")
	resp := newResponse()

	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusBadRequest]
		return !exists
	}())

	resp = resp.ErrorOnStatus(http.StatusBadRequest, anError)

	assert.Check(t, cmp.ErrorIs(resp.statusHandler[http.StatusBadRequest](nil), anError))
}

func Test_ResponseBuilder_SuccessOnStatus(t *testing.T) {
	resp := newResponse()

	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusOK]
		return !exists
	}())
	assert.Check(t, func() bool {
		_, exists := resp.statusHandler[http.StatusTeapot]
		return !exists
	}())

	resp = resp.SuccessOnStatus(http.StatusOK, http.StatusTeapot)
	assert.Check(t, resp.statusHandler[http.StatusOK](nil) == nil)
	assert.Check(t, resp.statusHandler[http.StatusTeapot](nil) == nil)
}

func Test_ResponseBuilder_ReceiveJSON(t *testing.T) {
	type responseBody struct {
		Hello string `json:"hello"`
	}

	httpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
		assert.NilError(t, json.NewEncoder(rw).Encode(&responseBody{Hello: "hi!"}))
	}))
	defer httpServer.Close()
	httpServerURL, err := url.Parse(httpServer.URL)
	assert.NilError(t, err)

	t.Run("ok", func(t *testing.T) {
		var body responseBody

		resp := NewRequest(http.MethodPost, httpServerURL.String()).Do(context.Background())
		assert.Check(t, func() bool {
			_, exists := resp.statusHandler[http.StatusTeapot]
			return !exists
		}())

		resp = resp.ReceiveJSON(http.StatusTeapot, &body)
		assert.NilError(t, resp.statusHandler[http.StatusTeapot](resp.resp))
		assert.Equal(t, body, responseBody{Hello: "hi!"})
	})

	t.Run("ko", func(t *testing.T) {
		var body struct {
			Hello int `json:"hello"`
		}

		resp := NewRequest(http.MethodPost, httpServerURL.String()).Do(context.Background())
		resp = resp.ReceiveJSON(http.StatusTeapot, &body)
		assert.ErrorContains(t, resp.statusHandler[http.StatusTeapot](resp.resp), "unable to parse JSON response body")
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
			assert.NilError(t, err)
		}
	})

	t.Run("body is closed every time when a response exists", func(t *testing.T) {
		responseBuilder := NewRequest(http.MethodGet, httpServerURL.String()).
			Client(httpServer.Client()).
			Do(context.Background()).
			SuccessOnStatus(http.StatusTeapot)
		spy := &spyReadCloser{readCloser: responseBuilder.resp.Body}
		responseBuilder.resp.Body = spy
		assert.NilError(t, responseBuilder.Error())
		assert.Check(t, spy.closeCallCount == 1)
	})

	t.Run("is case of building error, return without executing the query", func(t *testing.T) {
		var called bool
		resp := NewRequest(http.MethodGet, httpServerURL.String()).
			Client(httpServer.Client()).
			Do(context.Background()).
			OnStatus(http.StatusTeapot, func(*http.Response) error {
				called = true
				return nil
			})
		resp.builderError = errors.New("boom")
		assert.ErrorContains(t, resp.Error(), "boom")
		assert.Check(t, !called)
	})

	t.Run("body read limit is not negative", func(t *testing.T) {
		t.Run("0 means ContentLength", func(t *testing.T) {
			responseBuilder := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(context.Background())

			// simulate a faked Content-Length to check we only read part of the body
			responseBuilder.resp.ContentLength = 2

			assert.NilError(t, responseBuilder.
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					assert.NilError(t, err)
					assert.Equal(t, int64(len(body)), resp.ContentLength)
					assert.Equal(t, len(body), 2)
					return nil
				}).
				BodySizeReadLimit(20).
				Error())
		})

		t.Run("negative ContentLength", func(t *testing.T) {
			responseBuilder := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(context.Background())

			// simulate a faked Content-Length to check we only read part of the body
			responseBuilder.resp.ContentLength = -1

			assert.NilError(t, responseBuilder.
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					assert.NilError(t, err)
					assert.Equal(t, resp.ContentLength, int64(-1))
					assert.Equal(t, len(body), 3)
					return nil
				}).
				BodySizeReadLimit(3).
				Error())
		})

		t.Run("inferior to content length", func(t *testing.T) {
			assert.ErrorContains(t,
				NewRequest(http.MethodPost, httpServerURL.String()).
					Client(httpServer.Client()).
					Do(context.Background()).
					BodySizeReadLimit(2).
					Error(),
				"content length 14 is above read limit 2",
			)
		})

		t.Run("superior to content length", func(t *testing.T) {
			assert.NilError(t, NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(context.Background()).
				OnStatus(http.StatusTeapot, func(resp *http.Response) error {
					body, err := io.ReadAll(resp.Body)
					assert.NilError(t, err)
					t.Log(resp.ContentLength, len(body))
					// read only content-length
					return nil
				}).
				BodySizeReadLimit(20).
				Error())
		})
	})

	t.Run("a handler exists for the response status code", func(t *testing.T) {
		var called bool

		resp := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpServer.Client()).Do(context.Background())
		resp.OnStatus(http.StatusTeapot, func(*http.Response) error {
			called = true
			return nil
		})

		assert.NilError(t, resp.Error())
		assert.Check(t, called)
	})

	t.Run("no handler exists for the response status code", func(t *testing.T) {
		t.Run("response contains a body", func(t *testing.T) {
			var called bool

			resp := NewRequest(http.MethodPost, httpServerURL.String()).
				Client(httpServer.Client()).
				Do(context.Background()).
				OnStatus(http.StatusOK, func(*http.Response) error {
					called = true
					return nil
				})

			assert.ErrorContains(t, resp.Error(), "unhandled request status with b64 body ImhlbGxvIHdvcmxkISI=")
			assert.Check(t, !called)
		})

		t.Run("response does not contain a body", func(t *testing.T) {
			resp := NewRequest(http.MethodGet, httpServerURL.String()).Client(httpServer.Client()).Do(context.Background())
			err := resp.Error()
			assert.Check(t, cmp.ErrorContains(err, "failed with status 418: unhandled request status"))
			assert.Check(t, !strings.Contains(err.Error(), "with b64 body"))
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
