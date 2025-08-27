package example

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/krostar/test"
	"github.com/krostar/test/check"

	"github.com/krostar/httpclient"
	httpclienttest "github.com/krostar/httpclient/test"
)

func Test_Client_CreateUser(t *testing.T) {
	srv := httpclienttest.NewServer(func(serverAddress url.URL, serverDoer httpclient.Doer, checkResponseFunc any) error {
		client, err := New(serverAddress)
		if err != nil {
			return err
		}

		userID, err := client.CreateUser(context.Background(), "john.doe")
		return checkResponseFunc.(func(UserID, error) error)(userID, err)
	})

	matcher := httpclienttest.
		NewRequestMatcherBuilder().
		Method(http.MethodPost).
		URLPath("/users").
		BodyJSON(
			&apiCreateUserRequest{UserName: "john.doe"},
			func() any { return new(apiCreateUserRequest) },
			true,
		)

	for name, tt := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(userID UserID, err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				return json.NewEncoder(rw).Encode(apiCreateUserResponse{UserID: 42})
			},
			check: func(userID UserID, err error) error {
				test.Require(t, err == nil)
				test.Assert(t, userID == UserID(42))
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(userID UserID, err error) error {
				test.Assert(t, err != nil && errors.Is(err, ErrUnauthorized))
				test.Assert(t, userID == UserID(0))
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(userID UserID, err error) error {
				test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status"), err)
				test.Assert(t, userID == 0)
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			test.Assert(t, srv.AssertRequest(matcher, tt.write, tt.check) == nil)
		})
	}
}

func Test_Client_GetUserByID(t *testing.T) {
	srv := httpclienttest.NewServer(func(serverAddress url.URL, serverDoer httpclient.Doer, checkResponseFunc any) error {
		client, err := New(serverAddress)
		if err != nil {
			return err
		}

		user, err := client.GetUserByID(context.Background(), 42)
		return checkResponseFunc.(func(*User, error) error)(user, err)
	})

	matcher := httpclienttest.
		NewRequestMatcherBuilder().
		Method(http.MethodGet).
		URLPath("/users/42")

	for name, tt := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(user *User, err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				return json.NewEncoder(rw).Encode(apiGetUserByIDResponse{ID: 42, Name: "john.doe"})
			},
			check: func(user *User, err error) error {
				test.Require(t, err == nil)
				test.Assert(check.Compare(t, user, &User{
					ID:   42,
					Name: "john.doe",
				}))
				return nil
			},
		},
		"ko not found": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusNotFound)
				return nil
			},
			check: func(user *User, err error) error {
				test.Assert(t, err != nil && errors.Is(err, ErrUserNotFound))
				test.Assert(t, user == nil)
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(user *User, err error) error {
				test.Assert(t, err != nil && errors.Is(err, ErrUnauthorized))
				test.Assert(t, user == nil)
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(user *User, err error) error {
				test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status"))
				test.Assert(t, user == nil)
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) { test.Assert(t, srv.AssertRequest(matcher, tt.write, tt.check) == nil) })
	}
}

func Test_Client_DeleteUserByID(t *testing.T) {
	srv := httpclienttest.NewServer(func(serverAddress url.URL, serverDoer httpclient.Doer, checkResponseFunc any) error {
		client, err := New(serverAddress)
		if err != nil {
			return err
		}

		err = client.DeleteUserByID(context.Background(), 42)
		return checkResponseFunc.(func(error) error)(err)
	})

	matcher := httpclienttest.
		NewRequestMatcherBuilder().
		Method(http.MethodDelete).
		URLPath("/users/42")

	for name, tt := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusOK)
				return nil
			},
			check: func(err error) error {
				test.Require(t, err == nil)
				return nil
			},
		},
		"ko not found": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusNotFound)
				return nil
			},
			check: func(err error) error {
				test.Assert(t, err != nil && errors.Is(err, ErrUserNotFound))
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(err error) error {
				test.Assert(t, err != nil && errors.Is(err, ErrUnauthorized))
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(err error) error {
				test.Assert(t, err != nil && strings.Contains(err.Error(), "unhandled status"))
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			test.Assert(t, srv.AssertRequest(matcher, tt.write, tt.check) == nil)
		})
	}
}
