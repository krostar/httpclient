[![License](https://img.shields.io/badge/license-MIT-blue)](https://choosealicense.com/licenses/mit/)
![go.mod Go version](https://img.shields.io/github/go-mod/go-version/krostar/httpclient?label=go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/krostar/httpclient)
[![Latest tag](https://img.shields.io/github/v/tag/krostar/httpclient)](https://github.com/krostar/httpclient/tags)
[![Go Report](https://goreportcard.com/badge/github.com/krostar/httpclient)](https://goreportcard.com/report/github.com/krostar/httpclient)

# httpclient

This package offers an easy way of building http request and handling http responses.

The main goal is to simplify the amount of code required to perform simple and complex http request, which reduce development time, ease maintenance and tests.

## Example 1: an easy one

Say we want to perform a **POST** request to **example.com** on **/users/** to create a user using its email in the json body.
This endpoint would return a **201 Created** if the creation succeeded, or an error status, with a special **401 Unauthorized** to catch.

### the classic way using standard library

A typical go code would look like this:

```go
func performUserCreationRequest(ctx context.Context, userEmail string) (uint64, error) {
	// serialization of the request content
	body, err := json.Marshal(&UserCreationRequest{Email: userEmail})
	if err != nil {
		return 0, fmt.Errorf("unable to serialize in json: %v", err)
	}

	// create the request using the provided context to respect cancellation or deadlines
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.com/users", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("unable to create the request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// the client used is hardcoded to be http.Default client but in real-life scenario it is probably injected somehow to ease tests
	client := http.DefaultClient
	// perform the request
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("unable to perform request: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		// will be handled below
	case http.StatusUnauthorized:
		return 0, ErrUnauthorizedRequest
	default:
		return 0, fmt.Errorf("unhandled http status: %d", resp.StatusCode)
	}

	// deserialize the response
	var userCreationResponse UserCreationResponse
	if err := json.NewDecoder(resp.Body).Decode(&userCreationResponse); err != nil {
		return 0, fmt.Errorf("unable to deserialize json: %v", err)
	}

	// return the newly generated user id
	return userCreationResponse.UserID, nil
}
```

This is straightforward:
- create the json body
- create the request (don't forget some useful headers, don't forget the context propagation using NewRequestWithContext)
- perform the request on the provided http client
- handle errors (and treat differently authentication related error to return a sentinel error)
- handle success by parsing the json body
- return the parsed user id

but as much as this is a lovely and regular go code, this is a lot of code, and it is not perfect
(testing each branches takes some effort, reviewers has to check the request is created using the WithContext method,
and check for the right headers, ..., security issues related to not handling huge body by restricting body readers for instance, ...).

### a developer friendly way

Here is the same code using this package:

```go
func performUserCreationRequest(ctx context.Context, userEmail string) (uint64, error) {
	var resp CreateUserResponse

	if err := httpclient.NewRequest(http.MethodPost, "https://example.com/users/").
		SendJSON(&CreateUserRequest{Email: userEmail}).
		Do(ctx).
		ReceiveJSON(http.StatusCreated, &resp).
		ErrorOnStatus(http.StatusUnauthorized, ErrUnauthorizedRequest).
		Error(); err != nil {
		return 0, err
	}

	return resp.UserID, nil
}
```

which is a lot easier to write, read, and test.

To go even further, if we write multiple method for the same API, we can create an API object like:
```go 
api := httpclient.
	NewAPI(client, url.URL{
		Scheme: "https",
		Host:   "example.com",
	}).
	WithResponseHandler(http.StatusUnauthorized, func(rw *http.Response) error {
		return ErrUnauthorizedRequest
	})
```

this object can be configured using a lot of options to define some default behavior for each request,

it can be used like this:

```go
func (api myAPIMethods) performUserCreationRequest(ctx context.Context, userEmail string) (uint64, error) {
	var resp CreateUserResponse

	if err := api.
		Do(ctx, api.Post("/users/").SendJSON(&CreateUserRequest{Email: userEmail})).
		ReceiveJSON(http.StatusCreated, &resp).
		Error(); err != nil {
		return 0, err
	}

	return resp.UserID, nil
}
```

which reduce duplication of error handling by setting common handler for common response handler, or by setting common request attributes.
This is also useful to avoid repeating the API address and to reduce number of test cases.

## Example 2: a more complex one

Say we want to perform a **PUT** request to **example.com** on **/users/$userID/email** to update the email of the user identified using its `$userID`,
with a json body containing an email attribute. This endpoint would return a **204 No Content** if the update was taken into account immediately and in success,
or a status **202 Accepted** meaning the update was successful but the propagation of this update is yet to be done (the response body would for this case contain an estimated date and time after which the propagation should be done).

### the classic way using standard library

A typical go code would look like this:

```go
func performUpdateUserEmailRequest(ctx context.Context, userID uint64, userEmail string) (bool, error) {
	// serialization of the request parameters and content
	strUserID := strconv.FormatUint(userID, 10)

	body, err := json.Marshal(&UpdateUserEmailRequest{Email: userEmail})
	if err != nil {
		return false, fmt.Errorf("unable to serialize in json: %v", err)
	}

	// create the request using the provided context to respect cancellation or deadlines
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "https://example.com/users/"+strUserID+"/email", bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("unable to create the request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// the client used is hardcoded to be http.Default client but in real-life scenario it is probably injected somehow to ease tests
	client := http.DefaultClient
	// perform the request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("unable to perform request: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusAccepted:
		// will be handled below
	case http.StatusUnauthorized:
		return false, ErrUnauthorizedRequest
	default:
		return false, fmt.Errorf("unhandled http status: %d", resp.StatusCode)
	}

	// deserialize the 201 response
	var updateUserEmailResponse UpdateUserEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateUserEmailResponse); err != nil {
		return false, fmt.Errorf("unable to deserialize json: %v", err)
	}

	// return whenever the propagation time is in the future or not
	return updateUserEmailResponse.PropagationTime.Before(time.Now()), nil
}
```

### the same using this package

Here is the same code using the request builder and response handler:

```go
func performUpdateUserEmailRequest(ctx context.Context, userID uint64, userEmail string) (bool, error) {
	var resp UpdateUserEmailResponse

	if err := httpclient.NewRequest(http.MethodPut, "http://example.com/users/{userID}/email").
		PathReplacer("{userID}", strconv.FormatUint(userID, 10)).
		SendJSON(&UpdateUserEmailRequest{Email: userEmail}).
		Do(ctx).
		SuccessOnStatus(http.StatusOK).
		ReceiveJSON(http.StatusAccepted, &resp).
		ErrorOnStatus(http.StatusUnauthorized, ErrUnauthorizedRequest).
		Error(); err != nil {
		return false, err
	}

	return resp.PropagationTime.Before(time.Now()), nil
}
```

### More ?

More examples on the API object and on how to ease tests can be browsed in `./internal/example` package.

## Contribution

This project is using nix, which mean by using `nix develop` you can have the same development environment than me or GitHub Action.
It contains everything needed to develop, lint, and test.

You can also use `act` to run the same steps GitHub Action is using during pull requests, but locally.

Feel free to open issues and pull requests!