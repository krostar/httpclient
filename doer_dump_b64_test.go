package httpclient

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_DoerWrapDumpB64(t *testing.T) {
	httpServer, httpServerURL := newHTTPServerForTesting(t, func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
		_, err := rw.Write([]byte(`"hello world"`))
		assert.NilError(t, err)
	})

	callback := func(req, resp string) {
		reqDecoded, err := base64.StdEncoding.DecodeString(req)
		assert.NilError(t, err)
		assert.Check(t, strings.Contains(string(reqDecoded), "POST /foo HTTP/1.1"))
		assert.Check(t, strings.Contains(string(reqDecoded), "Host:"))
		assert.Check(t, strings.Contains(string(reqDecoded), "User-Agent:"))
		assert.Check(t, strings.Contains(string(reqDecoded), "hi!"))

		respDecoded, err := base64.StdEncoding.DecodeString(resp)
		assert.NilError(t, err)
		assert.Check(t, strings.Contains(string(respDecoded), "HTTP/1.1 418 I'm a teapot"))
		assert.Check(t, strings.Contains(string(respDecoded), "Content-Length:"))
		assert.Check(t, strings.Contains(string(respDecoded), `"hello world"`))
	}

	req := newHTTPRequestForTesting(t, http.MethodPost, httpServerURL.String()+"/foo", strings.NewReader("hi!"))
	resp, err := DoerWrapDumpB64(httpServer.Client(), callback).Do(req)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusTeapot)
	assert.NilError(t, resp.Body.Close())
}
