package httpclienttest

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// DoerStub implements httpclient.Doer and returns pre-configured responses
// without making actual HTTP requests. Useful for controlling exact responses
// or testing error conditions.
//
// Two modes: strict order (exact consumption order) or flexible order (first match).
// Safe for concurrent use.
type DoerStub struct {
	m           sync.Mutex
	strictOrder bool
	calls       []DoerStubCall
}

// NewDoerStub creates DoerStub with provided call configurations.
// Calls slice defines responses for matching requests.
//
// If strictOrder is true, calls consumed in exact order, non-matches error.
// If false, first matching call (or no matcher) used and consumed.
// See DoerStubCall for configuring individual responses.
func NewDoerStub(calls []DoerStubCall, strictOrder bool) *DoerStub {
	copied := make([]DoerStubCall, len(calls))
	copy(copied, calls)
	return &DoerStub{
		calls:       copied,
		strictOrder: strictOrder,
	}
}

// Do implements httpclient.Doer by returning pre-configured responses based on
// the configured DoerStubCall slice. The behavior depends on the strictOrder setting:
//
// In strict order mode (strictOrder=true):
//   - calls are processed in the exact order they were configured
//   - if a call has a RequestMatcher and the request doesn't match, an error is returned
//   - if a call has no RequestMatcher, it matches any request
//
// In flexible order mode (strictOrder=false):
//   - calls are searched in order until a matching one is found
//   - non-matching calls are skipped
//   - the first matching call (or call without a matcher) is used and consumed
//
// Returns an error if no configured calls remain or if no call matches the request.
// This method is safe for concurrent use.
func (d *DoerStub) Do(req *http.Request) (*http.Response, error) {
	d.m.Lock()
	defer d.m.Unlock()

	idx := -1

	for i, call := range d.calls {
		if call.Matcher == nil {
			idx = i
			break
		}

		if err := call.Matcher.MatchRequest(req); err != nil {
			if d.strictOrder {
				return nil, fmt.Errorf("request does not match: %v", err)
			}
			continue
		}

		idx = i

		break
	}

	if idx == -1 {
		return nil, errors.New("http doer not configured for this call")
	}

	call := d.calls[idx]
	d.calls = append(d.calls[:idx], d.calls[idx+1:]...)

	return call.Response, call.Error
}

// RemainingCalls returns a copy of all DoerStubCall configurations that have
// not yet been consumed by calls to Do. This is useful in tests to verify
// that all expected HTTP calls were actually made. The returned slice is a
// copy and safe to modify. This method is safe for concurrent use.
func (d *DoerStub) RemainingCalls() []DoerStubCall {
	d.m.Lock()
	defer d.m.Unlock()

	calls := make([]DoerStubCall, len(d.calls))
	copy(calls, d.calls)
	return calls
}

// DoerStubCall defines the configuration for a single stubbed HTTP call.
// It consists of an optional RequestMatcher to determine if an incoming request
// should use this configuration, and a Response/Error pair to return.
//
// If Matcher is nil, this call configuration will match any request.
// If Matcher is set, the incoming request must pass all the matcher's
// assertions for this configuration to be used.
type DoerStubCall struct {
	Matcher RequestMatcher

	Response *http.Response
	Error    error
}
