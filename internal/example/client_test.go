package example

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/krostar/httpclient"
	httpclienttest "github.com/krostar/httpclient/test"
)

func Test_CreateUser(t *testing.T) {
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

	for name, test := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(userID UserID, err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				return json.NewEncoder(rw).Encode(apiCreateUserResponse{UserID: 42})
			},
			check: func(userID UserID, err error) error {
				assert.Check(t, err)
				assert.Check(t, userID == UserID(42))
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(userID UserID, err error) error {
				assert.Check(t, cmp.ErrorIs(err, ErrUnauthorized))
				assert.Check(t, userID == UserID(0))
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(userID UserID, err error) error {
				assert.Check(t, cmp.ErrorContains(err, "unhandled request status"))
				assert.Check(t, userID == UserID(0))
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) { assert.NilError(t, srv.AssertRequest(matcher, test.write, test.check)) })
	}
}

func Test_GetUserByID(t *testing.T) {
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

	for name, test := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(user *User, err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				return json.NewEncoder(rw).Encode(apiGetUserByIDResponse{ID: 42, Name: "john.doe"})
			},
			check: func(user *User, err error) error {
				assert.Check(t, err)
				assert.Check(t, cmp.DeepEqual(user, &User{
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
				assert.Check(t, cmp.ErrorIs(err, ErrUserNotFound))
				assert.Check(t, user == nil)
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(user *User, err error) error {
				assert.Check(t, cmp.ErrorIs(err, ErrUnauthorized))
				assert.Check(t, user == nil)
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(user *User, err error) error {
				assert.Check(t, cmp.ErrorContains(err, "unhandled request status"))
				assert.Check(t, user == nil)
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) { assert.NilError(t, srv.AssertRequest(matcher, test.write, test.check)) })
	}
}

func Test_DeleteUserByID(t *testing.T) {
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

	for name, test := range map[string]struct {
		write func(rw http.ResponseWriter) error
		check func(err error) error
	}{
		"ok": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusOK)
				return nil
			},
			check: func(err error) error {
				assert.Check(t, err)
				return nil
			},
		},
		"ko not found": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusNotFound)
				return nil
			},
			check: func(err error) error {
				assert.Check(t, cmp.ErrorIs(err, ErrUserNotFound))
				return nil
			},
		},
		"ko unauthorized": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusUnauthorized)
				return nil
			},
			check: func(err error) error {
				assert.Check(t, cmp.ErrorIs(err, ErrUnauthorized))
				return nil
			},
		},
		"ko": {
			write: func(rw http.ResponseWriter) error {
				rw.WriteHeader(http.StatusInternalServerError)
				return nil
			},
			check: func(err error) error {
				assert.Check(t, cmp.ErrorContains(err, "unhandled request status"))
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) { assert.NilError(t, srv.AssertRequest(matcher, test.write, test.check)) })
	}
}
