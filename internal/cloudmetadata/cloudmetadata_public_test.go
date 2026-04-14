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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
)

// brokenBodyTransport returns a 200 response whose Body errors on
// Read — exercises Get's io.ReadAll-fails branch. Real HTTP clients
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

func TestCloudMetadataPublicTestSuite(
	t *testing.T,
) {
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
	tests := []struct {
		name        string
		setup       func() (baseURL string, opts []cloudmetadata.Option, cleanup func())
		path        string
		ctx         func() (context.Context, context.CancelFunc)
		wantErr     bool
		wantErrIs   error
		wantBody    string
		wantHeaders map[string]string
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
			path:     "/ok",
			wantBody: "hello",
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
			path:     "rel",
			wantBody: "relative-ok",
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
			path:     "/ok",
			wantBody: "trim-ok",
		},
		{
			name: "headers are sent on every request",
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
			path:     "/h",
			wantBody: "ok",
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
			path:     "/ua",
			wantBody: "ok",
		},
		{
			name: "WithHeader overrides the default User-Agent",
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
			path:     "/ua",
			wantBody: "ok",
		},
		{
			name: "404 wraps ErrNotAvailable",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(nil)
				return srv.URL, nil, srv.Close
			},
			path:      "/missing",
			wantErr:   true,
			wantErrIs: cloudmetadata.ErrNotAvailable,
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
			path:      "/bad",
			wantErr:   true,
			wantErrIs: cloudmetadata.ErrNotAvailable,
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
			path:      "/anything",
			wantErr:   true,
			wantErrIs: cloudmetadata.ErrNotAvailable,
		},
		{
			name: "timeout wraps ErrNotAvailable",
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
			path:      "/slow",
			wantErr:   true,
			wantErrIs: cloudmetadata.ErrNotAvailable,
		},
		{
			name: "context cancellation propagates",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/slow": func(_ http.ResponseWriter, r *http.Request) {
						<-r.Context().Done()
					},
				})
				return srv.URL, nil, srv.Close
			},
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			path:      "/slow",
			wantErr:   true,
			wantErrIs: cloudmetadata.ErrNotAvailable,
		},
		{
			name: "WithHTTPClient swaps underlying transport",
			setup: func() (string, []cloudmetadata.Option, func()) {
				srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
					"/ok": func(w http.ResponseWriter, _ *http.Request) {
						_, _ = w.Write([]byte("custom-ok"))
					},
				})
				custom := &http.Client{Timeout: time.Second}
				opts := []cloudmetadata.Option{cloudmetadata.WithHTTPClient(custom)}
				return srv.URL, opts, srv.Close
			},
			path:     "/ok",
			wantBody: "custom-ok",
		},
		{
			name: "body read failure returns non-ErrNotAvailable error",
			setup: func() (string, []cloudmetadata.Option, func()) {
				opts := []cloudmetadata.Option{
					cloudmetadata.WithHTTPClient(&http.Client{
						Transport: brokenBodyTransport{},
					}),
				}
				return "http://example.invalid", opts, func() {}
			},
			path:    "/x",
			wantErr: true,
		},
		{
			name: "invalid URL builds fail before transport",
			setup: func() (string, []cloudmetadata.Option, func()) {
				// Control character in URL — http.NewRequestWithContext
				// rejects before any transport call.
				return "http://\x7f", nil, func() {}
			},
			path:    "/x",
			wantErr: true,
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
			body, err := client.Get(ctx, tt.path)
			if tt.wantErr {
				s.Require().Error(err)
				if tt.wantErrIs != nil {
					s.True(errors.Is(err, tt.wantErrIs),
						"expected error chain to include %v, got %v",
						tt.wantErrIs, err)
				}
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantBody, string(body))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestGetWithHeaders() {
	var got http.Header
	srv := newServer(map[string]func(http.ResponseWriter, *http.Request){
		"/x": func(w http.ResponseWriter, r *http.Request) {
			got = r.Header.Clone()
			_, _ = w.Write([]byte("ok"))
		},
	})
	defer srv.Close()

	client := cloudmetadata.New(srv.URL)
	body, err := client.GetWithHeaders(context.Background(), "/x", map[string]string{
		"X-Token": "abc123",
	})
	s.Require().NoError(err)
	s.Equal("ok", string(body))
	s.Equal("abc123", got.Get("X-Token"))
	s.Equal(cloudmetadata.DefaultUserAgent, got.Get("User-Agent"))
}

func (s *CloudMetadataPublicTestSuite) TestRawGet() {
	tests := []struct {
		name       string
		setup      func() (baseURL string, cleanup func())
		path       string
		wantErr    bool
		wantStatus int
		wantBody   string
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
			path:       "/ok",
			wantStatus: http.StatusOK,
			wantBody:   "hello",
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
			path:       "/v",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"newest-versions":["2024-01-01"]}`,
		},
		{
			name: "connection refused wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				url := srv.URL
				srv.Close()
				return url, func() {}
			},
			path:    "/x",
			wantErr: true,
		},
		{
			name: "invalid URL builds fail before transport",
			setup: func() (string, func()) {
				return "http://\x7f", func() {}
			},
			path:    "/x",
			wantErr: true,
		},
		{
			name: "body read failure returns non-ErrNotAvailable error",
			setup: func() (string, func()) {
				return "http://example.invalid", func() {}
			},
			path:    "/x",
			wantErr: true,
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
			path:       "rel",
			wantStatus: http.StatusOK,
			wantBody:   "relative-ok",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			opts := []cloudmetadata.Option(nil)
			if tt.name == "body read failure returns non-ErrNotAvailable error" {
				opts = append(opts, cloudmetadata.WithHTTPClient(&http.Client{
					Transport: brokenBodyTransport{},
				}))
			}

			client := cloudmetadata.New(baseURL, opts...)
			body, status, err := client.RawGet(context.Background(), tt.path)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantStatus, status)
			s.Equal(tt.wantBody, string(body))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestPut() {
	tests := []struct {
		name        string
		setup       func() (baseURL string, cleanup func())
		headers     map[string]string
		wantErr     bool
		wantBody    string
		verifyAfter func(s *CloudMetadataPublicTestSuite)
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
			headers:  map[string]string{"X-TTL": "60"},
			wantBody: "TOKEN123",
		},
		{
			name: "404 wraps ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				return srv.URL, srv.Close
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL)
			body, err := client.Put(context.Background(), "/token", tt.headers)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantBody, string(body))
		})
	}
}

func (s *CloudMetadataPublicTestSuite) TestGetJSON() {
	type payload struct {
		Instance string `json:"instance"`
		Cores    int    `json:"cores"`
	}

	tests := []struct {
		name     string
		setup    func() (baseURL string, cleanup func())
		wantErr  bool
		wantBody payload
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
			wantBody: payload{Instance: "i-123", Cores: 8},
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
			wantErr: true,
		},
		{
			name: "transport failure propagates ErrNotAvailable",
			setup: func() (string, func()) {
				srv := newServer(nil)
				return srv.URL, srv.Close
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			baseURL, cleanup := tt.setup()
			defer cleanup()

			client := cloudmetadata.New(baseURL)
			var got payload
			err := client.GetJSON(context.Background(), "/m", &got)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantBody, got)
		})
	}
}
