package httpclient

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/krostar/test"
)

func Test_DoerWrapDumpB64(t *testing.T) {
	httpServer, httpServerURL := newHTTPServerForTesting(t, func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
		_, err := rw.Write([]byte(`"hello world"`))
		test.Require(t, err == nil)
	})

	callback := func(req, resp string) {
		reqDecoded, err := base64.StdEncoding.DecodeString(req)
		test.Require(t, err == nil)
		test.Assert(t, strings.Contains(string(reqDecoded), "POST /foo HTTP/1.1"))
		test.Assert(t, strings.Contains(string(reqDecoded), "Host:"))
		test.Assert(t, strings.Contains(string(reqDecoded), "User-Agent:"))
		test.Assert(t, strings.Contains(string(reqDecoded), "hi!"))

		respDecoded, err := base64.StdEncoding.DecodeString(resp)
		test.Require(t, err == nil)
		test.Assert(t, strings.Contains(string(respDecoded), "HTTP/1.1 418 I'm a teapot"))
		test.Assert(t, strings.Contains(string(respDecoded), "Content-Length:"))
		test.Assert(t, strings.Contains(string(respDecoded), `"hello world"`))
	}

	req := newHTTPRequestForTesting(t, http.MethodPost, httpServerURL.String()+"/foo", strings.NewReader("hi!"))
	resp, err := DoerWrapDumpB64(httpServer.Client(), callback).Do(req)
	test.Require(t, err == nil && resp != nil)
	test.Assert(t, resp.StatusCode == http.StatusTeapot)
	test.Assert(t, resp.Body.Close() == nil)
}
