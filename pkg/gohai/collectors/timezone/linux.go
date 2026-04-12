//go:build linux

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

// readFileFn reads a file's contents. Swappable for tests.
var readFileFn = os.ReadFile

// nowFn returns the current time. Swappable for tests.
var nowFn = time.Now

const (
	localtimePath  = "/etc/localtime"
	timezonePath   = "/etc/timezone"
	zoneinfoPrefix = "/usr/share/zoneinfo/"
)

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFuncs(readlinkFn, readFileFn, nowFn), nil
}

// collectFromFuncs assembles Info from a readlink function, a file-reader,
// and a clock. Never returns an error — any missing source leaves the
// affected field empty while the others still populate.
//
// Name resolution order:
//  1. /etc/localtime symlink target (systemd-style hosts)
//  2. /etc/timezone file contents (Debian/Ubuntu style, or container
//     images that copy zoneinfo files instead of symlinking)
func collectFromFuncs(
	readlink func(string) (string, error),
	readFile func(string) ([]byte, error),
	now func() time.Time,
) *Info {
	info := &Info{}
	abbrev, offset := now().Zone()
	info.Abbrev = abbrev
	info.Offset = offset
	info.Name = resolveName(readlink, readFile)
	return info
}

func resolveName(
	readlink func(string) (string, error),
	readFile func(string) ([]byte, error),
) string {
	if target, err := readlink(localtimePath); err == nil {
		return strings.TrimPrefix(target, zoneinfoPrefix)
	}
	if b, err := readFile(timezonePath); err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}
