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

package fips_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
)

type FipsLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFipsLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FipsLinuxPublicTestSuite))
}

type stringReadCloser struct{ *strings.Reader }

func (stringReadCloser) Close() error { return nil }

type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

func (s *FipsLinuxPublicTestSuite) TestParseFips() {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{"enabled", "1\n", true, false},
		{"disabled", "0\n", false, false},
		{"no trailing newline enabled", "1", true, false},
		{"empty file", "", false, false},
		{"unexpected value", "yes", false, false},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			info, err := fips.ParseFips(strings.NewReader(tt.content))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, info.Kernel.Enabled)
		})
	}
}

func (s *FipsLinuxPublicTestSuite) TestParseFipsReadError() {
	_, err := fips.ParseFips(erroringReader{})
	s.Error(err)
}

func (s *FipsLinuxPublicTestSuite) TestCollectFromFunc() {
	tests := []struct {
		name    string
		open    func(string) (io.ReadCloser, error)
		wantErr bool
		want    bool
	}{
		{
			name: "enabled",
			open: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("1\n")}, nil
			},
			want: true,
		},
		{
			name: "disabled",
			open: func(string) (io.ReadCloser, error) {
				return stringReadCloser{strings.NewReader("0\n")}, nil
			},
			want: false,
		},
		{
			name: "file missing reports disabled",
			open: func(string) (io.ReadCloser, error) {
				return nil, os.ErrNotExist
			},
			want: false,
		},
		{
			name:    "other open error propagated",
			open:    func(string) (io.ReadCloser, error) { return nil, errors.New("permission denied") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			info, err := fips.CollectFromFunc(tt.open)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, info.Kernel.Enabled)
		})
	}
}
