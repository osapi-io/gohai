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

package gohai_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type SpecPublicTestSuite struct {
	suite.Suite
}

func (s *SpecPublicTestSuite) TestCollectors() {
	specs := gohai.Collectors()
	s.Len(specs, len(gohai.NewRegistry().Names()), "one spec for each registered collector")

	byName := map[string]gohai.CollectorSpec{}
	for _, c := range specs {
		byName[c.Name] = c
		s.Len(c.Reads, len(gohai.SpecPlatforms()), "%s has the reads of each platform", c.Name)
	}

	dns := byName["dns"]
	s.Equal("dns.NewWith(src)", dns.Constructor)
	s.Equal("github.com/osapi-io/gohai/pkg/gohai/collectors/dns", dns.Package)
	s.Empty(dns.Network)
	s.Contains(dns.Fields, gohai.Field{Name: "nameservers", Type: "[]string"})
	s.Equal("debian", dns.Reads[0].Platform)
	s.Contains(dns.Reads[0].Files, "/etc/resolv.conf")
	s.Contains(dns.Reads[0].Commands, "resolvectl dns")
	s.Equal("darwin", dns.Reads[2].Platform)
	s.Contains(dns.Reads[2].Commands, "scutil --dns")

	ec2 := byName["ec2"]
	s.Equal("ec2.New()", ec2.Constructor)
	s.Contains(ec2.Network, "metadata service")
	s.Equal([]string{"dmi"}, byName["alibaba"].Dependencies)
}

func (s *SpecPublicTestSuite) TestCollectorByName() {
	tests := []struct {
		name   string
		lookup string
		wantOK bool
	}{
		{name: "when the collector exists", lookup: "services", wantOK: true},
		{name: "when no collector has the name", lookup: "nope", wantOK: false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, ok := gohai.CollectorByName(tc.lookup)
			s.Equal(tc.wantOK, ok)
			if ok {
				s.Equal(tc.lookup, got.Name)
			}
		})
	}
}

func (s *SpecPublicTestSuite) TestCollectorProblems() {
	fields := []gohai.Field{{Name: "x", Type: "string"}}
	tests := []struct {
		name      string
		specs     []gohai.CollectorSpec
		wantRules []string
	}{
		{
			name:  "when the built-in collectors break no rule",
			specs: gohai.Collectors(),
		},
		{
			name: "when two collectors have one name",
			specs: []gohai.CollectorSpec{
				{Name: "a", Category: "system", Fields: fields},
				{Name: "a", Category: "system", Fields: fields},
			},
			wantRules: []string{gohai.RuleName},
		},
		{
			name: "when a name is not lowercase letters, digits and underscores",
			specs: []gohai.CollectorSpec{
				{Name: "Upper", Category: "system", Fields: fields},
				{Name: "9lives", Category: "system", Fields: fields},
				{Name: "has-dash", Category: "system", Fields: fields},
				{Name: "", Category: "system", Fields: fields},
			},
			wantRules: []string{gohai.RuleName, gohai.RuleName, gohai.RuleName, gohai.RuleName},
		},
		{
			name:      "when the category is not one of the categories",
			specs:     []gohai.CollectorSpec{{Name: "a", Category: "kitchen", Fields: fields}},
			wantRules: []string{gohai.RuleCategory},
		},
		{
			name:      "when gohai.Facts has no field for it",
			specs:     []gohai.CollectorSpec{{Name: "a", Category: "system"}},
			wantRules: []string{gohai.RuleFacts},
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
			wantRules: []string{gohai.RuleDependency, gohai.RuleDependency},
		},
	}

	seen := map[string]bool{}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			var rules []string
			for _, p := range gohai.CollectorProblems(tc.specs) {
				rules = append(rules, p.Rule)
				seen[p.Rule] = true
				s.NotEmpty(p.Message)
				s.NotEmpty(p.Fix, "every rule has a fix")
			}
			s.Equal(tc.wantRules, rules)
		})
	}

	// Each check is a rule that CollectorProblems applies, and each rule has
	// a check: the cases above trigger every one.
	for _, c := range gohai.CollectorChecks() {
		s.NotEmpty(c.Requires, c.Rule)
		s.True(seen[c.Rule], "no case triggers the rule %s", c.Rule)
	}
}

func (s *SpecPublicTestSuite) TestCollectorTerms() {
	terms := gohai.CollectorTerms()
	s.NotEmpty(terms)
	for _, t := range terms {
		s.NotEmpty(t.Name)
		s.NotEmpty(t.Does, t.Name)
		s.NotEmpty(t.When, t.Name)
	}
}

func (s *SpecPublicTestSuite) TestJSONName() {
	type t struct {
		Tagged   string `json:"tagged,omitempty"`
		Untagged string
	}
	rt := reflect.TypeFor[t]()
	s.Equal("tagged", gohai.JSONName(rt.Field(0)))
	s.Equal("Untagged", gohai.JSONName(rt.Field(1)), "a field with no tag has its Go name")
}

func TestSpecPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SpecPublicTestSuite))
}
