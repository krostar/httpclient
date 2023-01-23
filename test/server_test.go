package httpclienttest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/krostar/httpclient"
)

func Test_TestingServer(t *testing.T) {
	createSomething := func(doer httpclient.Doer, u url.URL) (uint, error) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String()+"/foo", nil)
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
			return fmt.Errorf("boom")
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

		assert.NilError(t, srv.AssertRequest(
			NewRequestMatcherBuilder().URLPath("/foo"),
			func(rw http.ResponseWriter) error {
				calledWriteResponse++
				rw.WriteHeader(http.StatusTeapot)
				return nil
			},
			func(id uint, err error) {
				calledCheckResponse++
				assert.Check(t, id == 42)
				assert.Check(t, err == nil)
			}),
		)
		assert.Check(t, calledWriteResponse == 1)
		assert.Check(t, calledCheckResponse == 1)
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("unable to create doer", func(t *testing.T) {
			assert.ErrorContains(t,
				srv.AssertRequest(nil, nil, nil),
				"doer execution failed: boom",
			)
		})

		t.Run("request does not match", func(t *testing.T) {
			assert.ErrorContains(t,
				srv.AssertRequest(NewRequestMatcherBuilder().URLPath("/notfoo"), nil, func(uint, error) {}),
				"request does not match",
			)
		})

		t.Run("unable to write response", func(t *testing.T) {
			assert.ErrorContains(t, srv.AssertRequest(
				NewRequestMatcherBuilder().URLPath("/foo"),
				func(http.ResponseWriter) error {
					return errors.New("boom")
				},
				func(uint, error) {}),
				"unable to write response: boom",
			)
		})
	})
}
