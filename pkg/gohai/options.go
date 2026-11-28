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

import "github.com/osapi-io/gohai/pkg/gohai/source"

// Option configures a Gohai instance.
type Option func(*config)

type config struct {
	useDefaults bool
	enabled     []string
	disabled    []string
	only        []string
	categories  []string
	withTimings bool
	sources     source.Sources
}

// WithSources makes each collector read the host through src: its
// filesystem, its commands, its clock, its family, its resolver and the
// transport of the cloud collectors' metadata reads. A zero field is the
// real host's, except a zero family with a given filesystem, which is the
// family of that filesystem's /etc/os-release. Without it, collectors read
// the real host. A caller with a fake host, such as osapi in its tests, gets
// the same collectors reading that host, and its src.HTTP answers the
// metadata services, so no test reads the network.
func WithSources(
	src source.Sources,
) Option {
	return func(c *config) {
		c.sources = src
	}
}

// WithDefaults opts in to the recommended default collector set:
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
// from the WithDefaults / WithEnabled set. New refuses it with
// WithCollectors.
func WithDisabled(
	names ...string,
) Option {
	return func(c *config) { c.disabled = append(c.disabled, names...) }
}

// WithCollectors restricts collection to ONLY the named collectors and
// their dependencies. Overrides any default-set selection. New refuses it
// with WithDisabled.
func WithCollectors(
	names ...string,
) Option {
	return func(c *config) { c.only = append(c.only, names...) }
}

// WithCategory enables every collector whose Category() matches one
// of the given names ("cloud", "hardware", "system", ...). Stacks
// with WithEnabled / WithDefaults: categories add collectors, and
// WithDisabled still subtracts. Unknown category names error at
// New() time so typos surface immediately.
func WithCategory(
	names ...string,
) Option {
	return func(c *config) { c.categories = append(c.categories, names...) }
}

// WithTimings embeds per-collector timing data into Facts under a
// top-level `_timings` object. Each entry records the collector's
// wall-clock duration, its status (ok, absent, unsupported, failed or
// partial), and the error string of a collector that has one. A failed
// collector has no fact, and a partial one keeps the fact it read. It is
// off by default, and Facts.Status and Facts.Errors hold the status and
// the errors either way.
func WithTimings() Option {
	return func(c *config) { c.withTimings = true }
}
