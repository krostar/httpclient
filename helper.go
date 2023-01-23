package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ParsePostForm sets req.PostForm by calling req.ParseForm forms, but also handles non-standard http methods.
// Like req.ParseForm, ParsePostForm is idempotent.
func ParsePostForm(req *http.Request) error {
	if req.PostForm != nil {
		return nil
	}

	if err := req.ParseForm(); err != nil {
		return fmt.Errorf("unable to parse body as form: %v", err)
	}

	switch {
	case req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH": // handled by ParseForm
	default:
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("unable to read body: %v", err)
		}

		values, err := url.ParseQuery(string(body))
		if err != nil {
			return fmt.Errorf("unable to parse form values from body: %v", err)
		}

		req.PostForm = values
	}

	return nil
}
