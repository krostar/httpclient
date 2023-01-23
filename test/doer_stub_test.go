package httpclienttest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_DoerStub(t *testing.T) {
	newHTTPRequest := func(t *testing.T, method string) *http.Request {
		req, err := http.NewRequestWithContext(context.Background(), method, "/", nil)
		assert.NilError(t, err)
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
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			assert.Check(t, cmp.ErrorContains(err, "boom"))
			assert.Check(t, resp.StatusCode == http.StatusTeapot)

			assert.Check(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("all calls not made in order", func(t *testing.T) {
			client := NewDoerStub(calls, true)

			resp, err := client.Do(newHTTPRequest(t, http.MethodPost))
			assert.ErrorContains(t, err, "request does not match")
			assert.Check(t, resp == nil)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			assert.Check(t, len(client.RemainingCalls()) == 1)
		})

		t.Run("no remaining calls", func(t *testing.T) {
			client := NewDoerStub(calls, true)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			assert.Check(t, cmp.ErrorContains(err, "boom"))
			assert.Check(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			assert.ErrorContains(t, err, "http doer not configured for this call")
			assert.Check(t, resp == nil)
		})
	})

	t.Run("not strict order", func(t *testing.T) {
		t.Run("all calls made in order", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			assert.Check(t, cmp.ErrorContains(err, "boom"))
			assert.Check(t, resp.StatusCode == http.StatusTeapot)

			assert.Check(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("all calls not made in order", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodPost))
			assert.Check(t, cmp.ErrorContains(err, "boom"))
			assert.Check(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			assert.Check(t, len(client.RemainingCalls()) == 0)
		})

		t.Run("no remaining calls", func(t *testing.T) {
			client := NewDoerStub(calls, false)

			resp, err := client.Do(newHTTPRequest(t, http.MethodGet))
			assert.NilError(t, err)
			assert.Check(t, resp.StatusCode == http.StatusOK)

			resp, err = client.Do(newHTTPRequest(t, http.MethodPost))
			assert.Check(t, cmp.ErrorContains(err, "boom"))
			assert.Check(t, resp.StatusCode == http.StatusTeapot)

			resp, err = client.Do(newHTTPRequest(t, http.MethodGet))
			assert.ErrorContains(t, err, "http doer not configured for this call")
			assert.Check(t, resp == nil)
		})
	})
}
