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

package fips

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const fipsPath = "/proc/sys/crypto/fips_enabled"

var openFn = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func collect(
	_ context.Context,
) (any, error) {
	return collectFromFunc(openFn)
}

func collectFromFunc(
	open func(string) (io.ReadCloser, error),
) (*Info, error) {
	rc, err := open(fipsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Info{}, nil
		}
		return nil, fmt.Errorf("open %s: %w", fipsPath, err)
	}
	defer func() { _ = rc.Close() }()
	return parseFips(rc)
}

func parseFips(
	r io.Reader,
) (*Info, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read fips: %w", err)
	}
	return &Info{Kernel: Kernel{Enabled: strings.TrimSpace(string(b)) == "1"}}, nil
}
