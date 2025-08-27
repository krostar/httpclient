package httpclienttest

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/krostar/test"
)

func Test_DoerStub(t *testing.T) {
	newHTTPRequest := func(t *testing.T, method string) *http.Request {
		req, err := http.NewRequestWithContext(t.Context(), method, "/", http.NoBody)
		test.Require(t, err == nil)
		return req
	}

	calls := []DoerStubCall{
		{
			Matcher:  NewRequestMatcherBuilder().Method(http.MethodGet),
			Response: &http.Response{StatusCode: http.StatusOK},
			Error:    nil,
		}, {
			Matcher:  nil,
			Response: &http.Response{StatusCode: http.StatusTeapot},
			Error:    errors.New("boom"),
		},
	}

	t.Run("strict order", func(t *testing.T) {
		t.Run("all calls made in order", func(t *testing.T) {
			client := NewDoerStub(calls, true)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			test.Require(t, err != nil && strings.Contains(err.Error(), "boom"))
			test.Assert(t, resp.StatusCode == http.StatusTeapot)

			test.Assert(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("all calls not made in order", func(t *testing.T) {
			client := NewDoerStub(calls, true)

			resp, err := client.Do(newHTTPRequest(t, http.MethodPost))
			test.Require(t, err != nil && strings.Contains(err.Error(), "request does not match"))
			test.Assert(t, resp == nil)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			test.Assert(t, len(client.RemainingCalls()) == 1)
		})

		t.Run("no remaining calls", func(t *testing.T) {
			client := NewDoerStub(calls, true)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			test.Require(t, err != nil && strings.Contains(err.Error(), "boom"))
			test.Assert(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err != nil && strings.Contains(err.Error(), "http doer not configured for this call"))
			test.Assert(t, resp == nil)
		})
	})

	t.Run("not strict order", func(t *testing.T) {
		t.Run("all calls made in order", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			test.Assert(t, err != nil && strings.Contains(err.Error(), "boom"))
			test.Require(t, resp.StatusCode == http.StatusTeapot)

			test.Assert(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("all calls not made in order", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodPost))
			test.Require(t, err != nil && strings.Contains(err.Error(), "boom"))
			test.Assert(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			test.Assert(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("no remaining calls", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err == nil)
			test.Assert(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			test.Require(t, err != nil && strings.Contains(err.Error(), "boom"))
			test.Assert(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			test.Require(t, err != nil && strings.Contains(err.Error(), "http doer not configured for this call"))
			test.Assert(t, resp == nil)
		})
	})
}
