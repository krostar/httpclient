package example

import (
	"context"
	"net/http"
	"net/url"

	"github.com/krostar/httpclient"
)

type Client struct {
	api *httpclient.API
}

func New(serverURL url.URL) (*Client, error) {
	api := httpclient.NewAPI(http.DefaultClient, serverURL)                                                       // use serverURL as base path
	api = api.WithResponseHandler(http.StatusUnauthorized, func(*http.Response) error { return ErrUnauthorized }) // set default response
	api = api.WithResponseHandler(http.StatusNotFound, func(*http.Response) error { return ErrUserNotFound })     // set default response
	api = api.WithResponseHandler(http.StatusOK, func(*http.Response) error { return nil })                       // set default response
	return &Client{api: api}, nil
}

func (c *Client) CreateUser(ctx context.Context, userName string) (UserID, error) {
	var response apiCreateUserResponse

	// create a post request to /users with provided json body, expect a status ok and bind json body to provided dest
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

func (c *Client) GetUserByID(ctx context.Context, userID UserID) (*User, error) {
	var response apiGetUserByIDResponse

	// create a get request to /users/<provided userID>, expect a status ok and bind json body to provided dest
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

func (c *Client) DeleteUserByID(ctx context.Context, userID UserID) error {
	// create a delete request to /users/<provided userID>
	// any non default response status will return an error
	return c.api.
		Execute(ctx, c.api.
			Delete("/users/{userID}").
			PathReplacer("{userID}", userID.String()),
		)
}
