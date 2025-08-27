package example

import (
	"context"
	"net/http"
	"net/url"

	"github.com/krostar/httpclient"
)

// Client provides a high-level interface for interacting with a user management API.
// It encapsulates the httpclient.API instance and provides domain-specific methods
// for common operations like creating, retrieving, and deleting users.
//
// The client is designed to be safe for concurrent use by multiple goroutines,
// as the underlying httpclient.API is thread-safe and creates new request
// builders for each operation.
type Client struct {
	api *httpclient.API
}

// New creates a new Client instance configured to communicate with the API
// server at the specified URL. The client is configured with sensible defaults
// for common HTTP status codes and error handling patterns.
//
// Default response handlers:
//   - 200 OK: Treated as success (no error)
//   - 401 Unauthorized: Returns ErrUnauthorized sentinel error
//   - 404 Not Found: Returns ErrUserNotFound sentinel error
//
// The client uses http.DefaultClient as the underlying HTTP client, which
// provides reasonable defaults for most use cases. For production applications,
// consider configuring a custom HTTP client with appropriate timeouts,
// connection pooling, and other settings.
//
// The returned client is safe for concurrent use by multiple goroutines.
func New(serverURL url.URL) (*Client, error) {
	api := httpclient.NewAPI(http.DefaultClient, serverURL)                                                       // use serverURL as base path
	api = api.WithResponseHandler(http.StatusUnauthorized, func(*http.Response) error { return ErrUnauthorized }) // set default response for status 401
	api = api.WithResponseHandler(http.StatusNotFound, func(*http.Response) error { return ErrUserNotFound })     // set default response for status 404
	api = api.WithResponseHandler(http.StatusOK, func(*http.Response) error { return nil })                       // set default response for status 200
	return &Client{api: api}, nil
}

// CreateUser creates a new user in the system with the specified username.
// It sends a POST request to the /users endpoint with a JSON body containing
// the user creation data and returns the newly assigned user ID.
func (c *Client) CreateUser(ctx context.Context, userName string) (UserID, error) {
	var response apiCreateUserResponse

	// create a post request to /users with provided json body
	// expect a status ok and bind json body to provided dest
	// any non default response status will return an error
	if err := c.api.
		Do(ctx, c.api.
			Post("/users").
			SendJSON(apiCreateUserRequest{UserName: userName}),
		).
		ReceiveJSON(http.StatusOK, &response).
		Error(); err != nil {
		return 0, err
	}

	return UserID(response.UserID), nil
}

// GetUserByID retrieves a user from the system by their unique identifier.
// It sends a GET request to the /users/{userID} endpoint and returns the
// user information as a domain model object.
func (c *Client) GetUserByID(ctx context.Context, userID UserID) (*User, error) {
	var response apiGetUserByIDResponse

	// create a get request to /users/<provided userID>
	// expect a status ok and bind json body to provided dest
	// any non default response status will return an error
	if err := c.api.
		Do(ctx, c.api.
			Get("/users/{userID}").
			PathReplacer("{userID}", userID.String()),
		).
		ReceiveJSON(http.StatusOK, &response).
		Error(); err != nil {
		return nil, err
	}

	return response.ToModel(), nil
}

// DeleteUserByID removes a user from the system by their unique identifier.
// It sends a DELETE request to the /users/{userID} endpoint and relies on
// the default response handlers for error handling.
//
// This method uses the Execute() convenience method instead of Do() because
// DELETE operations typically don't return response bodies that need parsing.
// The Execute() method applies all default response handlers and returns
// only the error result.
func (c *Client) DeleteUserByID(ctx context.Context, userID UserID) error {
	// create a delete request to /users/<provided userID>
	// any non default response status will return an error
	return c.api.
		Execute(ctx, c.api.
			Delete("/users/{userID}").
			PathReplacer("{userID}", userID.String()),
		)
}
