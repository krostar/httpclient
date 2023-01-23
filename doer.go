package httpclient

import (
	"net/http"
)

// Doer defines the Do method, implemented by default by http.DefaultClient.
// Its main usage is to avoid being tightly coupled with http.Client object to allow additional behavior on top of the underlying client.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}
