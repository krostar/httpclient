package httpclient

import (
	"encoding/base64"
	"net/http"
	"net/http/httputil"
)

// DoerWrapDumpB64 wraps Doer with request/response dumping capability.
// Captures complete HTTP traffic (headers and bodies) as base64 strings.
//
// Useful for debugging, logging, or testing HTTP traffic inspection.
// Dumps include all headers and body content.
//
// If dumpFunc is nil, no dumping occurs but wrapper still applied.
// Uses httputil.DumpRequestOut and httputil.DumpResponse.
func DoerWrapDumpB64(doer Doer, dumpFunc func(requestB64, responseB64 string)) Doer {
	if dumpFunc == nil {
		dumpFunc = func(string, string) {}
	}

	doer = &doerWrapDump64{
		doer: doer,
		dump: dumpFunc,
	}

	return doer
}

// doerWrapDump64 implements Doer, wrapping another Doer with
// base64-encoded dumping of HTTP requests and responses.
type doerWrapDump64 struct {
	doer Doer
	dump func(string, string)
}

// Do implements Doer interface, executing request through wrapped Doer
// while capturing and dumping request/response as base64 strings.
func (w doerWrapDump64) Do(req *http.Request) (*http.Response, error) {
	requestB64 := w.request(req)
	resp, err := w.doer.Do(req)
	responseB64 := w.response(resp)

	w.dump(requestB64, responseB64)

	return resp, err
}

// request creates base64-encoded dump of HTTP request using httputil.DumpRequestOut.
// Returns empty string if req is nil, error message if dumping fails.
func (doerWrapDump64) request(req *http.Request) string {
	if req == nil {
		return ""
	}

	var out []byte

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		out = dump
	} else {
		out = []byte("unable to dump request: " + err.Error())
	}

	return base64.StdEncoding.EncodeToString(out)
}

// response creates base64-encoded dump of HTTP response using httputil.DumpResponse.
// Returns empty string if resp is nil, error message if dumping fails.
func (doerWrapDump64) response(resp *http.Response) string {
	if resp == nil {
		return ""
	}

	var out []byte

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		out = dump
	} else {
		out = []byte("unable to dump response: " + err.Error())
	}

	return base64.StdEncoding.EncodeToString(out)
}
