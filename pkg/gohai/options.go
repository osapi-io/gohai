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

// Option configures a Gohai instance.
type Option func(*config)

type config struct {
	useDefaults bool
	enabled     []string
	disabled    []string
	only        []string
	categories  []string
	withTimings bool
}

// WithDefaults opts in to the recommended default collector set —
// every collector whose DefaultEnabled() is true. Without this option
// (and without WithCollectors / WithEnabled), gohai.New() returns an
// empty registry and Collect() runs nothing. The CLI wires this in at
// the command layer; SDK consumers opt in explicitly.
func WithDefaults() Option {
	return func(c *config) { c.useDefaults = true }
}

// WithEnabled explicitly enables one or more collectors on top of
// whatever WithDefaults selects (or in isolation when WithDefaults
// isn't passed).
func WithEnabled(
	names ...string,
) Option {
	return func(c *config) { c.enabled = append(c.enabled, names...) }
}

// WithDisabled explicitly disables one or more collectors. Subtracts
// from the WithDefaults / WithEnabled set.
func WithDisabled(
	names ...string,
) Option {
	return func(c *config) { c.disabled = append(c.disabled, names...) }
}

// WithCollectors restricts collection to ONLY the named collectors and
// their dependencies. Overrides any default-set selection.
func WithCollectors(
	names ...string,
) Option {
	return func(c *config) { c.only = append(c.only, names...) }
}

// WithCategory enables every collector whose Category() matches one
// of the given names ("cloud", "hardware", "system", ...). Stacks
// with WithEnabled / WithDefaults — categories add collectors,
// WithDisabled still subtracts. Unknown category names error at
// New() time so typos surface immediately.
func WithCategory(
	names ...string,
) Option {
	return func(c *config) { c.categories = append(c.categories, names...) }
}

// WithTimings embeds per-collector timing + error data into Facts
// under a top-level `_timings` object. Each entry records the
// collector's wall-clock duration, status (`ok` / `err`), and — for
// failed collectors — the error string. Failed collectors are still
// dropped from the typed output; their timing entry is how the
// failure surfaces when this option is on. Disabled by default —
// when off, Facts contain only collector results.
func WithTimings() Option {
	return func(c *config) { c.withTimings = true }
}
