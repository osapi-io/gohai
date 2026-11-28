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
// Tests don't stub this package. They point a Client at an
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

// MaxBody is the most that one metadata response may hold. A metadata
// document, such as Azure's compute instance, is some kilobytes. A longer
// body is from an endpoint that is not the provider's, or a broken one, and
// fails.
const MaxBody = 1 << 20

// transport is the one transport of each Client, so that the clients of the
// cloud collectors share their connections. Proxy lookup is off: metadata
// IPs are link-local and must not route through HTTP_PROXY. An idle
// connection closes after idleConnTimeout, so a long-running caller, such
// as osapi's agent, does not keep one open between collections.
var transport = &http.Transport{
	Proxy:           nil,
	IdleConnTimeout: idleConnTimeout,
}

// idleConnTimeout is how long an idle metadata connection stays open.
const idleConnTimeout = 30 * time.Second

// ErrNotAvailable signals the provider's metadata endpoint couldn't
// be reached (connection refused, timeout, DNS failure, HTTP non-2xx).
// Collectors unwrap this via errors.Is and return (nil, nil) so the
// typed field drops from Facts without surfacing a misleading error.
var ErrNotAvailable = errors.New("cloud metadata endpoint not available")

// transportError is the error of a request that got no response. When ctx
// ended, it is ctx's error, not ErrNotAvailable, so a cancelled collection
// is an error and is not taken for a host with no metadata service.
func transportError(
	ctx context.Context,
	err error,
) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("metadata request: %w", ctxErr)
	}

	return fmt.Errorf("%w: %w", ErrNotAvailable, err)
}

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
// chef-ohai/<version> pattern, because metadata services sometimes rate-
// limit or filter by UA, and a stable identifier makes gohai traffic
// distinguishable in provider logs.
const DefaultUserAgent = "gohai"

// New returns a Client rooted at baseURL. Default timeout is 2s;
// override with WithTimeout. Each Client shares one transport, with proxy
// lookup disabled (metadata IPs are link-local, so honoring HTTP_PROXY would
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
			Timeout:   defaultTimeout,
			Transport: transport,
			// A metadata service does not redirect. A redirect would get
			// the request again, with its token headers, such as EC2's
			// X-aws-ec2-metadata-token, at the host that the redirect
			// names: Go removes only Authorization and Cookie. The 3xx is
			// the answer instead, which is not a 2xx.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
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
	k string,
	v string,
) Option {
	return func(c *Client) { c.headers[k] = v }
}

// WithTransport makes rt the transport of the Client's requests. A nil rt
// keeps the shared transport. A caller's source.Sources.HTTP comes here,
// so a fake host answers the metadata reads.
func WithTransport(
	rt http.RoundTripper,
) Option {
	return func(c *Client) {
		if rt != nil {
			c.httpClient.Transport = rt
		}
	}
}

// WithTimeout overrides the default 2s per-request timeout.
func WithTimeout(
	d time.Duration,
) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
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
		return nil, 0, transportError(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readBody(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// Lookup fetches baseURL + path and returns the body on 2xx. Only a
// request with no answer, or a 404, wraps ErrNotAvailable: the document
// is not there. Any other non-2xx, such as a 500, is an error of its own,
// so a metadata service that answers with a failure is not taken for a
// host with no such document. A cancelled context is its own error.
func (c *Client) Lookup(
	ctx context.Context,
	path string,
) ([]byte, error) {
	body, status, err := c.RawGet(ctx, path)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("%w: status 404 from %s", ErrNotAvailable, path)
	}
	if status < 200 || status >= 300 {
		return nil, &StatusError{Status: status, Path: path}
	}

	return body, nil
}

// StatusError is the error of a Lookup whose answer was a non-2xx status
// other than 404. A collector that probes a host which DMI did not
// confirm can tell a 4xx (another service, not this cloud) from a 5xx (a
// metadata service that failed) by Status.
type StatusError struct {
	// Status is the HTTP status code of the answer.
	Status int
	// Path is the path that was asked for.
	Path string
}

// Error returns "unexpected status <status> from <path>".
func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d from %s", e.Status, e.Path)
}

// ClientError reports whether err is a StatusError with a 4xx status.
func ClientError(
	err error,
) bool {
	var se *StatusError

	return errors.As(err, &se) && se.Status >= 400 && se.Status < 500
}

// Put issues a PUT against baseURL + path with per-request headers
// merged onto the Client's default headers. It sends no body. Used by
// EC2's IMDSv2 token flow. EC2 is the only provider that
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
		return nil, transportError(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read the body, up to MaxBody, so that the shared transport can
		// reuse the connection.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxBody))

		return nil, fmt.Errorf(
			"%w: unexpected status %d from %s",
			ErrNotAvailable,
			resp.StatusCode,
			path,
		)
	}
	body, err := readBody(resp.Body)
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

// readBody reads at most MaxBody of body, and fails when body has more.
func readBody(
	body io.Reader,
) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(body, MaxBody+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxBody {
		return nil, fmt.Errorf("body is more than %d bytes", MaxBody)
	}

	return b, nil
}
