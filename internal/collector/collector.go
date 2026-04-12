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

// Package collector defines the Collector interface and tier classifications
// for gohai's pluggable fact collection system.
package collector

import "context"

// Tier classifies a collector for default enable/disable behavior.
type Tier int

const (
	// TierCore collectors are foundational and enabled by default.
	TierCore Tier = iota + 1
	// TierExtended collectors are enabled by default but cover broader topics.
	TierExtended
	// TierOptIn collectors are disabled by default and require explicit opt-in.
	TierOptIn
)

// String returns the human-readable name of the tier.
func (t Tier) String() string {
	switch t {
	case TierCore:
		return "core"
	case TierExtended:
		return "extended"
	case TierOptIn:
		return "opt-in"
	default:
		return "unknown"
	}
}

// EnabledByDefault reports whether collectors of this tier run by default.
func (t Tier) EnabledByDefault() bool {
	return t == TierCore || t == TierExtended
}

// Collector is the interface every fact collector must implement.
type Collector interface {
	// Name returns the unique identifier (e.g., "platform").
	Name() string

	// Tier returns the tier classification.
	Tier() Tier

	// Dependencies returns names of collectors that must run before this one.
	Dependencies() []string

	// Collect gathers facts and returns a typed struct (or nil if not supported).
	Collect(
		ctx context.Context,
	) (any, error)
}
