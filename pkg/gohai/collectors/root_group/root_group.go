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

// Package rootgroup reports the primary group name for the root user.
package rootgroup

import (
	"fmt"
	"os/user"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// lookupUserFn and lookupGroupFn are the package-level seams wrapping
// os/user. Tests swap them via SetLookupUserFn / SetLookupGroupFn.
var (
	lookupUserFn  = user.Lookup
	lookupGroupFn = user.LookupGroupId
)

// Info holds the root group data.
type Info struct {
	Name string `json:"name"` // primary group name for root — typically "root" on Linux, "wheel" on macOS/BSD
}

// Collector is the public interface every rootgroup variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields common to every OS variant.
type base struct{}

// Name returns "root_group".
func (base) Name() string { return "root_group" }

// DefaultEnabled returns true — root_group is on by default.
func (base) DefaultEnabled() bool { return true }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the root_group collector variant appropriate to the host.
// os/user-backed lookups work identically on Linux and macOS, but the
// expected result differs (Linux → "root", macOS → "wheel").
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// resolveRootGroup performs the two-hop lookup Ohai uses: look up user
// "root" to get its primary GID, then look up that GID to get the group
// name. Correct even on systems where root's primary GID isn't 0.
// Shared by both Linux and Darwin variants.
func resolveRootGroup() (*Info, error) {
	u, err := lookupUserFn("root")
	if err != nil {
		return nil, fmt.Errorf("lookup user root: %w", err)
	}
	g, err := lookupGroupFn(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("lookup gid %s: %w", u.Gid, err)
	}
	return &Info{Name: g.Name}, nil
}
