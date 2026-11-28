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

// Package sourcetest builds the sources a collector test reads: a
// filesystem that holds given files, a runner that answers given
// command lines, and a transport that answers given metadata URLs.
package sourcetest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

// FS is an in-memory filesystem that holds files, by path.
func FS(
	t testing.TB,
	files map[string]string,
) avfs.VFS {
	t.Helper()
	fs := memfs.New()
	for path, content := range files {
		require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, fs.WriteFile(path, []byte(content), 0o644))
	}

	return fs
}

// Commands is the output of each command line of a fake host, "name arg
// arg", and the error of each line that fails. A test table holds it as
// data; Mock makes the runner.
type Commands struct {
	Out map[string]string
	Err map[string]error
}

// Mock is a generated source.Runner that answers each line of c: the
// output, or the error with the output. Any other line fails, as a
// command the host does not have.
func (c Commands) Mock(
	t testing.TB,
) *mocks.MockRunner {
	t.Helper()
	m := mocks.NewMockRunner(gomock.NewController(t))
	m.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, name string, args ...string) ([]byte, error) {
			line := strings.Join(append([]string{name}, args...), " ")
			if err, ok := c.Err[line]; ok {
				return []byte(c.Out[line]), err
			}
			if out, ok := c.Out[line]; ok {
				return []byte(out), nil
			}

			return nil, fmt.Errorf("exec %s: not found: %w", line, exec.ErrNotFound)
		}).
		AnyTimes()

	return m
}

// Links adds symbolic links to fs, each path to its target, as sysfs
// links a device to its driver, a link's directory before the link. It
// returns fs.
func Links(
	t testing.TB,
	fs avfs.VFS,
	links map[string]string,
) avfs.VFS {
	t.Helper()
	paths := make([]string, 0, len(links))
	for path := range links {
		paths = append(paths, path)
	}
	// A link inside a linked directory comes after the directory's link.
	slices.Sort(paths)
	for _, path := range paths {
		target := links[path]
		require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, fs.Symlink(target, path))
	}

	return fs
}

// FailRead makes a read of each of paths in fs fail with an error that is
// not fs.ErrNotExist, as a host gives for a file that it cannot read: it
// makes a directory at each path, and a read of a directory fails. It
// returns fs.
func FailRead(
	t testing.TB,
	fs avfs.VFS,
	paths ...string,
) avfs.VFS {
	t.Helper()
	for _, path := range paths {
		require.NoError(t, fs.MkdirAll(path, 0o755))
	}

	return fs
}

// ReadError wraps fs so that a ReadFile of each of paths returns err, and any
// other read goes to fs. It is for a file that the collector lists and then
// reads, where FailRead's directory would change what the collector lists.
func ReadError(
	fs avfs.VFS,
	err error,
	paths ...string,
) avfs.VFS {
	fail := make(map[string]bool, len(paths))
	for _, path := range paths {
		fail[path] = true
	}

	return readErrorFS{VFS: fs, err: err, fail: fail}
}

// readErrorFS is the filesystem that ReadError returns.
type readErrorFS struct {
	avfs.VFS
	err  error
	fail map[string]bool
}

// ReadFile returns the error for a path that ReadError named, and reads any
// other path from the wrapped filesystem.
func (r readErrorFS) ReadFile(
	name string,
) ([]byte, error) {
	if r.fail[name] {
		return nil, r.err
	}

	return r.VFS.ReadFile(name)
}

// HTTP is a transport that answers each URL of bodies, such as
// "http://169.254.169.254/latest/meta-data/instance-id", with 200 and
// its body. Any other URL answers 404, as a metadata service does for a
// path it does not have. A cloud collector reads it as
// source.Sources.HTTP, and reads no network.
func HTTP(
	bodies map[string]string,
) http.RoundTripper {
	return metadata(bodies)
}

// Server is a transport that sends each request to the server at base, such
// as an httptest.Server's URL, in place of the host the request names. A
// cloud collector reads it as source.Sources.HTTP, so a test serves the
// metadata service with its own handler.
func Server(
	base string,
) http.RoundTripper {
	return server{base: base}
}

// server is the transport of Server.
type server struct{ base string }

// RoundTrip sends a copy of req to the server at s.base.
func (s server) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	to, err := url.Parse(s.base)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	out := req.Clone(req.Context())
	out.URL.Scheme = to.Scheme
	out.URL.Host = to.Host
	out.Host = to.Host

	return http.DefaultTransport.RoundTrip(out)
}

// metadata is the transport of HTTP. http.RoundTripper is the standard
// library's interface, so it has no generated mock.
type metadata map[string]string

// RoundTrip answers req from the bodies.
func (m metadata) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	body, ok := m[req.URL.String()]
	status := http.StatusOK
	if !ok {
		status = http.StatusNotFound
	}

	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}
