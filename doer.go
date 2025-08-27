package httpclient

import (
	"net/http"
)

// Doer defines interface for executing HTTP requests.
// Allows httpclient to work with any HTTP client implementation.
// Standard http.Client implements this interface directly.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}
