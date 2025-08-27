// Package httpclient provides a fluent, builder-pattern HTTP client.
//
// Simplifies HTTP request creation and response handling with a chainable API.
// Reduces boilerplate code while maintaining flexibility and type safety.
// Makes HTTP client code more readable, testable, and maintainable.
package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ParsePostForm parses request body as form data and populates req.PostForm.
// Extends standard library to handle non-standard HTTP methods.
//
// Standard library only parses body for POST, PUT, PATCH.
// Other methods (like DELETE) ignore body even with form data.
// This function fills that gap.
//
// Behavior:
//   - POST, PUT, PATCH: Delegates to standard req.ParseForm()
//   - Other methods: Manually parses request body as form data
//   - Idempotent (safe to call multiple times)
//   - Does nothing if req.PostForm already populated
//
// Populates req.PostForm with parsed values.
func ParsePostForm(req *http.Request) error {
	if req.PostForm != nil {
		return nil
	}

	if err := req.ParseForm(); err != nil {
		return fmt.Errorf("unable to parse body as form: %w", err)
	}

	switch req.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch: // handled by ParseForm
	default:
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("unable to read body: %w", err)
		}

		values, err := url.ParseQuery(string(body))
		if err != nil {
			return fmt.Errorf("unable to parse form values from body: %w", err)
		}

		req.PostForm = values
	}

	return nil
}
