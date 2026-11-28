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
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/cmd"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

// factsFiles writes, to a file system that validate then reads, the facts of
// an uptime collect as valid.json, and a document that breaks the schema as
// invalid.json. It returns the facts, the two paths, a path that does not
// exist, and the restore of the file system.
func factsFiles(
	s *suite.Suite,
) ([]byte, string, string, string, func()) {
	fs := memfs.New()
	dir := "/facts"
	s.Require().NoError(fs.MkdirAll(dir, 0o700))
	valid := fs.Join(dir, "valid.json")
	invalid := fs.Join(dir, "invalid.json")
	var facts bytes.Buffer
	s.Require().Zero(cmd.Execute(context.Background(),
		[]string{"collect", "--no-defaults", "--collector.uptime"},
		strings.NewReader(""), &facts, io.Discard))
	s.Require().NoError(fs.WriteFile(valid, facts.Bytes(), 0o600))
	s.Require().NoError(fs.WriteFile(invalid, []byte(`{"cpu": 7}`), 0o600))

	return facts.Bytes(), valid, invalid, fs.Join(dir, "missing.json"), cmd.SetHostFS(fs)
}

func (s *ValidatePublicTestSuite) TestReadInput() {
	facts, _, _, missing, restore := factsFiles(&s.Suite)
	defer restore()
	devNull, err := os.Open(os.DevNull)
	s.Require().NoError(err)
	defer func() { _ = devNull.Close() }()

	tests := []executeCase{
		{
			name: "validate reads stdin",
			args: []string{"validate"},
			in:   bytes.NewReader(facts),
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "schema valid")
			},
		},
		{
			name: "validate fails a file that is missing",
			args: []string{"validate", "--file", missing},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "read file")
			},
		},
		{
			name: "validate refuses a terminal on stdin",
			args: []string{"validate"},
			in:   devNull,
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "no input")
			},
		},
		{
			name: "validate refuses empty stdin",
			args: []string{"validate"},
			in:   strings.NewReader(""),
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "no input")
			},
		},
		{
			name: "validate fails when stdin fails",
			args: []string{"validate"},
			in:   iotest.ErrReader(errors.New("stdin closed")),
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "read stdin: stdin closed")
			},
		},
	}

	runExecute(&s.Suite, tests)
}

func (s *ValidatePublicTestSuite) TestRunValidate() {
	_, valid, invalid, _, restore := factsFiles(&s.Suite)
	defer restore()

	tests := []executeCase{
		{
			name: "validate passes a valid file",
			args: []string{"validate", "--file", valid},
			validateFunc: func(code int, out, errOut string) {
				s.Zero(code)
				s.Empty(errOut)
				s.Contains(out, "schema valid")
			},
		},
		{
			name: "validate fails a document that breaks the schema",
			args: []string{"validate", "--file", invalid},
			validateFunc: func(code int, _, errOut string) {
				s.Equal(1, code)
				s.Contains(errOut, "schema validation failed")
			},
		},
	}

	runExecute(&s.Suite, tests)
}
