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

// Register exposes the private registry to external tests so a test
// can inject a custom Collector (e.g., one that always errors) to
// exercise failure-path behaviour without depending on a real
// collector's upstream seam.
func (g *Gohai) Register(
	c collector.Collector,
) error {
	return g.registry.Register(c)
}

// Select appends the given collector (by name) to the list of
// collectors Collect will run. Pair with Register above.
func (g *Gohai) Select(
	name string,
) error {
	c, ok := g.registry.Get(name)
	if !ok {
		return UnknownCollectorError{Name: name}
	}
	g.selected = append(g.selected, c)
	return nil
}

// UnknownCollectorError is returned by Select when the named
// collector isn't registered. It is test-only, so it lives in export_test.go.
type UnknownCollectorError struct{ Name string }

func (e UnknownCollectorError) Error() string {
	return "unknown collector " + e.Name
}

var JSONName = jsonName

// BuiltinCollectors exposes the built-in collectors, built from src, so a
// test can check that each one takes src.
var BuiltinCollectors = builtinCollectors

// RegisterBuiltins exposes registerBuiltins, so a test can register the
// built-in collectors twice.
var RegisterBuiltins = registerBuiltins

// Set exposes Facts.set, so a test can check that each built-in collector's
// result reaches its field.
func (f *Facts) Set(
	name string,
	result any,
) {
	f.set(name, result)
}

// SetBuiltinCollectorsFn replaces the built-in collectors that New and
// NewRegistry register, and returns a func that puts them back.
func SetBuiltinCollectorsFn(
	fn func(source.Sources) []collector.Collector,
) func() {
	old := builtinCollectorsFn
	builtinCollectorsFn = fn

	return func() { builtinCollectorsFn = old }
}
