package httpclient

import (
	"encoding/base64"
	"net/http"
	"net/http/httputil"
)

// DoerWrapDumpB64 wraps the provided doer by calling a callback with a base64 encoded dump of the request and response.
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

type doerWrapDump64 struct {
	doer Doer
	dump func(string, string)
}

func (w doerWrapDump64) Do(req *http.Request) (*http.Response, error) {
	requestB64 := w.request(req)
	resp, err := w.doer.Do(req)
	responseB64 := w.response(resp)

	w.dump(requestB64, responseB64)

	return resp, err
}

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

func (doerWrapDump64) response(resp *http.Response) string {
	if resp == nil {
		return ""
	}

	var out []byte

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		out = dump
	} else {
		out = []byte("unable to dump response:" + err.Error())
	}

	return base64.StdEncoding.EncodeToString(out)
}
