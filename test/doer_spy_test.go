package httpclienttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_DoerSpy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	spiedClient := NewDoerSpy(srv.Client())

	var calls []DoerSpyRecord

	{ // first call
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/foo", nil)
		assert.NilError(t, err)

		resp, err := spiedClient.Do(req)
		defer func() { _ = resp.Body.Close() }()

		calls = append(calls, DoerSpyRecord{
			InputRequest:   req,
			OutputResponse: resp,
			OutputError:    err,
		})
	}

	{ // second call
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/bar", nil)
		assert.NilError(t, err)

		resp, err := spiedClient.Do(req)
		defer func() { _ = resp.Body.Close() }()

		calls = append(calls, DoerSpyRecord{
			InputRequest:   req,
			OutputResponse: resp,
			OutputError:    err,
		})
	}

	assert.Check(t, cmp.DeepEqual(spiedClient.Calls(), calls,
		gocmp.AllowUnexported(http.Request{}),
		cmpopts.EquateErrors(),
	))
	assert.Check(t, len(spiedClient.Calls()) == 0)
}
