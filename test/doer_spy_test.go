package httpclienttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_DoerSpy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	spiedClient := NewDoerSpy(srv.Client())

	var calls []DoerSpyRecord

	{ // first call
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/foo", http.NoBody)
		test.Assert(t, err == nil)

		resp, err := spiedClient.Do(req)
		defer func() { _ = resp.Body.Close() }()

		calls = append(calls, DoerSpyRecord{
			InputRequest:   req,
			OutputResponse: resp,
			OutputError:    err,
		})
	}

	{ // second call
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/bar", http.NoBody)
		test.Assert(t, err == nil)

		resp, err := spiedClient.Do(req)
		defer func() { _ = resp.Body.Close() }()

		calls = append(calls, DoerSpyRecord{
			InputRequest:   req,
			OutputResponse: resp,
			OutputError:    err,
		})
	}

	test.Assert(check.Compare(t, spiedClient.Calls(), calls,
		gocmp.AllowUnexported(http.Request{}),
		gocmp.FilterValues(
			func(_, _ context.Context) bool { return true },
			gocmp.Ignore(),
		),
		gocmpopts.EquateErrors(),
	))
	test.Assert(t, len(spiedClient.Calls()) == 0)
}
