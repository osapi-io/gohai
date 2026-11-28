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
	"sync"
	"sync/atomic"

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
	// Network is what it reads over the network, through source.Sources'
	// HTTP, such as the metadata service of a cloud. Empty for a collector
	// that reads only the host.
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
// platform, so they are always what the code does. The list is built once,
// on the first call, and each call returns its own deep copy, so a caller
// that changes a result does not change the next one.
func Collectors() []CollectorSpec {
	specs := collectorSpecs()
	out := make([]CollectorSpec, len(specs))
	for i, c := range specs {
		out[i] = c.clone()
	}

	return out
}

// clone is c with copies of its slices.
func (c CollectorSpec) clone() CollectorSpec {
	c.Dependencies = slices.Clone(c.Dependencies)
	c.Fields = slices.Clone(c.Fields)
	reads := make([]Reads, len(c.Reads))
	for i, r := range c.Reads {
		r.Files = slices.Clone(r.Files)
		r.Commands = slices.Clone(r.Commands)
		reads[i] = r
	}
	c.Reads = reads

	return c
}

// collectorSpecs builds the list of Collectors once, since each build runs
// every collector on three platforms.
var collectorSpecs = sync.OnceValue(buildCollectorSpecs)

// specBuilds counts the builds of the list of Collectors, so that a test
// can see that it is built once.
var specBuilds atomic.Int32

// buildCollectorSpecs is the list of Collectors.
func buildCollectorSpecs() []CollectorSpec {
	specBuilds.Add(1)
	fields := factFields()
	probe, _ := readlog.Sources(family.Debian)
	out := make([]CollectorSpec, 0, len(builtinCollectors(probe)))
	for i, c := range builtinCollectors(probe) {
		pkg := packageOf(c)
		spec := CollectorSpec{
			Name:           c.Name(),
			Category:       string(c.Category()),
			DefaultEnabled: c.DefaultEnabled(),
			Dependencies:   append([]string{}, c.Dependencies()...),
			Package:        pkg,
			Constructor:    path.Base(pkg) + ".NewWith(src)",
			Fields:         fields[c.Name()],
		}
		if collector.HasHTTP(c) {
			spec.Network = "the metadata service of " + c.Name() + ", over HTTP"
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
		if t.Kind() != reflect.Struct || name == "-" {
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
			Name: "Category() collector.Category",
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
			Does: "Builds the collector on a host of the caller's: its files, its commands, its clock, its family, its resolver and, for a cloud collector, the transport of its metadata reads. collector.Wire puts each source in the field of its type.",
			When: "For each collector. A test, or a program such as osapi, gives it a fake host. Read every file and run every command through src, never through os or os/exec.",
		},
		{
			Name: "New()",
			Does: "Builds the collector on the real host.",
			When: "For each collector.",
		},
		{
			Name: "Facts field",
			Does: "A field of gohai.Facts in pkg/gohai/facts.go whose JSON name is the collector's name. just generate writes the collector list, Facts.set and the JSON Schema from it.",
			When: "For each collector.",
		},
		{
			Name: "builtinCollectors",
			Does: "Registers the collector, built with NewWith(src), in pkg/gohai/builtins.gen.go, which go generate writes from Facts. Do not edit it by hand.",
			When: "For each built-in collector. gohai collector list then shows it, and gohai collector get shows what it reads on each platform.",
		},
	}
}
