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

// reset ends a colored span.
const reset = "\033[0m"

func (s *ThemePublicTestSuite) TestMute() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "a writer with color on gets secondary text",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[0;2mhello"+reset, got)
			},
		},
		{
			name: "a writer with color off gets plain text",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
		{
			name: "a writer that Colored did not make gets plain text",
			w:    &bytes.Buffer{},
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Mute(tc.w, "hello"))
		})
	}
}

func (s *ThemePublicTestSuite) TestAccent() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "a writer with color on gets the lavender accent",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;180;167;214mhello"+reset, got)
			},
		},
		{
			name: "a writer with color off gets plain text",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
		{
			name: "a writer that Colored did not make gets plain text",
			w:    &bytes.Buffer{},
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Accent(tc.w, "hello"))
		})
	}
}

func (s *ThemePublicTestSuite) TestOK() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "a writer with color on gets the success green",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;80;250;123mhello"+reset, got)
			},
		},
		{
			name: "a writer with color off gets plain text",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
		{
			name: "a writer that Colored did not make gets plain text",
			w:    &bytes.Buffer{},
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.OK(tc.w, "hello"))
		})
	}
}

func (s *ThemePublicTestSuite) TestErr() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "a writer with color on gets the error pink",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;255;110;199mhello"+reset, got)
			},
		},
		{
			name: "a writer with color off gets plain text",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
		{
			name: "a writer that Colored did not make gets plain text",
			w:    &bytes.Buffer{},
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Err(tc.w, "hello"))
		})
	}
}

func (s *ThemePublicTestSuite) TestInfo() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "a writer with color on gets the info cyan",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;0;212;255mhello"+reset, got)
			},
		},
		{
			name: "a writer with color off gets plain text",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
		{
			name: "a writer that Colored did not make gets plain text",
			w:    &bytes.Buffer{},
			validateFunc: func(got string) {
				s.Equal("hello", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Info(tc.w, "hello"))
		})
	}
}

func (s *ThemePublicTestSuite) TestBanner() {
	const (
		top = "█▀▀ █▀█ █░█ █▀█ █"
		bot = "█▄█ █▄█ █▀█ █░█ █"
	)
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "color on paints the top faint and the bottom lavender",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal(
					"\033[0;2m"+top+reset+"\n\033[38;2;180;167;214m"+bot+reset+"\n",
					got,
				)
			},
		},
		{
			name: "color off is the plain logo",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal(top+"\n"+bot+"\n", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Banner(tc.w))
		})
	}
}

func (s *ThemePublicTestSuite) TestSuccess() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "color on gives a colored mark",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;80;250;123m✓"+reset+" done", got)
			},
		},
		{
			name: "color off gives a text prefix",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("[ok] done", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Success(tc.w, "done"))
		})
	}
}

func (s *ThemePublicTestSuite) TestFailure() {
	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(string)
	}{
		{
			name: "color on gives a colored mark",
			w:    cli.Colored(&bytes.Buffer{}, true),
			validateFunc: func(got string) {
				s.Equal("\033[38;2;255;110;199m✗"+reset+" broken", got)
			},
		},
		{
			name: "color off gives a text prefix",
			w:    cli.Colored(&bytes.Buffer{}, false),
			validateFunc: func(got string) {
				s.Equal("[err] broken", got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(cli.Failure(tc.w, "broken"))
		})
	}
}

func (s *ThemePublicTestSuite) TestColored() {
	tests := []struct {
		name         string
		on           bool
		validateFunc func(*bytes.Buffer, io.Writer)
	}{
		{
			name: "it writes through to the writer it wraps",
			on:   true,
			validateFunc: func(buf *bytes.Buffer, w io.Writer) {
				_, err := io.WriteString(w, "text")
				s.Require().NoError(err)
				s.Equal("text", buf.String())
			},
		},
		{
			name: "off, it writes through too",
			validateFunc: func(buf *bytes.Buffer, w io.Writer) {
				_, err := io.WriteString(w, "text")
				s.Require().NoError(err)
				s.Equal("text", buf.String())
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			tc.validateFunc(&buf, cli.Colored(&buf, tc.on))
		})
	}
}
