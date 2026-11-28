// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cloudmetadata_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
)

// brokenBodyTransport returns a 200 response whose Body errors on
// Read, which exercises Get's io.ReadAll-fails branch. Real HTTP clients
// hit this on truncated responses / server resets mid-body.
type brokenBodyTransport struct{}

func (brokenBodyTransport) RoundTrip(
	*http.Request,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&erroringReader{}),
		Header:     http.Header{},
	}, nil
}

type erroringReader struct{}

func (*erroringReader) Read(
	[]byte,
) (int, error) {
	return 0, errors.New("simulated read failure")
}

type CloudMetadataPublicTestSuite struct {
	suite.Suite
}

func TestCloudMetadataPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CloudMetadataPublicTestSuite))
}

// newServer builds an httptest.Server whose handler is constructed
// from the given routes. Missing paths return 404 so tests exercise
// the non-2xx branch.
func newServer(
	routes map[string]func(http.ResponseWriter, *http.Request),
) *httptest.Server {
	mux := http.NewServeMux()
	for path, handler := range routes {
		mux.HandleFunc(path, handler)
	}
	return httptest.NewServer(mux)
}

func (s *CloudMetadataPublicTestSuite) TestGet() {
	// hits and gotToken record what reaches the host that a redirect names.
	var hits int
	var gotToken string
	// drainURL and drainConns are the server of the non-2xx drain row, and
	// the connections it accepted.
	var drainURL string
	var drainConns atomic.Int32

	tests := []struct {
		name         string
		setup        func() (baseURL string, opts []cloudmetadata.Option, cleanup func())
		path         string
		ctx          func() (context.Context, context.CancelFunc)
		validateFunc func([]byte, error)
	}{
		{
			name: "200 returns body",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ok": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("hello"))
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "/ok",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("hello", string(body))
			},
		},
		{
			name: "path without leading slash is normalized",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/rel": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("relative-ok"))
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "rel",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("relative-ok", string(body))
			},
		},
		{
			name: "baseURL trailing slash is stripped",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ok": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("trim-ok"))
					},
				})
				return srv.URL + "/", nil, srv.Close
			},
			path: "/ok",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("trim-ok", string(body))
			},
		},
		{
			name: "default User-Agent is sent when caller doesn't override",
			setup: func() (string, []cloudmetadata.Option, func()) {
				var got http.Header
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ua": func(w http.ResponseWriter, r *http.Request) {
						got = r.Header.Clone()
						_, _ = w.Write([]byte("ok"))
					},
				})
				return srv.URL, nil, func() {
					srv.Close()
					s.Equal(cloudmetadata.DefaultUserAgent, got.Get("User-Agent"))
				}
			},
			path: "/ua",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ok", string(body))
			},
		},
		{
			name: "404 wraps ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(nil)
				return srv.URL, nil, srv.Close
			},
			path: "/missing",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
		{
			name: "a non-2xx body is read, so the next request reuses the connection",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := httptest.NewUnstartedServer(http.HandlerFunc(
					func(w http.ResponseWriter, _ *http.Request) {
						http.Error(w, strings.Repeat("x", 8<<10), http.StatusNotFound)
					},
				))
				srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
					if state == http.StateNew {
						drainConns.Add(1)
					}
				}
				srv.Start()
				drainURL = srv.URL

				return srv.URL, nil, srv.Close
			},
			path: "/missing",
			validateFunc: func(_ []byte, err error) {
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
				_, err = cloudmetadata.New(drainURL).Get(context.Background(), "/missing")
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
				_, err = cloudmetadata.New(drainURL).Get(context.Background(), "/missing")
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.Equal(int32(1), drainConns.Load())
			},
		},
		{
			name: "500 wraps ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/bad": func(w http.ResponseWriter, _ *http.Request) {
						http.Error(w, "boom", http.StatusInternalServerError)
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "/bad",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
		{
			name: "connection refused wraps ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				// Start then immediately close the server so the port
				// is no longer listening when Get runs.
				srv := newServer(nil)
				url := srv.URL
				srv.Close()
				return url, nil, func() {}
			},
			path: "/anything",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
		{
			name: "a cancelled context is its own error, not ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/slow": func(_ http.ResponseWriter, r *http.Request) {
						<-r.Context().Done()
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "/slow",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			validateFunc: func(_ []byte, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.NotErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
		{
			name: "a body of MaxBody bytes is read",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/big": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write(make([]byte, cloudmetadata.MaxBody))
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "/big",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Len(body, cloudmetadata.MaxBody)
			},
		},
		{
			name: "a body of more than MaxBody bytes fails",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/big": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write(make([]byte, cloudmetadata.MaxBody+1))
					},
				})
				return srv.URL, nil, srv.Close
			},
			path: "/big",
			validateFunc: func(_ []byte, err error) {
				s.ErrorContains(err, "body is more than 1048576 bytes")
				s.NotErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
		{
			name: "body read failure returns non-ErrNotAvailable error",
			setup: func() (string, []cloudmetadata.Option, func()) {
				opts := []cloudmetadata.Option{
					cloudmetadata.WithTransport(brokenBodyTransport{}),
				}
				return "http://example.invalid", opts, func() {}
			},
			path: "/x",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "invalid URL builds fail before transport",
			setup: func() (string, []cloudmetadata.Option, func()) {
				// A control character in the URL: http.NewRequestWithContext
				// rejects before any transport call.
				return "http://\x7f", nil, func() {}
			},
			path: "/x",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "a redirect is not followed, so the token does not leave the metadata host",
			setup: func() (string, []cloudmetadata.Option, func()) {
				other := httptest.NewServer(
					http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
						hits++
						gotToken = r.Header.Get("X-aws-ec2-metadata-token")
					}),
				)
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/latest/meta-data/instance-id": func(w http.ResponseWriter, r *http.Request) {
						http.Redirect(w, r, other.URL+"/steal", http.StatusFound)
					},
				})
				opts := []cloudmetadata.Option{
					cloudmetadata.WithHeader("X-aws-ec2-metadata-token", "secret-token"),
				}

				return srv.URL, opts, func() {
					srv.Close()
					other.Close()
				}
			},
			path: "/latest/meta-data/instance-id",
			validateFunc: func(_ []byte, err error) {
				s.ErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.Zero(hits)
				s.Empty(gotToken)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, opts, cleanup := tt.setup()
			defer cleanup()

			ctx := context.Background()
			if tt.ctx != nil {
				c, cancel := tt.ctx()
				defer cancel()
				ctx = c
			}

			client := cloudmetadata.New(baseURL, opts...)
			tt.validateFunc(client.Get(ctx, tt.path))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		validateFunc func(a, b http.RoundTripper)
	}{
		{
			name: "the clients of the cloud collectors share their connections",
			validateFunc: func(a, b http.RoundTripper) {
				s.Same(a, b)
			},
		},
		{
			name: "the shared transport uses no proxy, because metadata IPs are link-local",
			validateFunc: func(a, _ http.RoundTripper) {
				t, ok := a.(*http.Transport)
				s.Require().True(ok)
				s.Nil(t.Proxy)
			},
		},
		{
			name: "an idle connection of the shared transport closes",
			validateFunc: func(a, _ http.RoundTripper) {
				t, ok := a.(*http.Transport)
				s.Require().True(ok)
				s.Positive(t.IdleConnTimeout)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(
				cloudmetadata.TransportOf(cloudmetadata.New("http://a.invalid")),
				cloudmetadata.TransportOf(cloudmetadata.New("http://b.invalid")),
			)
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestWithHeader() {
	tests := []struct {
		name         string
		setup        func() (baseURL string, opts []cloudmetadata.Option, cleanup func())
		path         string
		validateFunc func([]byte, error)
	}{
		{
			name: "each header is sent on every request, and the calls add up",
			setup: func() (string, []cloudmetadata.Option, func()) {
				var got http.Header
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/h": func(w http.ResponseWriter, r *http.Request) {
						got = r.Header.Clone()
						_, _ = w.Write([]byte("ok"))
					},
				})
				opts := []cloudmetadata.Option{
					cloudmetadata.WithHeader("Metadata", "true"),
					cloudmetadata.WithHeader("Authorization", "Bearer Oracle"),
				}
				return srv.URL, opts, func() {
					srv.Close()
					s.Equal("true", got.Get("Metadata"))
					s.Equal("Bearer Oracle", got.Get("Authorization"))
				}
			},
			path: "/h",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ok", string(body))
			},
		},
		{
			name: "a header overrides the default User-Agent",
			setup: func() (string, []cloudmetadata.Option, func()) {
				var got http.Header
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ua": func(w http.ResponseWriter, r *http.Request) {
						got = r.Header.Clone()
						_, _ = w.Write([]byte("ok"))
					},
				})
				opts := []cloudmetadata.Option{
					cloudmetadata.WithHeader("User-Agent", "custom/1.0"),
				}
				return srv.URL, opts, func() {
					srv.Close()
					s.Equal("custom/1.0", got.Get("User-Agent"))
				}
			},
			path: "/ua",
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ok", string(body))
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, opts, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL, opts...)
			tt.validateFunc(client.Get(context.Background(), tt.path))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestWithTimeout() {
	tests := []struct {
		name         string
		setup        func() (baseURL string, opts []cloudmetadata.Option, cleanup func())
		path         string
		validateFunc func([]byte, error)
	}{
		{
			name: "a request that passes the timeout wraps ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/slow": func(_ http.ResponseWriter, r *http.Request) {
						<-r.Context().Done()
					},
				})
				opts := []cloudmetadata.Option{
					cloudmetadata.WithTimeout(50 * time.Millisecond),
				}
				return srv.URL, opts, srv.Close
			},
			path: "/slow",
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, opts, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL, opts...)
			tt.validateFunc(client.Get(context.Background(), tt.path))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestWithTransport() {
	given := &http.Transport{}
	shared := cloudmetadata.TransportOf(cloudmetadata.New("http://a.invalid"))
	tests := []struct {
		name string
		rt   http.RoundTripper
		want http.RoundTripper
	}{
		{name: "a given transport carries the requests", rt: given, want: given},
		{name: "a nil transport keeps the shared one", rt: nil, want: shared},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := cloudmetadata.New("http://a.invalid", cloudmetadata.WithTransport(tt.rt))
			s.Same(tt.want, cloudmetadata.TransportOf(c))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestGetWithHeaders() {
	tests := []struct {
		name         string
		headers      map[string]string
		validateFunc func(body []byte, got http.Header, err error)
	}{
		{
			name:    "sends the headers of the call with the default User-Agent",
			headers: map[string]string{"X-Token": "abc123"},
			validateFunc: func(body []byte, got http.Header, err error) {
				s.Require().NoError(err)
				s.Equal("ok", string(body))
				s.Equal("abc123", got.Get("X-Token"))
				s.Equal(cloudmetadata.DefaultUserAgent, got.Get("User-Agent"))
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var got http.Header
			srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
				"/x": func(w http.ResponseWriter, r *http.Request) {
					got = r.Header.Clone()
					_, _ = w.Write([]byte("ok"))
				},
			})
			defer srv.Close()

			body, err := cloudmetadata.New(srv.URL).
				GetWithHeaders(context.Background(), "/x", tt.headers)
			tt.validateFunc(body, got, err)
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestRawGet() {
	// hits counts the requests that reach the host that a redirect names.
	var hits int

	tests := []struct {
		name         string
		setup        func() (baseURL string, cleanup func())
		path         string
		validateFunc func([]byte, int, error)
	}{
		{
			name: "200 returns body + status",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ok": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("hello"))
					},
				})
				return srv.URL, srv.Close
			},
			path: "/ok",
			validateFunc: func(body []byte, status int, err error) {
				s.Require().NoError(err)
				s.Equal(http.StatusOK, status)
				s.Equal("hello", string(body))
			},
		},
		{
			name: "400 returns body + status without wrapping ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/v": func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"newest-versions":["2024-01-01"]}`))
					},
				})
				return srv.URL, srv.Close
			},
			path: "/v",
			validateFunc: func(body []byte, status int, err error) {
				s.Require().NoError(err)
				s.Equal(http.StatusBadRequest, status)
				s.Equal(`{"newest-versions":["2024-01-01"]}`, string(body))
			},
		},
		{
			name: "connection refused wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				url := srv.URL
				srv.Close()
				return url, func() {}
			},
			path: "/x",
			validateFunc: func(_ []byte, _ int, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "invalid URL builds fail before transport",
			setup: func() (string, func()) {
				return "http://\x7f", func() {}
			},
			path: "/x",
			validateFunc: func(_ []byte, _ int, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "body read failure returns non-ErrNotAvailable error",
			setup: func() (string, func()) {
				return "http://example.invalid", func() {}
			},
			path: "/x",
			validateFunc: func(_ []byte, _ int, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "path without leading slash is normalized",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/rel": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("relative-ok"))
					},
				})
				return srv.URL, srv.Close
			},
			path: "rel",
			validateFunc: func(body []byte, status int, err error) {
				s.Require().NoError(err)
				s.Equal(http.StatusOK, status)
				s.Equal("relative-ok", string(body))
			},
		},
		{
			name: "a redirect returns its status, and is not followed",
			setup: func() (string, func()) {
				other := httptest.NewServer(
					http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
						hits++
					}),
				)
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/latest/meta-data/instance-id": func(w http.ResponseWriter, r *http.Request) {
						http.Redirect(w, r, other.URL+"/steal", http.StatusFound)
					},
				})

				return srv.URL, func() {
					srv.Close()
					other.Close()
				}
			},
			path: "/latest/meta-data/instance-id",
			validateFunc: func(_ []byte, status int, err error) {
				s.Require().NoError(err)
				s.Equal(http.StatusFound, status)
				s.Zero(hits)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			opts := []cloudmetadata.Option(nil)
			if tt.name == "body read failure returns non-ErrNotAvailable error" {
				opts = append(opts, cloudmetadata.WithTransport(brokenBodyTransport{}))
			}

			client := cloudmetadata.New(baseURL, opts...)
			tt.validateFunc(client.RawGet(context.Background(), tt.path))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestLookup() {
	tests := []struct {
		name         string
		setup        func() (baseURL string, cleanup func())
		validateFunc func([]byte, error)
	}{
		{
			name: "200 returns the body",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/doc": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("hello"))
					},
				})
				return srv.URL, srv.Close
			},
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("hello", string(body))
			},
		},
		{
			name: "404 wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				return srv.URL, srv.Close
			},
			validateFunc: func(body []byte, err error) {
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.Nil(body)
			},
		},
		{
			name: "500 is an error that is not ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/doc": func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
					},
				})
				return srv.URL, srv.Close
			},
			validateFunc: func(body []byte, err error) {
				s.Require().EqualError(err, "unexpected status 500 from /doc")
				s.NotErrorIs(err, cloudmetadata.ErrNotAvailable)
				var se *cloudmetadata.StatusError
				s.Require().ErrorAs(err, &se)
				s.Equal(http.StatusInternalServerError, se.Status)
				s.Equal("/doc", se.Path)
				s.False(cloudmetadata.ClientError(err))
				s.Nil(body)
			},
		},
		{
			name: "403 is a client error that is not ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/doc": func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusForbidden)
					},
				})
				return srv.URL, srv.Close
			},
			validateFunc: func(body []byte, err error) {
				s.Require().EqualError(err, "unexpected status 403 from /doc")
				s.NotErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.True(cloudmetadata.ClientError(err))
				s.Nil(body)
			},
		},
		{
			name: "connection refused wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				url := srv.URL
				srv.Close()
				return url, func() {}
			},
			validateFunc: func(body []byte, err error) {
				s.Require().ErrorIs(err, cloudmetadata.ErrNotAvailable)
				s.False(cloudmetadata.ClientError(err))
				s.Nil(body)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL)
			tt.validateFunc(client.Lookup(context.Background(), "/doc"))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestPut() {
	tests := []struct {
		name         string
		setup        func() (baseURL string, cleanup func())
		headers      map[string]string
		validateFunc func([]byte, error)
	}{
		{
			name: "200 with extra headers forwards the headers",
			setup: func() (string, func()) {
				var gotHdr string
				var gotMethod string
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/token": func(w http.ResponseWriter, r *http.Request) {
						gotHdr = r.Header.Get("X-TTL")
						gotMethod = r.Method
						_, _ = w.Write([]byte("TOKEN123"))
					},
				})
				return srv.URL, func() {
					srv.Close()
					s.Equal("60", gotHdr)
					s.Equal(http.MethodPut, gotMethod)
				}
			},
			headers: map[string]string{"X-TTL": "60"},
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Equal("TOKEN123", string(body))
			},
		},
		{
			name: "404 wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				return srv.URL, srv.Close
			},
			validateFunc: func(_ []byte, err error) {
				s.Require().Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL)
			tt.validateFunc(client.Put(context.Background(), "/token", tt.headers))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestGetJSON() {
	type payload struct {
		Instance string `json:"instance"`
		Cores    int    `json:"cores"`
	}

	tests := []struct {
		name         string
		setup        func() (baseURL string, cleanup func())
		validateFunc func(payload, error)
	}{
		{
			name: "decodes 200 body into struct",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/m": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = fmt.Fprint(w, `{"instance":"i-123","cores":8}`)
					},
				})
				return srv.URL, srv.Close
			},
			validateFunc: func(got payload, err error) {
				s.Require().NoError(err)
				s.Equal(payload{Instance: "i-123", Cores: 8}, got)
			},
		},
		{
			name: "malformed JSON returns decode error (not ErrNotAvailable)",
			setup: func() (string, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/m": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("not json"))
					},
				})
				return srv.URL, srv.Close
			},
			validateFunc: func(_ payload, err error) {
				s.Require().Error(err)
			},
		},
		{
			name: "transport failure propagates ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				return srv.URL, srv.Close
			},
			validateFunc: func(_ payload, err error) {
				s.Require().Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL)
			var got payload

			// GetJSON fills got, so the call has to finish before the row
			// is handed the value
			err := client.GetJSON(context.Background(), "/m", &got)

			tt.validateFunc(got, err)
		})
	}
}
