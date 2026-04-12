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

package users

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/host"
)

var usersFn = host.UsersWithContext

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, usersFn)
}

func collectFromGopsutil(
	ctx context.Context,
	fn func(context.Context) ([]host.UserStat, error),
) (any, error) {
	us, err := fn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Users: %w", err)
	}
	sessions := make([]Session, 0, len(us))
	for _, u := range us {
		sessions = append(sessions, Session{
			User:     u.User,
			Terminal: u.Terminal,
			Host:     u.Host,
			Started:  uint64(u.Started),
		})
	}
	return &Info{LoggedIn: sessions}, nil
}
