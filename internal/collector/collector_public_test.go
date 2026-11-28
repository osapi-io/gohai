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

package collector_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type CollectorPublicTestSuite struct {
	suite.Suite
}

func (s *CollectorPublicTestSuite) TestAbsent() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(bool)
	}{
		{
			name:         "a file that does not exist is absent",
			err:          fmt.Errorf("read: %w", fs.ErrNotExist),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a command that is not on the PATH is absent",
			err:          fmt.Errorf("exec ps: %w", exec.ErrNotFound),
			validateFunc: func(got bool) { s.True(got) },
		},
		{
			name:         "a denied read is not absent",
			err:          fmt.Errorf("read: %w", fs.ErrPermission),
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "a command that failed is not absent",
			err:          errors.New("exit status 1"),
			validateFunc: func(got bool) { s.False(got) },
		},
		{
			name:         "no error is not absent",
			validateFunc: func(got bool) { s.False(got) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(collector.Absent(tt.err))
		})
	}
}

func TestCollectorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CollectorPublicTestSuite))
}
