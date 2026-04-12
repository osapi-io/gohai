//go:build darwin

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
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

type ShellsDarwinPublicTestSuite struct {
	suite.Suite
}

func TestShellsDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsDarwinPublicTestSuite))
}

type stringReadCloser struct{ *strings.Reader }

func (stringReadCloser) Close() error { return nil }

type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

func (s *ShellsDarwinPublicTestSuite) TestParseShells() {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "macOS /etc/shells",
			content: "# comment\n/bin/bash\n/bin/zsh\n/bin/sh\n",
			want:    []string{"/bin/bash", "/bin/zsh", "/bin/sh"},
		},
		{
			name:    "empty file",
			content: "",
			want:    []string{},
		},
		{
			name:    "non-absolute entries skipped",
			content: "/bin/bash\nnologin\n/bin/zsh\n",
			want:    []string{"/bin/bash", "/bin/zsh"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			info, err := shells.ParseShells(strings.NewReader(tt.content))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, info.Paths)
		})
	}
}

func (s *ShellsDarwinPublicTestSuite) TestParseShellsReadError() {
	_, err := shells.ParseShells(erroringReader{})
	s.Error(err)
}

func (s *ShellsDarwinPublicTestSuite) TestCollectFromFunc() {
	tests := []struct {
		name    string
		open    func(string) (io.ReadCloser, error)
		wantErr bool
		want    []string
	}{
		{
			name: "success",
			open: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("/bin/zsh\n/bin/bash\n")}, nil
			},
			want: []string{"/bin/zsh", "/bin/bash"},
		},
		{
			name:    "other open error propagated",
			open:    func(string) (io.ReadCloser, error) { return nil, errors.New("permission denied") },
			wantErr: true,
		},
		{
			name: "missing /etc/shells soft-misses to empty list",
			open: func(string) (io.ReadCloser, error) {
				return nil, os.ErrNotExist
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			info, err := shells.CollectFromFunc(tt.open)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, info.Paths)
		})
	}
}
