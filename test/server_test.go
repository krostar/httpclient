package httpclienttest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/krostar/test"

	"github.com/krostar/httpclient"
)

func Test_Server(t *testing.T) {
	createSomething := func(doer httpclient.Doer, u url.URL) (uint, error) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, u.String()+"/foo", http.NoBody)
		if err != nil {
			return 0, fmt.Errorf("unable to create request: %v", err)
		}

		resp, err := doer.Do(req)
		if err != nil {
			return 0, fmt.Errorf("unable to perform request: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusTeapot {
			return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		return 42, nil
	}

	srv := NewServer(func(u url.URL, doer httpclient.Doer, checkResponse any) error {
		if checkResponse == nil {
			return errors.New("boom")
		}

		id, err := createSomething(doer, u)
		checkResponse.(func(uint, error))(id, err)
		return nil
	})

	t.Run("ok", func(t *testing.T) {
		var (
			calledWriteResponse uint
			calledCheckResponse uint
		)

		test.Assert(t, srv.AssertRequest(
			NewRequestMatcherBuilder().URLPath("/foo"),
			func(rw http.ResponseWriter) error {
				calledWriteResponse++
				rw.WriteHeader(http.StatusTeapot)
				return nil
			},
			func(id uint, err error) {
				calledCheckResponse++
				test.Assert(t, id == 42)
				test.Require(t, err == nil)
			}) == nil,
		)
		test.Assert(t, calledWriteResponse == 1)
		test.Assert(t, calledCheckResponse == 1)
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("unable to create doer", func(t *testing.T) {
			err := srv.AssertRequest(nil, nil, nil)
			test.Assert(t, err != nil && strings.Contains(err.Error(), "doer execution failed: boom"))
		})

		t.Run("request does not match", func(t *testing.T) {
			err := srv.AssertRequest(NewRequestMatcherBuilder().URLPath("/notfoo"), nil, func(uint, error) {})
			test.Assert(t, err != nil && strings.Contains(err.Error(), "request does not match"))
		})

		t.Run("unable to write response", func(t *testing.T) {
			err := srv.AssertRequest(
				NewRequestMatcherBuilder().URLPath("/foo"),
				func(http.ResponseWriter) error {
					return errors.New("boom")
				},
				func(uint, error) {})
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to write response: boom"))
		})
	})
}
