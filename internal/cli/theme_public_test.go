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

package cli_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cli"
)

type ThemePublicTestSuite struct {
	suite.Suite
}

func TestThemePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ThemePublicTestSuite))
}

func (s *ThemePublicTestSuite) TestIsTTY() {
	tests := []struct {
		name string
		w    io.Writer
		tty  bool
		want bool
	}{
		{
			name: "buffer is never a TTY",
			w:    &bytes.Buffer{},
			tty:  false,
			want: false,
		},
		{
			name: "file with terminal returns true",
			w:    devNull(s.T()),
			tty:  true,
			want: true,
		},
		{
			name: "file without terminal returns false",
			w:    devNull(s.T()),
			tty:  false,
			want: false,
		},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := cli.IsTTY(tc.w)
		restore()

		s.Equal(tc.want, got)
	}
}

func (s *ThemePublicTestSuite) TestColorize() {
	tests := []struct {
		name string
		tty  bool
		ansi string
		text string
		want string
	}{
		{
			name: "non-TTY returns plain text",
			tty:  false,
			ansi: "\033[0;2m",
			text: "hello",
			want: "hello",
		},
		{
			name: "TTY wraps with ANSI codes",
			tty:  true,
			ansi: "\033[0;2m",
			text: "hello",
			want: "\033[0;2mhello\033[0m",
		},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := cli.Colorize(devNull(s.T()), tc.ansi, tc.text)
		restore()

		s.Equal(tc.want, got)
	}
}

func (s *ThemePublicTestSuite) TestColorRoles() {
	tests := []struct {
		name string
		tty  bool
		fn   func(io.Writer, string) string
	}{
		{name: "Mute non-TTY", tty: false, fn: cli.Mute},
		{name: "Mute TTY", tty: true, fn: cli.Mute},
		{name: "Accent non-TTY", tty: false, fn: cli.Accent},
		{name: "Accent TTY", tty: true, fn: cli.Accent},
		{name: "OK non-TTY", tty: false, fn: cli.OK},
		{name: "OK TTY", tty: true, fn: cli.OK},
		{name: "Err non-TTY", tty: false, fn: cli.Err},
		{name: "Err TTY", tty: true, fn: cli.Err},
		{name: "Info non-TTY", tty: false, fn: cli.Info},
		{name: "Info TTY", tty: true, fn: cli.Info},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := tc.fn(devNull(s.T()), "hello")
		restore()

		s.Contains(got, "hello")

		if tc.tty {
			s.Contains(got, "\033[")
		} else {
			s.Equal("hello", got)
		}
	}
}

func (s *ThemePublicTestSuite) TestBanner() {
	tests := []struct {
		name    string
		tty     bool
		hasAnsi bool
	}{
		{name: "non-TTY plain text", tty: false, hasAnsi: false},
		{name: "TTY with ANSI colors", tty: true, hasAnsi: true},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := cli.Banner(devNull(s.T()))
		restore()

		s.Contains(got, "█▀▀ █▀█ █░█ █▀█ █")
		s.Contains(got, "█▄█ █▄█ █▀█ █░█ █")

		if tc.hasAnsi {
			s.Contains(got, "\033[")
		} else {
			s.NotContains(got, "\033[")
		}
	}
}

func (s *ThemePublicTestSuite) TestSuccess() {
	tests := []struct {
		name     string
		tty      bool
		wantText string
		wantAnsi bool
	}{
		{name: "non-TTY prefix", tty: false, wantText: "[ok] done", wantAnsi: false},
		{name: "TTY colored mark", tty: true, wantText: "done", wantAnsi: true},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := cli.Success(devNull(s.T()), "done")
		restore()

		s.Contains(got, tc.wantText)

		if tc.wantAnsi {
			s.Contains(got, "\033[")
			s.Contains(got, "✓")
		}
	}
}

func (s *ThemePublicTestSuite) TestFailure() {
	tests := []struct {
		name     string
		tty      bool
		wantText string
		wantAnsi bool
	}{
		{name: "non-TTY prefix", tty: false, wantText: "[err] broken", wantAnsi: false},
		{name: "TTY colored mark", tty: true, wantText: "broken", wantAnsi: true},
	}

	for _, tc := range tests {
		restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
		got := cli.Failure(devNull(s.T()), "broken")
		restore()

		s.Contains(got, tc.wantText)

		if tc.wantAnsi {
			s.Contains(got, "\033[")
			s.Contains(got, "✗")
		}
	}
}

func (s *ThemePublicTestSuite) TestPrint() {
	var buf bytes.Buffer
	cli.Print(&buf, "hello")

	s.Equal("hello\n", buf.String())
}

func (s *ThemePublicTestSuite) TestPrintf() {
	var buf bytes.Buffer
	cli.Printf(&buf, "count: %d", 42)

	s.Equal("count: 42", buf.String())
}

func devNull(
	t *testing.T,
) *os.File {
	t.Helper()

	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = f.Close() })

	return f
}
