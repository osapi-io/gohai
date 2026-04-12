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

type ShellsDarwinPublicTestSuite struct {
	suite.Suite
}

func TestShellsDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsDarwinPublicTestSuite))
}

func (s *ShellsDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		openFn  func(string) (io.ReadCloser, error)
		wantErr bool
		want    []string
	}{
		{
			name: "macOS /etc/shells",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{
					strings.NewReader("# comment\n/bin/bash\n/bin/zsh\n/bin/sh\n"),
				}, nil
			},
			want: []string{"/bin/bash", "/bin/zsh", "/bin/sh"},
		},
		{
			name: "non-absolute entries skipped",
			openFn: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("/bin/bash\nnologin\n/bin/zsh\n")}, nil
			},
			want: []string{"/bin/bash", "/bin/zsh"},
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
			c := &shells.Darwin{OpenFn: tt.openFn}
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

// TestNewDarwinWiresUpRealOpen exercises the real os.Open closure the
// factory wires in.
func (s *ShellsDarwinPublicTestSuite) TestNewDarwinWiresUpRealOpen() {
	c := shells.NewDarwin()
	s.Require().NotNil(c.OpenFn)

	rc, err := c.OpenFn("/dev/null")
	s.Require().NoError(err)
	s.Require().NoError(rc.Close())

	_, err = c.OpenFn("/gohai-test-does-not-exist/shells")
	s.Error(err)
}
