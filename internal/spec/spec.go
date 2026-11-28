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

// Package spec checks the descriptions of collectors that gohai.Collectors
// returns against the rules of a collector. gohai collector validate prints
// the rules and the problems.
package spec

import (
	"slices"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai"
)

// The rules that Problems checks, by the name a problem reports.
const (
	RuleName       = "name"
	RuleCategory   = "category"
	RuleFacts      = "facts"
	RuleDependency = "dependency"
)

// Check is one rule: what a collector must do, and how to fix it.
type Check struct {
	Rule     string `json:"rule"`
	Requires string `json:"requires"`
	Fix      string `json:"fix"`
}

// Checks are the rules of a collector, which gohai collector
// validate prints and checks.
func Checks() []Check {
	return []Check{
		{
			Rule:     RuleName,
			Requires: "Each collector has a name of lowercase letters, digits and underscores, and no two collectors have one name.",
			Fix:      "Name the collector for the facts it returns, such as kernel_modules, in its Name method.",
		},
		{
			Rule:     RuleCategory,
			Requires: "Each collector's category is one of the categories of internal/collector.",
			Fix:      "Return one of the collector.Category constants from Category, or add the category there and to docs/collectors/README.md.",
		},
		{
			Rule:     RuleFacts,
			Requires: "Each collector has a field of gohai.Facts whose JSON name is its name, so the schema and the docs have its facts.",
			Fix:      "Add the field to Facts in pkg/gohai/facts.go, then run go generate ./schemas/... to write the schema.",
		},
		{
			Rule:     RuleDependency,
			Requires: "Each dependency of a collector is another registered collector.",
			Fix:      "Name a collector that gohai collector list shows, or remove the dependency.",
		},
	}
}

// Problem is a rule that a collector breaks.
type Problem struct {
	Collector string `json:"collector"`
	Rule      string `json:"rule"`
	Message   string `json:"message"`
	Fix       string `json:"fix"`
}

// Problems are the rules that the collectors of specs break.
func Problems(
	specs []gohai.CollectorSpec,
) []Problem {
	fix := map[string]string{}
	for _, c := range Checks() {
		fix[c.Rule] = c.Fix
	}
	var out []Problem
	add := func(
		name string,
		rule string,
		message string,
	) {
		out = append(out, Problem{Collector: name, Rule: rule, Message: message, Fix: fix[rule]})
	}
	names := map[string]bool{}
	for _, c := range specs {
		if names[c.Name] {
			add(c.Name, RuleName, "two collectors have the name "+c.Name+".")
		}
		names[c.Name] = true
	}
	for _, c := range specs {
		if !validName(c.Name) {
			add(
				c.Name,
				RuleName,
				"the name "+c.Name+" is not lowercase letters, digits and underscores.",
			)
		}
		if !slices.Contains(collector.Categories(), collector.Category(c.Category)) {
			add(c.Name, RuleCategory, "the category "+c.Category+" is not one of the categories.")
		}
		if c.Fields == nil {
			add(c.Name, RuleFacts, "gohai.Facts has no field named "+c.Name+".")
		}
		for _, d := range c.Dependencies {
			if d == c.Name || !names[d] {
				add(
					c.Name,
					RuleDependency,
					"the dependency "+d+" is not another registered collector.",
				)
			}
		}
	}

	return out
}

// validName reports whether name is lowercase letters, digits and
// underscores, starting with a letter.
func validName(
	name string,
) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}

	return true
}
