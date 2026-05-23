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

func (s *ThemePublicTestSuite) TestMute() {
	tests := []struct {
		name     string
		w        io.Writer
		tty      bool
		wantAnsi bool
	}{
		{name: "non-file writer returns plain", w: &bytes.Buffer{}, tty: false, wantAnsi: false},
		{name: "file non-TTY returns plain", w: devNull(s.T()), tty: false, wantAnsi: false},
		{name: "file TTY wraps with ANSI", w: devNull(s.T()), tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Mute(tc.w, "hello")
			restore()

			s.Contains(got, "hello")
			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.Equal("hello", got)
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestAccent() {
	tests := []struct {
		name     string
		tty      bool
		wantAnsi bool
	}{
		{name: "non-TTY returns plain text", tty: false, wantAnsi: false},
		{name: "TTY wraps with ANSI", tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Accent(devNull(s.T()), "hello")
			restore()

			s.Contains(got, "hello")
			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.Equal("hello", got)
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestOK() {
	tests := []struct {
		name     string
		tty      bool
		wantAnsi bool
	}{
		{name: "non-TTY returns plain text", tty: false, wantAnsi: false},
		{name: "TTY wraps with ANSI", tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.OK(devNull(s.T()), "hello")
			restore()

			s.Contains(got, "hello")
			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.Equal("hello", got)
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestErr() {
	tests := []struct {
		name     string
		tty      bool
		wantAnsi bool
	}{
		{name: "non-TTY returns plain text", tty: false, wantAnsi: false},
		{name: "TTY wraps with ANSI", tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Err(devNull(s.T()), "hello")
			restore()

			s.Contains(got, "hello")
			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.Equal("hello", got)
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestInfo() {
	tests := []struct {
		name     string
		tty      bool
		wantAnsi bool
	}{
		{name: "non-TTY returns plain text", tty: false, wantAnsi: false},
		{name: "TTY wraps with ANSI", tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Info(devNull(s.T()), "hello")
			restore()

			s.Contains(got, "hello")
			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.Equal("hello", got)
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestBanner() {
	tests := []struct {
		name     string
		tty      bool
		wantAnsi bool
	}{
		{name: "non-TTY plain text", tty: false, wantAnsi: false},
		{name: "TTY with ANSI colors", tty: true, wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Banner(devNull(s.T()))
			restore()

			s.Contains(got, "█▀▀ █▀█ █░█ █▀█ █")
			s.Contains(got, "█▄█ █▄█ █▀█ █░█ █")

			if tc.wantAnsi {
				s.Contains(got, "\033[")
			} else {
				s.NotContains(got, "\033[")
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestSuccess() {
	tests := []struct {
		name     string
		tty      bool
		contains string
		wantAnsi bool
	}{
		{name: "non-TTY prefix", tty: false, contains: "[ok] done", wantAnsi: false},
		{name: "TTY colored mark", tty: true, contains: "done", wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Success(devNull(s.T()), "done")
			restore()

			s.Contains(got, tc.contains)

			if tc.wantAnsi {
				s.Contains(got, "\033[")
				s.Contains(got, "✓")
			}
		})
	}
}

func (s *ThemePublicTestSuite) TestFailure() {
	tests := []struct {
		name     string
		tty      bool
		contains string
		wantAnsi bool
	}{
		{name: "non-TTY prefix", tty: false, contains: "[err] broken", wantAnsi: false},
		{name: "TTY colored mark", tty: true, contains: "broken", wantAnsi: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			restore := cli.SetIsTerminalFn(func(_ int) bool { return tc.tty })
			got := cli.Failure(devNull(s.T()), "broken")
			restore()

			s.Contains(got, tc.contains)

			if tc.wantAnsi {
				s.Contains(got, "\033[")
				s.Contains(got, "✗")
			}
		})
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
