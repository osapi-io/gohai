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

// lookupFn looks up a group by GID. Swappable for tests.
var lookupFn = user.LookupGroupId

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFunc(lookupFn)
}

// collectFromFunc resolves the group name for GID 0 via the supplied lookup
// function. Returns an error if the lookup fails.
func collectFromFunc(
	lookup func(string) (*user.Group, error),
) (*Info, error) {
	g, err := lookup("0")
	if err != nil {
		return nil, fmt.Errorf("lookup gid 0: %w", err)
	}
	return &Info{Name: g.Name}, nil
}
