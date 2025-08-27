package httpclient

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_ParsePostForm(t *testing.T) {
	t.Run("method handled by PostForm", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))
		})

		t.Run("ko", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Body = nil
			err := ParsePostForm(req)
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to parse body as form"))
		})
	})

	t.Run("method not handled by PostForm", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))
		})

		t.Run("ko", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(";;"))
			err := ParsePostForm(req)
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to parse form values from body"))
		})
	})

	t.Run("idempotency", func(t *testing.T) {
		t.Run("regular methods", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))
		})

		t.Run("custom methods", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"foo": {"bar"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))

			test.Assert(t, ParsePostForm(req) == nil)
			test.Assert(check.Compare(t, req.PostForm, url.Values{"foo": {"bar"}}))
		})
	})
}

func newHTTPServerForTesting(t *testing.T, handler http.HandlerFunc) (*httptest.Server, url.URL) {
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	httpServerURL, err := url.Parse(httpServer.URL)
	test.Require(t, err == nil)

	return httpServer, *httpServerURL
}

func newHTTPRequestForTesting(t *testing.T, method, endpoint string, body io.Reader, callbacks ...func(*testing.T, *http.Request)) *http.Request {
	req, err := http.NewRequestWithContext(t.Context(), method, endpoint, body)
	test.Require(t, err == nil)

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
					if errA != nil || errB != nil {
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

type doerSpy struct {
	doer  Doer
	calls []doerSpyCall
}

type doerSpyCall struct {
	req  *http.Request
	resp *http.Response
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
	}{
		req:  req,
		resp: resp,
	})

	return resp, err
}

type doerFail struct {
	err error
}

func (fail *doerFail) Do(*http.Request) (*http.Response, error) { return nil, fail.err }
