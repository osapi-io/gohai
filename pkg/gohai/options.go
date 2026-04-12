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
	enabled  []string
	disabled []string
	only     []string
}

// WithEnabled explicitly enables one or more opt-in collectors.
func WithEnabled(
	names ...string,
) Option {
	return func(c *config) { c.enabled = append(c.enabled, names...) }
}

// WithDisabled explicitly disables one or more collectors.
func WithDisabled(
	names ...string,
) Option {
	return func(c *config) { c.disabled = append(c.disabled, names...) }
}

// WithCollectors restricts collection to ONLY the named collectors and their
// dependencies. Overrides the default tier-based selection.
func WithCollectors(
	names ...string,
) Option {
	return func(c *config) { c.only = append(c.only, names...) }
}
