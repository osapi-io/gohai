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

// Package cloudmetadata is the shared HTTP client every cloud-provider
// collector (ec2, gce, azure, digital_ocean, openstack, oci, linode,
// alibaba, scaleway) uses to talk to its provider's link-local
// metadata endpoint. Handles short-timeout GETs, per-provider
// authentication headers (Azure's `Metadata: true`, OCI's
// `Authorization: Bearer Oracle`), and a typed ErrNotAvailable signal
// so collectors can short-circuit to a nil result when the host isn't
// on that provider.
//
// Tests don't stub this package — they point a Client at an
// httptest.Server URL and exercise the real HTTP round-trip against
// canned responses.
package cloudmetadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the per-request timeout applied when the caller
// doesn't override via WithTimeout. Cloud metadata endpoints are
// link-local so requests either succeed almost instantly or we're
// not on that provider; two seconds is generous for the happy path
// without being user-visible when we're not.
const defaultTimeout = 2 * time.Second

// ErrNotAvailable signals the provider's metadata endpoint couldn't
// be reached (connection refused, timeout, DNS failure, HTTP non-2xx).
// Collectors unwrap this via errors.Is and return (nil, nil) so the
// typed field drops from Facts without surfacing a misleading error.
var ErrNotAvailable = errors.New("cloud metadata endpoint not available")

// Client is a thin HTTP wrapper scoped to one provider's metadata
// base URL. Construct via New; swap the base URL for tests by
// pointing it at an httptest.Server.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// Option configures a Client at construction time.
type Option func(*Client)

// DefaultUserAgent is the User-Agent header applied to every request
// unless the caller overrides it via WithHeader. Matches Ohai's
// chef-ohai/<version> pattern — metadata services occasionally rate-
// limit or filter by UA, and a stable identifier makes gohai traffic
// distinguishable in provider logs.
const DefaultUserAgent = "gohai"

// New returns a Client rooted at baseURL. Default timeout is 2s;
// override with WithTimeout. Default transport has proxy lookup
// disabled (metadata IPs are link-local — honoring HTTP_PROXY would
// route cloud probes through a proxy and almost certainly fail).
// The User-Agent header defaults to DefaultUserAgent; override with
// WithHeader("User-Agent", "...") if needed.
func New(
	baseURL string,
	opts ...Option,
) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: map[string]string{"User-Agent": DefaultUserAgent},
		httpClient: &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				// Proxy lookup deliberately disabled — metadata IPs are
				// link-local and must not route through HTTP_PROXY.
				Proxy: nil,
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithHeader adds a header sent with every request. Providers use
// this for auth / content negotiation (Azure's "Metadata: true",
// OCI's "Authorization: Bearer Oracle"). Multiple calls accumulate.
func WithHeader(
	k, v string,
) Option {
	return func(c *Client) { c.headers[k] = v }
}

// WithTimeout overrides the default 2s per-request timeout.
func WithTimeout(
	d time.Duration,
) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHTTPClient swaps the underlying http.Client wholesale. Tests
// rarely need this — pointing baseURL at an httptest.Server exercises
// the full HTTP round-trip via the default client. Reserved for
// advanced cases (custom Transport for TLS-on-metadata, retry
// wrappers, etc.).
func WithHTTPClient(
	h *http.Client,
) Option {
	return func(c *Client) { c.httpClient = h }
}

// Get fetches baseURL + path and returns the response body on 2xx.
// Wraps transport failures and non-2xx responses in ErrNotAvailable
// so callers can branch with errors.Is.
func (c *Client) Get(
	ctx context.Context,
	path string,
) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// GetWithHeaders is Get with per-request headers merged onto the
// Client's default headers. Used by EC2's IMDSv2 flow where the
// token header must accompany every read but isn't known until
// Collect runs.
func (c *Client) GetWithHeaders(
	ctx context.Context,
	path string,
	headers map[string]string,
) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, headers)
}

// RawGet issues a GET and returns the response body + status code,
// without treating non-2xx as ErrNotAvailable. Used by Azure's
// api-version negotiation which relies on reading the HTTP 400 body
// Azure emits when no api-version is supplied. Transport failures
// still wrap ErrNotAvailable.
func (c *Client) RawGet(
	ctx context.Context,
	path string,
) ([]byte, int, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// Put issues a PUT against baseURL + path with per-request headers
// merged onto the Client's default headers. Body may be nil. Used by
// EC2's IMDSv2 token flow — that's currently the only provider that
// writes rather than reads the metadata endpoint.
func (c *Client) Put(
	ctx context.Context,
	path string,
	headers map[string]string,
) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, headers)
}

// do is the shared request driver. Prefixes the path with / when
// missing, applies the Client's default headers plus any per-request
// headers, runs the request, and returns the body for 2xx responses.
// Transport failures and non-2xx responses wrap ErrNotAvailable.
func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	extraHeaders map[string]string,
) ([]byte, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"%w: unexpected status %d from %s",
			ErrNotAvailable,
			resp.StatusCode,
			path,
		)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

// GetJSON fetches path and JSON-decodes into out. Transport-level
// failures propagate as ErrNotAvailable (same as Get); JSON decode
// errors propagate as-is.
func (c *Client) GetJSON(
	ctx context.Context,
	path string,
	out any,
) error {
	body, err := c.Get(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode json from %s: %w", path, err)
	}
	return nil
}
