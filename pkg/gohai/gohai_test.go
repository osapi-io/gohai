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

package gohai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/mocks"
)

type GohaiTestSuite struct {
	suite.Suite
}

func TestGohaiTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GohaiTestSuite))
}

// newCycleCollector returns a generated Collector mock declaring deps, for
// exercising dependency-cycle detection. Collect returns nothing.
func newCycleCollector(
	ctrl *gomock.Controller,
	name string,
	deps ...string,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(true).AnyTimes()
	m.EXPECT().Dependencies().Return(deps).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()

	return m
}

func (s *GohaiTestSuite) TestCollect() {
	tests := []struct {
		name         string
		register     func(*collector.Registry)
		validateFunc func(*Facts, error)
	}{
		{
			name: "a run error, such as a dependency cycle, is returned",
			register: func(reg *collector.Registry) {
				s.Require().
					NoError(reg.Register(newCycleCollector(gomock.NewController(s.T()), "a", "b")))
				s.Require().
					NoError(reg.Register(newCycleCollector(gomock.NewController(s.T()), "b", "a")))
			},
			validateFunc: func(_ *Facts, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			tt.register(reg)
			g := &Gohai{registry: reg}
			sel, err := reg.SelectedWith(true, nil, nil)
			s.Require().NoError(err)
			g.selected = sel
			tt.validateFunc(g.Collect(context.Background()))
		})
	}
}
