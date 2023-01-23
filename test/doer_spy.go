package httpclienttest

import (
	"net/http"
	"sync"

	"github.com/krostar/httpclient"
)

// NewDoerSpy creates a new spy on provided Doer.
// Spied doer keep track of input and output made on the underlying doer.
func NewDoerSpy(doer httpclient.Doer) *DoerSpy {
	return &DoerSpy{doer: doer}
}

// DoerSpy implements Doer and exposes calls made on the underlying Doer.
// It is safe to call it concurrently.
type DoerSpy struct {
	doer httpclient.Doer

	m     sync.Mutex
	calls []DoerSpyRecord
}

// Do wraps the underlying doer call and keep track of input and outputs.
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

// Calls returns the list of calls made on the Doer and clears the list.
func (d *DoerSpy) Calls() []DoerSpyRecord {
	d.m.Lock()
	defer d.m.Unlock()

	calls := make([]DoerSpyRecord, len(d.calls))
	copy(calls, d.calls)
	d.calls = nil

	return calls
}

// DoerSpyRecord stores input and outputs of one Doer call.
type DoerSpyRecord struct {
	InputRequest   *http.Request
	OutputResponse *http.Response
	OutputError    error
}
