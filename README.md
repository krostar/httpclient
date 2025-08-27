[![License](https://img.shields.io/badge/license-MIT-blue)](https://choosealicense.com/licenses/mit/)
![go.mod Go version](https://img.shields.io/github/go-mod/go-version/krostar/httpclient?label=go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/krostar/httpclient)
[![Latest tag](https://img.shields.io/github/v/tag/krostar/httpclient)](https://github.com/krostar/httpclient/tags)
[![Go Report](https://goreportcard.com/badge/github.com/krostar/httpclient)](https://goreportcard.com/report/github.com/krostar/httpclient)

# httpclient

A minimalist, fluent HTTP client Go library that simplifies request creation and response handling through builder patterns.

## Overview

This package provides a chainable API that reduces boilerplate code while maintaining type safety. It makes HTTP client code more readable and testable by offering:

- **Fluent Interface**: Chain method calls for readable request building.
- **Type Safety**: Compile-time checks for request/response handling.
- **Testing Support**: Built-in utilities for mocking and testing HTTP interactions.
- **Flexibility**: Works with any HTTP client implementing the `Doer` interface.

## Installation

```bash
go get github.com/krostar/httpclient
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/krostar/httpclient"
)

func main() {
    var user User
    err := httpclient.NewRequest("GET", "https://api.example.com/users/123").
        Do(context.Background()).
        ReceiveJSON(200, &user).
        Error()

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %+v\n", user)
}
```

## Core Concepts

- **RequestBuilder**: provides a fluent interface for constructing HTTP requests.
- **ResponseBuilder**: handles HTTP responses with status-specific logic.
- **API**: provides request defaults and reusable configuration.

## Examples

### Simple GET Request

Fetch a user by ID:

```go
var user User
err := httpclient.NewRequest("GET", "https://api.example.com/users/123").
    Do(ctx).
    ReceiveJSON(200, &user).
    Error()
```

### POST with JSON

Create a new user:

```go
newUser := CreateUserRequest{Name: "John", Email: "john@example.com"}
var response CreateUserResponse

err := httpclient.NewRequest("POST", "https://api.example.com/users").
    SendJSON(&newUser).
    Do(ctx).
    ReceiveJSON(201, &response).
    Error()
```

### Complex Request with Error Handling

A more comprehensive example showing headers, query parameters, and error handling:

```go
var users []User
err := httpclient.NewRequest("GET", "https://api.example.com/users").
    SetHeader("Authorization", "Bearer "+token).
    SetHeader("User-Agent", "MyApp/1.0").
    SetQueryParam("page", "1").
    SetQueryParam("limit", "10").
    Do(ctx).
    ReceiveJSON(200, &users).
    SuccessOnStatus(304). // 304 Not Modified is also success
    ErrorOnStatus(401, ErrUnauthorized).
    ErrorOnStatus(403, ErrForbidden).
    ErrorOnStatus(429, ErrRateLimited).
    Error()
```

### Using the API Type

For applications making multiple requests to the same service, use the `API` type to reduce duplication:

```go
// Create reusable API client
api := httpclient.NewAPI(http.DefaultClient, url.URL{
    Scheme: "https",
    Host:   "api.example.com",
}).
    WithRequestHeaders(http.Header{
        "Authorization": []string{"Bearer " + token},
        "User-Agent":    []string{"MyApp/1.0"},
    }).
    WithResponseHandler(401, func(resp *http.Response) error {
        return ErrUnauthorized
    }).
    WithResponseBodySizeReadLimit(1024 * 1024) // 1MB limit

// Use the API for multiple requests
var user User
err := api.Do(ctx, api.Get("/users/123")).
    ReceiveJSON(200, &user).
    Error()

var users []User
err = api.Do(ctx, api.Get("/users").
    SetQueryParam("page", "1")).
    ReceiveJSON(200, &users).
    Error()
```

## Testing

The `httpclienttest` package provides utilities for testing HTTP client code (`DoerStub`, `DoerSpy`, request matching, ...).


## Comparison

### The classic way using standard library

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
    // handled below
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

This approach is straightforward but verbose:

- create the JSON body
- create the request with proper headers and context propagation
- perform the request using the HTTP client
- jandle errors with specific logic for authentication failures
- parse the JSON response body on success
- return the parsed user ID

However, this approach has several drawbacks: extensive boilerplate code, complex testing requirements, manual verification of context usage and headers, and potential security issues like unrestricted response body reading.

### Developer-friendly alternative

The same functionality using httpclient:

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

This approach is significantly more concise, readable, and maintainable.

For applications making multiple requests to the same API, create a reusable API object:

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

This object supports extensive configuration options to define default behavior for all requests:

Usage example:

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

This reduces duplication by centralizing error handling and request attributes, eliminates repeated API addresses, and simplifies testing.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
