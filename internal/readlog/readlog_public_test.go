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

package readlog_test

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/readlog"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

type ReadlogPublicTestSuite struct {
	suite.Suite
}

func (s *ReadlogPublicTestSuite) TestSources() {
	tests := []struct {
		name         string
		validateFunc func(source.Sources, *readlog.Log)
	}{
		{
			name: "gives the family, and a clock at zero",
			validateFunc: func(src source.Sources, _ *readlog.Log) {
				s.Equal(family.Darwin, src.Family)
				s.Equal(int64(0), src.Now().Unix())
			},
		},
		{
			name: "the resolver resolves nothing",
			validateFunc: func(src source.Sources, _ *readlog.Log) {
				addrs, err := src.Resolver.LookupHost(context.Background(), "x")
				s.NoError(err)
				s.Nil(addrs)
				names, err := src.Resolver.LookupAddr(context.Background(), "1.2.3.4")
				s.NoError(err)
				s.Nil(names)
			},
		},
		{
			name: "the transport refuses each request, with no network",
			validateFunc: func(src source.Sources, _ *readlog.Log) {
				req, err := http.NewRequestWithContext(
					context.Background(),
					http.MethodGet,
					"http://169.254.169.254/latest/meta-data/",
					nil,
				)
				s.Require().NoError(err)
				resp, err := src.HTTP.RoundTrip(req)
				s.Nil(resp)
				s.EqualError(err, "readlog: the host has no network")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(readlog.Sources(family.Darwin))
		})
	}
}

func (s *ReadlogPublicTestSuite) TestFiles() {
	tests := []struct {
		name         string
		validateFunc func(source.Sources, *readlog.Log)
	}{
		{
			name: "the filesystem holds nothing, and logs each path once, sorted",
			validateFunc: func(src source.Sources, log *readlog.Log) {
				_, err := src.FS.Open("/a")
				s.ErrorIs(err, fs.ErrNotExist)
				_, _ = src.FS.OpenFile("/b", os.O_RDONLY, 0)
				_, _ = src.FS.ReadFile("/c")
				_, _ = src.FS.ReadDir("/d")
				_, _ = src.FS.Stat("/e")
				_, _ = src.FS.Lstat("/f")
				_, _ = src.FS.Readlink("/g")
				_, _ = src.FS.EvalSymlinks("/h")
				_, _ = src.FS.Glob("/i/*")
				_ = src.FS.WalkDir("/j", func(string, fs.DirEntry, error) error { return nil })
				_, _ = src.FS.ReadFile("/c")

				s.Equal(
					[]string{"/a", "/b", "/c", "/d", "/e", "/f", "/g", "/h", "/i/*", "/j"},
					log.Files(),
				)
			},
		},
		{
			name: "a Sub logs its directory, and its reads with their full path",
			validateFunc: func(src source.Sources, log *readlog.Log) {
				root, err := src.FS.Sub("/")
				s.Require().NoError(err)
				_, _ = root.ReadFile("etc/os-release")
				_ = root.MkdirAll("etc/ssh", 0o755)
				etc, err := root.Sub("etc")
				s.Require().NoError(err)
				_, _ = etc.ReadFile("ssh/sshd_config")
				_, err = src.FS.Sub("/missing")
				s.Error(err)

				s.Equal(
					[]string{"/", "/etc", "/etc/os-release", "/etc/ssh/sshd_config", "/missing"},
					log.Files(),
				)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(readlog.Sources(family.Darwin))
		})
	}
}

func (s *ReadlogPublicTestSuite) TestCommands() {
	tests := []struct {
		name         string
		validateFunc func(source.Sources, *readlog.Log)
	}{
		{
			name: "a command answers with no output, and is logged",
			validateFunc: func(src source.Sources, log *readlog.Log) {
				out, err := src.Exec.Execute(
					context.Background(),
					"systemctl",
					"list-units",
					"--all",
				)
				s.NoError(err)
				s.Nil(out)
				_, _ = src.Exec.Execute(context.Background(), "hostname")

				s.Equal([]string{"hostname", "systemctl list-units --all"}, log.Commands())
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(readlog.Sources(family.Darwin))
		})
	}
}

func TestReadlogPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadlogPublicTestSuite))
}
