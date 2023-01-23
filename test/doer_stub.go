package httpclienttest

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// NewDoerStub returns a new stubbed Doer.
// See DoerStubCall for how to configure calls.
func NewDoerStub(calls []DoerStubCall, strictOrder bool) *DoerStub {
	copied := make([]DoerStubCall, len(calls))
	copy(copied, calls)
	return &DoerStub{
		calls:       copied,
		strictOrder: strictOrder,
	}
}

// DoerStub implements Doer and returns pre-configured calls.
// It is safe to call it concurrently.
type DoerStub struct {
	m           sync.Mutex
	strictOrder bool
	calls       []DoerStubCall
}

// Do wraps the underlying doer call and returns pre-configured responses.
// If strictOrder is true, Doer will go through each configured calls in order
// and if the request matcher is set and don't match the request, an error will be returned.
// Otherwise, if strictOrder is false, Doer will go through each configured calls in order
// but if the request matcher is set and don't match the request, the call will be skipped and the next will be tried.
// If no calls are remaining, or if structOrder is false and no call match, an error will be returned.
func (d *DoerStub) Do(req *http.Request) (*http.Response, error) {
	d.m.Lock()
	defer d.m.Unlock()

	idx := -1

	for i, call := range d.calls {
		if call.Matcher == nil {
			idx = i
			break
		}

		if err := call.Matcher.MatchRequest(req); err == nil {
			idx = i
			break
		} else if d.strictOrder {
			return nil, fmt.Errorf("request does not match: %v", err)
		} else {
			continue
		}
	}

	if idx == -1 {
		return nil, errors.New("http doer not configured for this call")
	}

	call := d.calls[idx]
	d.calls = append(d.calls[:idx], d.calls[idx+1:]...)

	return call.Response, call.Error
}

// RemainingCalls returns calls that were not made but configured.
func (d *DoerStub) RemainingCalls() []DoerStubCall {
	d.m.Lock()
	defer d.m.Unlock()

	calls := make([]DoerStubCall, len(d.calls))
	copy(calls, d.calls)
	return calls
}

// DoerStubCall define the configuration of a call.
// If a matcher is not set, the duo 'response,error' will be returned regardless of the request.
// Otherwise, the request will be checked against matcher and duo 'response,error' will be returned only if the request match.
type DoerStubCall struct {
	Matcher RequestMatcher

	Response *http.Response
	Error    error
}
