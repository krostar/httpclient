package httpclienttest

import (
	"net/http"
	"sync"

	"github.com/krostar/httpclient"
)

// DoerSpy implements httpclient.Doer and records all calls to the underlying Doer.
// Useful for verifying specific HTTP requests were made.
// Safe for concurrent use.
type DoerSpy struct {
	doer httpclient.Doer

	m     sync.Mutex
	calls []DoerSpyRecord
}

// NewDoerSpy creates DoerSpy wrapping the provided Doer.
// Forwards all requests while recording inputs and outputs for inspection.
func NewDoerSpy(doer httpclient.Doer) *DoerSpy {
	return &DoerSpy{doer: doer}
}

// Do implements httpclient.Doer by forwarding the request to the wrapped Doer
// and recording both the input request and output response/error for later retrieval.
// This method is safe for concurrent use.
func (d *DoerSpy) Do(req *http.Request) (*http.Response, error) {
	resp, err := d.doer.Do(req)
	d.m.Lock()
	d.calls = append(d.calls, DoerSpyRecord{
		InputRequest:   req,
		OutputResponse: resp,
		OutputError:    err,
	})
	d.m.Unlock()
	return resp, err
}

// Calls returns a copy of all recorded calls made through this spy and clears
// the internal call history. Each call returns a fresh slice, making it safe
// to call concurrently. The returned slice contains calls in the order they
// were made.
func (d *DoerSpy) Calls() []DoerSpyRecord {
	d.m.Lock()
	defer d.m.Unlock()

	calls := make([]DoerSpyRecord, len(d.calls))
	copy(calls, d.calls)
	d.calls = nil

	return calls
}

// DoerSpyRecord represents a single recorded HTTP request/response interaction.
// It captures the complete input and output of one call to the Do method,
// including any error that may have occurred during the request.
type DoerSpyRecord struct {
	InputRequest   *http.Request
	OutputResponse *http.Response
	OutputError    error
}
