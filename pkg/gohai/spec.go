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
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/readlog"
	"github.com/osapi-io/gohai/pkg/gohai/family"
)

// CollectorSpec is what one collector is: what it returns, how a caller
// builds it on a host of its own, and what it reads from the host. A
// program that reads a host through gohai, such as osapi, gets its facts
// from these collectors, and a fake host of its tests gives each one what
// it reads.
type CollectorSpec struct {
	// Name is the collector and the key of its facts, such as services.
	Name string `json:"name"`
	// Category is the group it belongs to, such as software.
	Category string `json:"category"`
	// DefaultEnabled is true when it runs with no --collector flag.
	DefaultEnabled bool `json:"default_enabled"`
	// Dependencies are the collectors that run before it.
	Dependencies []string `json:"dependencies"`
	// Package is its Go package.
	Package string `json:"package"`
	// Constructor builds it on a host of the caller's, such as
	// services.NewWith(src), with src a source.Sources.
	Constructor string `json:"constructor"`
	// Fields are the fields of its facts.
	Fields []Field `json:"fields"`
	// Reads are what it reads from a host of each platform.
	Reads []Reads `json:"reads"`
	// Network is what it reads over the network, outside source.Sources,
	// such as the metadata service of a cloud. Empty for a collector that
	// reads only the host.
	Network string `json:"network,omitempty"`
}

// Field is one field of a collector's facts.
type Field struct {
	// Name is the field's JSON name.
	Name string `json:"name"`
	// Type is its Go type.
	Type string `json:"type"`
}

// Reads are what a collector reads from a host of one platform, when the
// host has none of it: the files it opens or looks at and the commands it
// runs, before anything that it reads tells it to read more. With content,
// it also reads under these paths, such as each device under
// /sys/class/net.
type Reads struct {
	// Platform is the family, as family.Detect names it, such as debian.
	Platform string `json:"platform"`
	// Files are the files and directories.
	Files []string `json:"files"`
	// Commands are the commands, each with its arguments.
	Commands []string `json:"commands"`
}

// SpecPlatforms are the platforms whose reads a CollectorSpec lists.
func SpecPlatforms() []string {
	return []string{family.Debian, family.RHEL, family.Darwin}
}

// Collectors are the built-in collectors, in the order that gohai runs
// them. The reads come from running each collector on an empty host of each
// platform, so they are always what the code does.
func Collectors() []CollectorSpec {
	fields := factFields()
	probe, _ := readlog.Sources(family.Debian)
	out := make([]CollectorSpec, 0, len(builtinCollectors(probe)))
	for i, c := range builtinCollectors(probe) {
		pkg := packageOf(c)
		spec := CollectorSpec{
			Name:           c.Name(),
			Category:       c.Category(),
			DefaultEnabled: c.DefaultEnabled(),
			Dependencies:   append([]string{}, c.Dependencies()...),
			Package:        pkg,
			Constructor:    path.Base(pkg) + ".NewWith(src)",
			Fields:         fields[c.Name()],
		}
		if c.Category() == collector.CategoryCloud {
			// A cloud collector reads its provider's metadata service, not
			// the host, so a run would only wait for the network.
			spec.Constructor = path.Base(pkg) + ".New()"
			spec.Network = "the metadata service of " + c.Name() + ", over HTTP"
			for _, platform := range SpecPlatforms() {
				spec.Reads = append(
					spec.Reads,
					Reads{Platform: platform, Files: []string{}, Commands: []string{}},
				)
			}
			out = append(out, spec)

			continue
		}
		for _, platform := range SpecPlatforms() {
			src, log := readlog.Sources(platform)
			_, _ = builtinCollectors(src)[i].Collect(context.Background(), collector.PriorResults{})
			spec.Reads = append(spec.Reads, Reads{
				Platform: platform,
				Files:    log.Files(),
				Commands: log.Commands(),
			})
		}
		out = append(out, spec)
	}

	return out
}

// CollectorByName is the collector with the name, and whether there is one.
func CollectorByName(
	name string,
) (CollectorSpec, bool) {
	for _, c := range Collectors() {
		if c.Name == name {
			return c, true
		}
	}

	return CollectorSpec{}, false
}

