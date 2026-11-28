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

package spec_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/spec"
	"github.com/osapi-io/gohai/pkg/gohai"
)

type SpecPublicTestSuite struct {
	suite.Suite
}

func TestSpecPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SpecPublicTestSuite))
}

func (s *SpecPublicTestSuite) TestProblems() {
	fields := []gohai.Field{{Name: "x", Type: "string"}}
	tests := []struct {
		name         string
		specs        []gohai.CollectorSpec
		validateFunc func(rules []string)
	}{
		{
			name:  "when the built-in collectors break no rule",
			specs: gohai.Collectors(),
			validateFunc: func(rules []string) {
				s.Empty(rules)
			},
		},
		{
			name: "when two collectors have one name",
			specs: []gohai.CollectorSpec{
				{Name: "a", Category: "system", Fields: fields},
				{Name: "a", Category: "system", Fields: fields},
			},
			validateFunc: func(rules []string) {
				s.Equal([]string{spec.RuleName}, rules)
			},
		},
		{
			name: "when a name is not lowercase letters, digits and underscores",
			specs: []gohai.CollectorSpec{
				{Name: "Upper", Category: "system", Fields: fields},
				{Name: "9lives", Category: "system", Fields: fields},
				{Name: "has-dash", Category: "system", Fields: fields},
				{Name: "", Category: "system", Fields: fields},
			},
			validateFunc: func(rules []string) {
				s.Equal([]string{spec.RuleName, spec.RuleName, spec.RuleName, spec.RuleName}, rules)
			},
		},
		{
			name:  "when the category is not one of the categories",
			specs: []gohai.CollectorSpec{{Name: "a", Category: "kitchen", Fields: fields}},
			validateFunc: func(rules []string) {
				s.Equal([]string{spec.RuleCategory}, rules)
			},
		},
		{
			name:  "when gohai.Facts has no field for it",
			specs: []gohai.CollectorSpec{{Name: "a", Category: "system"}},
			validateFunc: func(rules []string) {
				s.Equal([]string{spec.RuleFacts}, rules)
			},
		},
		{
			name: "when a dependency is itself or not registered",
			specs: []gohai.CollectorSpec{
				{
					Name:         "a",
					Category:     "system",
					Fields:       fields,
					Dependencies: []string{"a", "missing"},
				},
			},
			validateFunc: func(rules []string) {
				s.Equal([]string{spec.RuleDependency, spec.RuleDependency}, rules)
			},
		},
	}

	seen := map[string]bool{}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			var rules []string
			for _, p := range spec.Problems(tc.specs) {
				rules = append(rules, p.Rule)
				seen[p.Rule] = true
				s.NotEmpty(p.Message)
				// Every rule has a fix.
				s.NotEmpty(p.Fix)
			}
			tc.validateFunc(rules)
		})
	}

	// Each check is a rule that Problems applies, and each rule has
	// a check: the cases above trigger every one.
	for _, c := range spec.Checks() {
		s.Run("a case triggers the rule "+c.Rule, func() {
			s.NotEmpty(c.Requires)
			s.True(seen[c.Rule])
		})
	}
}
