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

package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

type KernelDarwinPublicTestSuite struct {
	suite.Suite
}

func TestKernelDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelDarwinPublicTestSuite))
}

func (s *KernelDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		unameFn func() (string, string, string, string, error)
		wantErr bool
		want    kernel.Info
	}{
		{
			name: "canonical macOS host",
			unameFn: func() (string, string, string, string, error) {
				return "Darwin", "23.4.0",
					"Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
					"arm64", nil
			},
			want: kernel.Info{
				Name: "Darwin", Release: "23.4.0",
				Version: "Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
				Machine: "arm64",
			},
		},
		{
			name: "uname error propagated",
			unameFn: func() (string, string, string, string, error) {
				return "", "", "", "", errors.New("uname failed")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &kernel.Darwin{UnameFn: tt.unameFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
