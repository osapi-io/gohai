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
	"os/user"
)

// Linux resolves the root group on generic Linux hosts. Typically
// returns "root". Embeds base for Name/DefaultEnabled/Dependencies.
type Linux struct {
	base

	LookupUserFn  func(string) (*user.User, error)
	LookupGroupFn func(string) (*user.Group, error)
}

// NewLinux returns a Linux variant wired to os/user.
func NewLinux() *Linux {
	return &Linux{
		LookupUserFn:  user.Lookup,
		LookupGroupFn: user.LookupGroupId,
	}
}

// Collect performs the two-hop lookup (root → gid → group name).
func (l *Linux) Collect(_ context.Context) (any, error) {
	return resolveRootGroup(l.LookupUserFn, l.LookupGroupFn)
}
