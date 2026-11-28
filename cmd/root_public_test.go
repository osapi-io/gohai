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
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type RootPublicTestSuite struct {
	suite.Suite
}

func TestRootPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RootPublicTestSuite))
}

// failWriter fails each write, as a closed pipe does.
type failWriter struct{}

func (failWriter) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("broken pipe")
}

// executeCase is one run of the gohai command, a row of the test of a
// function that the command reaches.
type executeCase struct {
	name        string
	ctx         context.Context
	args        []string
	in          io.Reader
	out         io.Writer
	newRegistry func() (gohai.Registry, error)
	// setup, when set, runs before the command and returns its restore.
	setup        func() func()
	validateFunc func(code int, out, errOut string)
}

// runExecute runs each case through cmd.Execute.
func runExecute(
	s *suite.Suite,
	tests []executeCase,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.newRegistry != nil {
				defer cmd.SetNewRegistryFn(tt.newRegistry)()
			}
			if tt.setup != nil {
				defer tt.setup()()
			}
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			in := tt.in
			if in == nil {
				in = strings.NewReader("")
			}
			var out, errOut bytes.Buffer
			w := tt.out
			if w == nil {
				w = &out
			}
			code := cmd.Execute(ctx, tt.args, in, w, &errOut)
			tt.validateFunc(code, out.String(), errOut.String())
		})
	}
}

func (s *RootPublicTestSuite) TestExecute() {
	tests := []executeCase{
		{
			name: "no command prints the help, with the banner",
			args: nil,
			validateFunc: func(code int, out, _ string) {
				s.Zero(code)
				s.Contains(out, "Collect system facts")
				s.Contains(out, "collector")
			},
		},
		{
			name: "help of a subcommand has no banner",
			args: []string{"collect", "--help"},
			validateFunc: func(code int, out, _ string) {
				s.Zero(code)
				s.Contains(out, "--format ohai")
			},
		},
	}

	runExecute(&s.Suite, tests)
}
