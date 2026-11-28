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

package sourcetest_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/sourcetest"
)

type SourcetestPublicTestSuite struct {
	suite.Suite
}

func (s *SourcetestPublicTestSuite) TestFS() {
	fs := sourcetest.FS(s.T(), map[string]string{"/proc/loadavg": "0.1 0.2 0.3"})
	b, err := fs.ReadFile("/proc/loadavg")
	s.Require().NoError(err)
	s.Equal("0.1 0.2 0.3", string(b))
}

func (s *SourcetestPublicTestSuite) TestLinks() {
	fs := sourcetest.Links(s.T(), sourcetest.FS(s.T(), nil), map[string]string{
		"/sys/bus/pci/devices/0000:00:02.0/driver": "../../../bus/pci/drivers/i915",
	})
	target, err := fs.Readlink("/sys/bus/pci/devices/0000:00:02.0/driver")
	s.Require().NoError(err)
	s.Equal("../../../bus/pci/drivers/i915", target)
}

func (s *SourcetestPublicTestSuite) TestFailRead() {
	tests := []struct {
		name         string
		paths        []string
		read         string
		validateFunc func([]byte, error)
	}{
		{
			name:  "a read of a path fails, with an error that is not not-exist",
			paths: []string{"/proc/mdstat"},
			read:  "/proc/mdstat",
			validateFunc: func(b []byte, err error) {
				s.Require().Error(err)
				s.NotErrorIs(err, fs.ErrNotExist)
				s.Empty(b)
			},
		},
		{
			name:  "a file that is not named still reads",
			paths: []string{"/proc/mdstat"},
			read:  "/etc/os-release",
			validateFunc: func(b []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ID=fedora\n", string(b))
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			vfs := sourcetest.FailRead(
				s.T(),
				sourcetest.FS(s.T(), map[string]string{"/etc/os-release": "ID=fedora\n"}),
				tt.paths...,
			)
			tt.validateFunc(vfs.ReadFile(tt.read))
		})
	}
}

func (s *SourcetestPublicTestSuite) TestCommands() {
	r := sourcetest.Commands{
		Out: map[string]string{"sysctl -n vm.loadavg": "{ 1 2 3 }", "df -kP": "partial"},
		Err: map[string]error{"df -kP": errors.New("exit status 1")},
	}.Mock(s.T())
	out, err := r.Execute(context.Background(), "sysctl", "-n", "vm.loadavg")
	s.Require().NoError(err)
	s.Equal("{ 1 2 3 }", string(out))

	out, err = r.Execute(context.Background(), "df", "-kP")
	s.EqualError(err, "exit status 1")
	s.Equal("partial", string(out))

	_, err = r.Execute(context.Background(), "nope")
	s.ErrorContains(err, "exec nope")
}

func (s *SourcetestPublicTestSuite) TestHTTP() {
	tests := []struct {
		name         string
		url          string
		validateFunc func(*http.Response)
	}{
		{
			name: "answers a URL it has with 200 and the body",
			url:  "http://169.254.169.254/latest/meta-data/instance-id",
			validateFunc: func(resp *http.Response) {
				s.Equal(http.StatusOK, resp.StatusCode)
				b, err := io.ReadAll(resp.Body)
				s.Require().NoError(err)
				s.Equal("i-123", string(b))
			},
		},
		{
			name: "answers any other URL with 404",
			url:  "http://169.254.169.254/nope",
			validateFunc: func(resp *http.Response) {
				s.Equal(http.StatusNotFound, resp.StatusCode)
			},
		},
	}
	rt := sourcetest.HTTP(map[string]string{
		"http://169.254.169.254/latest/meta-data/instance-id": "i-123",
	})
	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				tt.url,
				nil,
			)
			s.Require().NoError(err)
			resp, err := rt.RoundTrip(req)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()
			tt.validateFunc(resp)
		})
	}
}

func TestSourcetestPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SourcetestPublicTestSuite))
}
