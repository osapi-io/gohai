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

package timezone

import (
	"context"
	"os"
	"strings"
	"time"
)

// readlinkFn reads the target of a symbolic link. Swappable for tests.
var readlinkFn = os.Readlink

// nowFn returns the current time. Swappable for tests.
var nowFn = time.Now

const (
	localtimePath  = "/etc/localtime"
	zoneinfoPrefix = "/var/db/timezone/zoneinfo/"
)

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFuncs(readlinkFn, nowFn), nil
}

// collectFromFuncs assembles Info from a readlink function and a clock.
// Never returns an error: a missing /etc/localtime leaves Name empty while
// Abbrev/Offset still come from the clock.
func collectFromFuncs(
	readlink func(string) (string, error),
	now func() time.Time,
) *Info {
	info := &Info{}
	abbrev, offset := now().Zone()
	info.Abbrev = abbrev
	info.Offset = offset
	if target, err := readlink(localtimePath); err == nil {
		info.Name = strings.TrimPrefix(target, zoneinfoPrefix)
	}
	return info
}
