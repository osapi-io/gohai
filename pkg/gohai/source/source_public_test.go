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

package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

type SourcePublicTestSuite struct {
	suite.Suite
}

func (s *SourcePublicTestSuite) TestHost() {
	h := source.Host()
	s.NotNil(h.FS)
	s.NotNil(h.Exec)
	s.NotNil(h.Now)
	s.Equal(family.Detect(), h.Family)
}

func (s *SourcePublicTestSuite) TestDefaultsKeepsWhatIsGiven() {
	fs := memfs.New()
	now := func() time.Time { return time.Unix(0, 0) }
	got := source.Sources{FS: fs, Now: now, Family: family.Debian}.Defaults()
	s.Same(fs, got.FS)
	s.Equal(time.Unix(0, 0), got.Now())
	s.Equal(family.Debian, got.Family)
	s.NotNil(got.Exec)
}

func (s *SourcePublicTestSuite) TestDefaultsFamilyFromFS() {
	fs := memfs.New()
	s.Require().NoError(fs.MkdirAll("/etc", 0o755))
	s.Require().NoError(fs.WriteFile("/etc/os-release", []byte("ID=rocky\n"), 0o644))

	s.Equal(family.RHEL, source.Sources{FS: fs}.Defaults().Family)
	s.Empty(source.Sources{FS: memfs.New()}.Defaults().Family,
		"a filesystem with no os-release is no family, not the real host's")
}

func (s *SourcePublicTestSuite) TestCommands() {
	tests := []struct {
		name    string
		command string
		args    []string
		want    string
		wantErr string
	}{
		{
			name:    "when the command succeeds",
			command: "echo",
			args:    []string{"hello"},
			want:    "hello\n",
		},
		{
			name:    "when the command writes to stderr, only stdout is data",
			command: "sh",
			args:    []string{"-c", "echo out; echo warning >&2"},
			want:    "out\n",
		},
		{
			name:    "when the command runs, LC_ALL is C",
			command: "sh",
			args:    []string{"-c", "echo $LC_ALL"},
			want:    "C\n",
		},
		{name: "when the command fails", command: "false", wantErr: "exec false"},
		{
			name:    "when the command fails, the error has its stderr",
			command: "sh",
			args:    []string{"-c", "echo no such container >&2; exit 3"},
			wantErr: "exit status 3: no such container",
		},
		{
			name:    "when the command does not exist",
			command: "gohai-no-such-command",
			wantErr: "exec gohai-no-such-command",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			out, err := source.Commands().Execute(context.Background(), tc.command, tc.args...)
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(tc.want, string(out))
		})
	}
}

func (s *SourcePublicTestSuite) TestCommandsWith() {
	tests := []struct {
		name    string
		ctx     func() context.Context
		args    []string
		limit   int
		want    string
		wantErr string
	}{
		{
			name:    "when the command runs past the timeout",
			args:    []string{"-c", "sleep 5"},
			wantErr: "exec sh: stopped after 200ms",
		},
		{
			name: "when a child keeps the pipes open after the command exits",
			args: []string{"-c", "echo done; sleep 5 &"},
			want: "done\n",
		},
		{
			name:    "when stdout is more than the limit",
			args:    []string{"-c", "printf 0123456789abcdef; printf more"},
			limit:   10,
			want:    "0123456789",
			wantErr: "exec sh: stdout is more than 10 bytes",
		},
		{
			name: "when the caller's context is cancelled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			args:    []string{"-c", "sleep 5"},
			wantErr: "exec sh: context canceled",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}
			limit := tc.limit
			if limit == 0 {
				limit = source.MaxOutput
			}
			start := time.Now()
			out, err := source.CommandsWith(200*time.Millisecond, 100*time.Millisecond, limit).
				Execute(ctx, "sh", tc.args...)
			s.Less(time.Since(start), 3*time.Second)
			s.Equal(tc.want, string(out))
			if tc.wantErr != "" {
				s.ErrorContains(err, tc.wantErr)

				return
			}
			s.NoError(err)
		})
	}
}

func TestSourcePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SourcePublicTestSuite))
}
