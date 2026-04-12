//go:build linux

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

// erroringReader fails on every Read.
type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

func (s *ShellsLinuxPublicTestSuite) TestParseShells() {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "canonical /etc/shells",
			content: "# comment line\n/bin/sh\n/bin/bash\n\n/usr/bin/zsh\n",
			want:    []string{"/bin/sh", "/bin/bash", "/usr/bin/zsh"},
		},
		{
			name:    "empty file",
			content: "",
			want:    []string{},
		},
		{
			name:    "only comments and blanks",
			content: "# only comments\n\n   \n",
			want:    []string{},
		},
		{
			name:    "whitespace trimmed",
			content: "  /bin/bash  \n\t/bin/sh\t\n",
			want:    []string{"/bin/bash", "/bin/sh"},
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

func (s *ShellsLinuxPublicTestSuite) TestParseShellsReadError() {
	_, err := shells.ParseShells(erroringReader{})
	s.Error(err)
}

func (s *ShellsLinuxPublicTestSuite) TestCollectFromFunc() {
	tests := []struct {
		name    string
		open    func(string) (io.ReadCloser, error)
		wantErr bool
		want    []string
	}{
		{
			name: "success",
			open: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("/bin/sh\n/bin/bash\n")}, nil
			},
			want: []string{"/bin/sh", "/bin/bash"},
		},
		{
			name:    "open error propagated",
			open:    func(string) (io.ReadCloser, error) { return nil, errors.New("no such file") },
			wantErr: true,
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
