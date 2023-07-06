package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_ParsePostForm(t *testing.T) {
	t.Run("method handled by PostForm", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})
		})

		t.Run("ko", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Body = nil
			assert.ErrorContains(t, ParsePostForm(req), "unable to parse body as form")
		})
	})

	t.Run("method not handled by PostForm", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})
		})

		t.Run("ko", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(";;"))
			assert.ErrorContains(t, ParsePostForm(req), "unable to parse form values from body")
		})
	})

	t.Run("idempotency", func(t *testing.T) {
		t.Run("regular methods", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})
		})

		t.Run("custom methods", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})

			assert.NilError(t, ParsePostForm(req))
			assert.DeepEqual(t, req.PostForm, url.Values{"foo": {"bar"}})
		})
	})
}

func newHTTPServerForTesting(t *testing.T, handler http.HandlerFunc) (*httptest.Server, url.URL) {
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	httpServerURL, err := url.Parse(httpServer.URL)
	assert.NilError(t, err)

	return httpServer, *httpServerURL
}

func newHTTPRequestForTesting(t *testing.T, method, endpoint string, body io.Reader, callbacks ...func(*testing.T, *http.Request)) *http.Request {
	req, err := http.NewRequestWithContext(context.Background(), method, endpoint, body)
	assert.NilError(t, err)

	for _, callback := range callbacks {
		callback(t, req)
	}

	return req
}

func compareHTTPRequests(x, y *http.Request) error {
	if diff := gocmp.Diff(x, y,
		gocmpopts.IgnoreUnexported(http.Request{}),
		gocmp.FilterPath(
			func(path gocmp.Path) bool {
				return path.GoString() == "{*http.Request}.Body"
			},
			gocmp.Comparer(func(a, b io.Reader) bool {
				switch {
				case a == nil && b == nil:
					return true
				case a != nil && b == nil:
					return false
				case a == nil && b != nil:
					return false
				case a != nil && b != nil:
					contentA, err := io.ReadAll(a)
					if err != nil {
						return false
					}
					contentB, err := io.ReadAll(b)
					if err != nil {
						return false
					}
					return bytes.Equal(contentA, contentB)
				default:
					return false
				}
			}),
		),
		gocmp.FilterPath(
			func(path gocmp.Path) bool {
				return path.GoString() == "{*http.Request}.GetBody"
			},
			gocmp.Comparer(func(a, b func() (io.ReadCloser, error)) bool {
				switch {
				case a != nil && b == nil:
				case a == nil && b != nil:
				case a == nil && b == nil:
					return true
				case a != nil && b != nil:
					bodyA, errA := a()
					bodyB, errB := b()
					if !(errA == nil && errB == nil) {
						return errA == errB
					}

					contentA, err := io.ReadAll(bodyA)
					if err != nil {
						return false
					}
					contentB, err := io.ReadAll(bodyB)
					if err != nil {
						return false
					}
					return bytes.Equal(contentA, contentB)
				}
				return false
			}),
		),
	); diff != "" {
		return errors.New(diff)
	}
	return nil
}

func compareHTTPRequestFunc(x, y *http.Request) cmp.Comparison {
	return func() cmp.Result { return cmp.ResultFromError(compareHTTPRequests(x, y)) }
}

func compareHTTPResponseFunc(x, y *http.Response) cmp.Comparison {
	return func() cmp.Result {
		if diff := gocmp.Diff(x, y,
			gocmp.FilterPath(
				func(path gocmp.Path) bool { return strings.HasPrefix(path.GoString(), "{*http.Response}.Request") },
				gocmp.Comparer(func(x, y *http.Request) bool { return compareHTTPRequests(x, y) == nil }),
			),
		); diff != "" {
			return cmp.ResultFailure(diff)
		}
		return cmp.ResultSuccess
	}
}

type doerSpy struct {
	doer  Doer
	calls []doerSpyCall
}

type doerSpyCall struct {
	req  *http.Request
	resp *http.Response
	err  error //nolint: unused // it is not used, but it still makes sens to keep it
}

func (spy *doerSpy) Do(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	resp, err = spy.doer.Do(req)

	spy.calls = append(spy.calls, struct {
		req  *http.Request
		resp *http.Response
		err  error
	}{
		req:  req,
		resp: resp,
		err:  err,
	})

	return resp, err
}

type doerFail struct {
	err error
}

func (fail *doerFail) Do(*http.Request) (*http.Response, error) { return nil, fail.err }
