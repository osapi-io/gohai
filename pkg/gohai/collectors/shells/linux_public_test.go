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

package shells_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

type ShellsLinuxPublicTestSuite struct {
	suite.Suite
}

func TestShellsLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsLinuxPublicTestSuite))
}

// stringReadCloser wraps a strings.Reader to satisfy io.ReadCloser.
type stringReadCloser struct{ *strings.Reader }

func (stringReadCloser) Close() error { return nil }

// erroringReader fails on every Read — used to exercise the scanner
// error path.
type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

type erroringReadCloser struct{ erroringReader }

func (erroringReadCloser) Close() error { return nil }

func (s *ShellsLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		openFn  func(string) (io.ReadCloser, error)
		wantErr bool
		want    []string
	}{
		{
			name: "canonical /etc/shells",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{
					strings.NewReader("# comment\n/bin/sh\n/bin/bash\n\n/usr/bin/zsh\n"),
				}, nil
			},
			want: []string{"/bin/sh", "/bin/bash", "/usr/bin/zsh"},
		},
		{
			name: "non-absolute entries skipped",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{
					strings.NewReader("/bin/sh\nnologin\nbash\n/bin/zsh\n"),
				}, nil
			},
			want: []string{"/bin/sh", "/bin/zsh"},
		},
		{
			name: "whitespace trimmed",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("  /bin/bash  \n\t/bin/sh\t\n")}, nil
			},
			want: []string{"/bin/bash", "/bin/sh"},
		},
		{
			name: "empty file",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("")}, nil
			},
			want: []string{},
		},
		{
			name: "missing file soft-misses",
			openFn: func(string) (io.ReadCloser, error) {
				return nil, os.ErrNotExist
			},
			want: []string{},
		},
		{
			name: "other open error propagated",
			openFn: func(string) (io.ReadCloser, error) {
				return nil, errors.New("permission denied")
			},
			wantErr: true,
		},
		{
			name: "read error propagated",
			openFn: func(string) (io.ReadCloser, error) {
				return erroringReadCloser{}, nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &shells.Linux{OpenFn: tt.openFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*shells.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Paths)
		})
	}
}