// packageOf is the Go package of c's type.
func packageOf(
	c collector.Collector,
) string {
	t := reflect.TypeOf(c)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.PkgPath()
}

// factFields are the fields of each collector's facts, by the collector's
// name: the fields of the type of the Facts field whose JSON name it is.
func factFields() map[string][]Field {
	out := map[string][]Field{}
	ft := reflect.TypeFor[Facts]()
	for i := range ft.NumField() {
		f := ft.Field(i)
		name := jsonName(f)
		t := f.Type
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || name == "" || name == "-" {
			continue
		}
		var fields []Field
		for j := range t.NumField() {
			sf := t.Field(j)
			if !sf.IsExported() || jsonName(sf) == "-" {
				continue
			}
			fields = append(fields, Field{Name: jsonName(sf), Type: sf.Type.String()})
		}
		out[name] = fields
	}

	return out
}

// jsonName is the JSON name of f.
func jsonName(
	f reflect.StructField,
) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "" {
		return f.Name
	}

	return name
}

// The rules that CollectorProblems checks, by the name a problem reports.
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

// CollectorChecks are the rules of a collector, which gohai collector
// validate prints and checks.
func CollectorChecks() []Check {
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

// CollectorProblems are the rules that the collectors of specs break.
func CollectorProblems(
	specs []CollectorSpec,
) []Problem {
	fix := map[string]string{}
	for _, c := range CollectorChecks() {
		fix[c.Rule] = c.Fix
	}
	var out []Problem
	add := func(name, rule, message string) {
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
		if !slices.Contains(collector.Categories(), c.Category) {
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

// Term is one part of writing a collector: its name, what it does, and when
// to use it.
type Term struct {
	Name string `json:"name"`
	Does string `json:"does"`
	When string `json:"when"`
}

// CollectorTerms are the parts of a collector, which gohai collector spec
// get prints. Read them before you write a collector, then check the
// result with gohai collector validate.
func CollectorTerms() []Term {
	return []Term{
		{
			Name: "Name() string",
			Does: "Names the collector and the key of its facts, such as kernel_modules.",
			When: "Always. It is also the JSON name of its field of gohai.Facts.",
		},
		{
			Name: "Category() string",
			Does: "Puts the collector in a group, one of the collector.Category constants, that --category enables at once.",
			When: "Always.",
		},
		{
			Name: "DefaultEnabled() bool",
			Does: "Runs the collector with no --collector flag when true.",
			When: "False for a collector that is slow, needs privilege, or returns a lot, such as the full package list.",
		},
		{
			Name: "Dependencies() []string",
			Does: "Runs the named collectors first. Collect reads their facts with collector.GetDep.",
			When: "When the collector builds on the facts of another, such as a cloud collector on dmi.",
		},
		{
			Name: "Collect(ctx, prior) (any, error)",
			Does: "Reads the host and returns the collector's typed facts, or nil when the host does not have them.",
			When: "Always. Return an error only for a failure, not for a fact that the host does not have.",
		},
		{
			Name: "NewWith(src source.Sources)",
			Does: "Builds the collector on a host of the caller's: its files, its commands, its clock, its family and its resolver. collector.Wire puts each source in the field of its type.",
			When: "For each collector that reads the host. A test, or a program such as osapi, gives it a fake host. Read every file and run every command through src, never through os or os/exec.",
		},
		{
			Name: "New()",
			Does: "Builds the collector on the real host.",
			When: "For each collector. A cloud collector, which reads its provider's metadata service over HTTP, has only New.",
		},
		{
			Name: "Facts field",
			Does: "A field of gohai.Facts in pkg/gohai/facts.go whose JSON name is the collector's name. go generate ./schemas/... writes the JSON Schema from it.",
			When: "For each collector.",
		},
		{
			Name: "builtinCollectors",
			Does: "Registers the collector, in pkg/gohai/gohai.go, built with NewWith(src).",
			When: "For each built-in collector. gohai collector list then shows it, and gohai collector get shows what it reads on each platform.",
		},
	}
}
