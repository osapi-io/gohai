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

package lsb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
)

type LSBLinuxPublicTestSuite struct {
	suite.Suite
}

func TestLSBLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LSBLinuxPublicTestSuite))
}

// lsbReleaseExec returns a MockExecutor that canned-answers
// `lsb_release -a`. Shared across tests.
func lsbReleaseExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lsb_release", "-a").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *LSBLinuxPublicTestSuite) TestCollect() {
	cliFull := []byte(`Distributor ID:	Ubuntu
Description:	Ubuntu 24.04 LTS
Release:	24.04
Codename:	noble
`)

	tests := []struct {
		name string
		exec executor.Executor
		want lsb.Info
	}{
		{
			name: "CLI succeeds: four fields populated",
			exec: lsbReleaseExec(s.T(), cliFull, nil),
			want: lsb.Info{
				ID: "Ubuntu", Release: "24.04", Codename: "noble",
				Description: "Ubuntu 24.04 LTS",
			},
		},
		{
			name: "CLI returns subset: only matching fields populated",
			exec: lsbReleaseExec(s.T(), []byte("Distributor ID:\tDebian\nRelease:\t12\n"), nil),
			want: lsb.Info{ID: "Debian", Release: "12"},
		},
		{
			name: "CLI output with unmatched / no-colon lines: those are skipped",
			exec: lsbReleaseExec(s.T(),
				[]byte("no colon line here\nunrelated: value\nDistributor ID:\tArch\n"),
				nil),
			want: lsb.Info{ID: "Arch"},
		},
		{
			name: "CLI missing: empty Info, no error",
			exec: lsbReleaseExec(s.T(), nil, errors.New("not found")),
			want: lsb.Info{},
		},
		{
			name: "nil Exec: empty Info, no error",
			exec: nil,
			want: lsb.Info{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &lsb.Linux{Exec: tt.exec}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*lsb.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
