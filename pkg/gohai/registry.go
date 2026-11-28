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
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Registry describes the built-in collectors: their names and categories.
// The CLI reads it to list the collectors and to build its flags.
type Registry interface {
	// Names returns the name of each collector, sorted.
	Names() []string
	// NamesInCategory returns the names, sorted, of each collector in cat,
	// or an empty slice when no collector is in cat.
	NamesInCategory(cat string) []string
	// CategoryOf returns the category of the collector called name, or ""
	// when no collector has that name.
	CategoryOf(name string) string
}

// registry is the Registry of NewRegistry.
type registry struct {
	reg *collector.Registry
}

// NewRegistry returns a Registry of the built-in collectors, or the error of
// a built-in collector that does not register.
func NewRegistry() (Registry, error) {
	r := collector.NewRegistry()
	if err := registerBuiltins(r, source.Sources{}); err != nil {
		return nil, err
	}

	return &registry{reg: r}, nil
}

// Names returns the name of each collector, sorted.
func (p *registry) Names() []string {
	return p.reg.Names()
}

// NamesInCategory returns the names, sorted, of each collector in cat, or an
// empty slice when no collector is in cat.
func (p *registry) NamesInCategory(
	cat string,
) []string {
	return p.reg.NamesInCategory(collector.Category(cat))
}

// CategoryOf returns the category of the collector called name, or "" when
// no collector has that name.
func (p *registry) CategoryOf(
	name string,
) string {
	c, ok := p.reg.Get(name)
	if !ok {
		return ""
	}

	return string(c.Category())
}
