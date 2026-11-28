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

package cmd_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
)

type ColorPublicTestSuite struct {
	suite.Suite
}

func TestColorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ColorPublicTestSuite))
}

func (s *ColorPublicTestSuite) TestColorFor() {
	tests := []struct {
		name         string
		env          map[string]string
		terminal     bool
		validateFunc func(bool)
	}{
		{
			name:         "a terminal takes color",
			terminal:     true,
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a writer that is not a terminal takes no color",
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "NO_COLOR turns color off on a terminal",
			env:          map[string]string{"NO_COLOR": "1"},
			terminal:     true,
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "TERM=dumb turns color off on a terminal",
			env:          map[string]string{"TERM": "dumb"},
			terminal:     true,
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "another TERM keeps color on a terminal",
			env:          map[string]string{"TERM": "xterm-256color"},
			terminal:     true,
			validateFunc: func(got bool) { s.True(got) },
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := cmd.ColorFor(
				&bytes.Buffer{},
				func(k string) string { return tt.env[k] },
				func(io.Writer) bool { return tt.terminal },
			)
			tt.validateFunc(got)
		})
	}
}

func (s *ColorPublicTestSuite) TestTerminal() {
	file, err := os.Create(filepath.Join(s.T().TempDir(), "out"))
	s.Require().NoError(err)
	s.T().Cleanup(func() { _ = file.Close() })

	tests := []struct {
		name         string
		w            io.Writer
		validateFunc func(bool)
	}{
		{
			name:         "a buffer is not a terminal",
			w:            &bytes.Buffer{},
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "a regular file is not a terminal",
			w:            file,
			validateFunc: func(got bool) { s.False(got) },
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(cmd.Terminal(tt.w))
		})
	}
}
