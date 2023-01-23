package httpclienttest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_RequestMatcherBuilder(t *testing.T) {
	jsonEncode := func(t *testing.T, dest any) io.Reader {
		buf := new(bytes.Buffer)
		assert.Check(t, json.NewEncoder(buf).Encode(dest))
		return buf
	}

	newRequest := func(method, endpoint string, body io.Reader) *http.Request {
		req, err := http.NewRequestWithContext(context.Background(), method, endpoint, body)
		assert.NilError(t, err)
		return req
	}

	for name, test := range map[string]struct {
		request       func() *http.Request
		setup         func(*RequestMatcherBuilder)
		errorContains []string
	}{
		"Method ok": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "/", nil) },
			setup:         func(builder *RequestMatcherBuilder) { builder.Method(http.MethodGet) },
			errorContains: nil,
		},
		"Method ko": {
			request:       func() *http.Request { return newRequest(http.MethodPost, "/", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.Method(http.MethodGet) },
			errorContains: []string{`request method "POST" != "GET"`},
		},
		"URLHost ok": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "http://example.com/", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.URLHost("example.com") },
			errorContains: nil,
		},
		"URLHost ko": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "http://notexample.com/", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.URLHost("example.com") },
			errorContains: []string{`request url host "notexample.com" != "example.com"`},
		},
		"URLPath ok": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "/foo", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.URLPath("/foo") },
			errorContains: nil,
		},
		"URLPath ko": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "/notfoo", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.URLPath("/foo") },
			errorContains: []string{`request url path "/notfoo" != "/foo"`},
		},
		"URLQueryParamsContains ok": {
			request:       func() *http.Request { return newRequest(http.MethodGet, "/?a=1&a=2&b=b", nil) },
			setup:         func(b *RequestMatcherBuilder) { b.URLQueryParamsContains(url.Values{"a": {"1", "2"}, "b": {"b"}}) },
			errorContains: nil,
		},
		"URLQueryParamsContains ko": {
			request: func() *http.Request { return newRequest(http.MethodGet, "/?a=2&a=1&notb=b", nil) },
			setup:   func(b *RequestMatcherBuilder) { b.URLQueryParamsContains(url.Values{"a": {"1", "2"}, "b": {"b"}}) },
			errorContains: []string{
				`expected url query param key a to be [1 2] but is [2 1]`,
				`expected url query param key b to be set`,
			},
		},
		"HeadersContains ok": {
			request: func() *http.Request {
				req := newRequest(http.MethodGet, "/", nil)
				req.Header = http.Header{"a": {"1", "2"}, "b": {"b"}}
				return req
			},
			setup:         func(b *RequestMatcherBuilder) { b.HeadersContains(http.Header{"a": {"1", "2"}, "b": {"b"}}) },
			errorContains: nil,
		},
		"HeadersContains ko": {
			request: func() *http.Request {
				req := newRequest(http.MethodGet, "/", nil)
				req.Header = http.Header{"a": {"2", "1"}, "notb": {"b"}}
				return req
			},
			setup: func(b *RequestMatcherBuilder) { b.HeadersContains(http.Header{"a": {"1", "2"}, "b": {"b"}}) },
			errorContains: []string{
				`expected header key a to be [1 2] but is [2 1]`,
				`expected header key b to be set`,
			},
		},
		"BodyForm not strict ok": {
			request: func() *http.Request {
				req := newRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"a": {"1", "2"}, "b": {"b"}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			setup:         func(b *RequestMatcherBuilder) { b.BodyForm(url.Values{"a": {"1", "2"}, "b": {"b"}}, false) },
			errorContains: nil,
		},
		"BodyForm not strict ko": {
			request: func() *http.Request {
				req := newRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"a": {"2", "1"}, "notb": {"b"}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			setup: func(b *RequestMatcherBuilder) { b.BodyForm(url.Values{"a": {"1", "2"}, "b": {"b"}}, false) },
			errorContains: []string{
				"key a values differ [1 2] [2 1]",
				"key b is expected to exist but is not found",
			},
		},
		"BodyForm strict ok": {
			request: func() *http.Request {
				req := newRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"a": {"1", "2"}, "b": {"b"}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			setup:         func(b *RequestMatcherBuilder) { b.BodyForm(url.Values{"a": {"1", "2"}, "b": {"b"}}, true) },
			errorContains: nil,
		},
		"BodyForm strict ko": {
			request: func() *http.Request {
				req := newRequest(http.MethodDelete, "/", strings.NewReader(url.Values{"a": {"2", "1"}, "notb": {"b"}}.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			setup: func(b *RequestMatcherBuilder) { b.BodyForm(url.Values{"a": {"1", "2"}, "b": {"b"}}, true) },
			errorContains: []string{
				"key a values differ [1 2] [2 1]",
				"key b is expected to exist but is not found",
				`remaining key found in form: "notb": {["b"]}`,
			},
		},
		"BodyJSON not strict ok": {
			request: func() *http.Request {
				req := newRequest(http.MethodPut, "/", jsonEncode(t, map[string]any{"hello": "world", "number": 42}))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setup: func(b *RequestMatcherBuilder) {
				b.BodyJSON(
					&struct {
						Hello  string `json:"hello"`
						Number uint64 `json:"number"`
					}{
						Hello:  "world",
						Number: 42,
					},
					func() any {
						return &struct {
							Hello  string `json:"hello"`
							Number uint64 `json:"number"`
						}{}
					},
					false,
				)
			},
			errorContains: nil,
		},
		"BodyJSON not strict ko": {
			request: func() *http.Request {
				req := newRequest(http.MethodPut, "/", jsonEncode(t, map[string]any{"hello": "notworld", "notnumber": 42}))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setup: func(b *RequestMatcherBuilder) {
				b.BodyJSON(
					&struct {
						Hello  string `json:"hello"`
						Number uint64 `json:"number"`
					}{
						Hello:  "world",
						Number: 42,
					},
					func() any {
						return &struct {
							Hello  string `json:"hello"`
							Number uint64 `json:"number"`
						}{}
					},
					false,
				)
			},
			errorContains: []string{"json does not match"},
		},
		"BodyJSON strict ok": {
			request: func() *http.Request {
				req := newRequest(http.MethodPut, "/", jsonEncode(t, map[string]any{"hello": "world", "number": 42}))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setup: func(b *RequestMatcherBuilder) {
				b.BodyJSON(
					&struct {
						Hello  string `json:"hello"`
						Number uint64 `json:"number"`
					}{
						Hello:  "world",
						Number: 42,
					},
					func() any {
						return &struct {
							Hello  string `json:"hello"`
							Number uint64 `json:"number"`
						}{}
					},
					true,
				)
			},
			errorContains: nil,
		},
		"BodyJSON strict ko": {
			request: func() *http.Request {
				req := newRequest(http.MethodPut, "/", jsonEncode(t, map[string]any{"hello": "notworld", "notnumber": 42}))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setup: func(b *RequestMatcherBuilder) {
				b.BodyJSON(
					&struct {
						Hello  string `json:"hello"`
						Number uint64 `json:"number"`
					}{
						Hello:  "world",
						Number: 42,
					},
					func() any {
						return &struct {
							Hello  string `json:"hello"`
							Number uint64 `json:"number"`
						}{}
					},
					true,
				)
			},
			errorContains: []string{"json: unknown field"},
		},
	} {
		name, test := name, test

		t.Run(name, func(t *testing.T) {
			builder := NewRequestMatcherBuilder()
			test.setup(builder)

			err := builder.MatchRequest(test.request())

			for _, contains := range test.errorContains {
				assert.Check(t, cmp.ErrorContains(err, contains))
			}
			if test.errorContains == nil {
				assert.Check(t, err)
			}
		})
	}
}
