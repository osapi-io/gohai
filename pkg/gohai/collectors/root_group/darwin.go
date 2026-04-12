//go:build darwin

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

package rootgroup

import (
	"context"
	"fmt"
	"os/user"
)

// lookupUserFn looks up a user by name. Swappable for tests.
var lookupUserFn = user.Lookup

// lookupGroupFn looks up a group by GID. Swappable for tests.
var lookupGroupFn = user.LookupGroupId

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFuncs(lookupUserFn, lookupGroupFn)
}

// collectFromFuncs resolves the primary group name for the root user via
// a two-hop lookup: user "root" → its primary GID → group name. On
// macOS/BSD the answer is typically "wheel". Matches Ohai's defensive
// resolution and is correct even if root's primary GID isn't 0.
func collectFromFuncs(
	lookupUser func(string) (*user.User, error),
	lookupGroup func(string) (*user.Group, error),
) (*Info, error) {
	u, err := lookupUser("root")
	if err != nil {
		return nil, fmt.Errorf("lookup user root: %w", err)
	}
	g, err := lookupGroup(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("lookup gid %s: %w", u.Gid, err)
	}
	return &Info{Name: g.Name}, nil
}
