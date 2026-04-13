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

package executor_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/executor"
)

// TestMain enables self-exec subprocess testing. When
// GOHAI_EXECUTOR_TEST_OUT is set, the test binary emulates stdout +
// optional non-zero exit — no reliance on /bin/echo or other external
// binaries.
func TestMain(m *testing.M) {
	if out := os.Getenv("GOHAI_EXECUTOR_TEST_OUT"); out != "" {
		fmt.Print(out)
		if os.Getenv("GOHAI_EXECUTOR_TEST_EXIT") == "1" {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type ExecutorPublicTestSuite struct {
	suite.Suite
}

func TestExecutorPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorPublicTestSuite))
}

func (s *ExecutorPublicTestSuite) TestNew() {
	s.NotNil(executor.New())
}

func (s *ExecutorPublicTestSuite) TestExecute() {
	tests := []struct {
		name    string
		setenv  map[string]string
		cmd     string
		args    []string
		wantOut string
		wantErr bool
	}{
		{
			name:    "success returns combined output",
			setenv:  map[string]string{"GOHAI_EXECUTOR_TEST_OUT": "hello"},
			cmd:     os.Args[0],
			wantOut: "hello",
		},
		{
			name:    "non-zero exit returns wrapped error with output captured",
			setenv:  map[string]string{"GOHAI_EXECUTOR_TEST_OUT": "boom\n", "GOHAI_EXECUTOR_TEST_EXIT": "1"},
			cmd:     os.Args[0],
			wantOut: "boom\n",
			wantErr: true,
		},
		{
			name:    "missing binary returns wrapped error",
			cmd:     "/gohai-test-does-not-exist",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			for k, v := range tt.setenv {
				s.T().Setenv(k, v)
			}
			e := executor.New()
			out, err := e.Execute(context.Background(), tt.cmd, tt.args...)
			if tt.wantErr {
				s.Error(err)
				if tt.wantOut != "" {
					s.Equal(tt.wantOut, string(out))
				}
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantOut, string(out))
		})
	}
}
