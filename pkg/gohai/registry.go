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

// PublicRegistry exposes registered collector metadata for CLI use.
type PublicRegistry struct {
	reg *collector.Registry
}

// NewRegistry returns a PublicRegistry pre-populated with built-in
// collectors, or the error of a built-in collector that does not register.
func NewRegistry() (*PublicRegistry, error) {
	r := collector.NewRegistry()
	if err := registerBuiltins(r, source.Sources{}); err != nil {
		return nil, err
	}

	return &PublicRegistry{reg: r}, nil
}

// Names returns all registered collector names.
func (p *PublicRegistry) Names() []string {
	return p.reg.Names()
}

// NamesInCategory returns the names of every collector in cat. Empty
// slice when the category has no collectors. Callers should treat
// that as a user-facing "unknown category" error.
func (p *PublicRegistry) NamesInCategory(
	cat string,
) []string {
	return p.reg.NamesInCategory(collector.Category(cat))
}

// CategoryOf returns the collector's category, or empty string when
// no collector is registered under that name. Intended for CLI help
// and `--list-collectors`-style output that groups by category.
func (p *PublicRegistry) CategoryOf(
	name string,
) string {
	c, ok := p.reg.Get(name)
	if !ok {
		return ""
	}
	return string(c.Category())
}
