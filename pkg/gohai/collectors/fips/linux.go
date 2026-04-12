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

const (
	fipsPath   = "/proc/sys/crypto/fips_enabled"
	policyPath = "/etc/crypto-policies/config"
)

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
	info := &Info{}

	kernel, err := readKernelFlag(open)
	if err != nil {
		return nil, err
	}
	info.Kernel = kernel

	info.Policy = readCryptoPolicy(open)

	return info, nil
}

func readKernelFlag(
	open func(string) (io.ReadCloser, error),
) (Kernel, error) {
	rc, err := open(fipsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Kernel{}, nil
		}
		return Kernel{}, fmt.Errorf("open %s: %w", fipsPath, err)
	}
	defer func() { _ = rc.Close() }()
	return parseFips(rc)
}

func parseFips(
	r io.Reader,
) (Kernel, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return Kernel{}, fmt.Errorf("read fips: %w", err)
	}
	return Kernel{Enabled: strings.TrimSpace(string(b)) == "1"}, nil
}

// readCryptoPolicy returns nil on hosts without /etc/crypto-policies
// (Debian/Ubuntu, older RHEL, Alpine, etc.). A missing file is not an
// error — it just means the distro doesn't ship the crypto-policies
// framework. Read errors other than not-exist also return nil; we'd
// rather omit the field than fail the whole collector.
func readCryptoPolicy(
	open func(string) (io.ReadCloser, error),
) *Policy {
	rc, err := open(policyPath)
	if err != nil {
		return nil
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil
	}
	return parsePolicy(string(b))
}

func parsePolicy(
	raw string,
) *Policy {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return &Policy{
			Name:          line,
			FIPSEffective: strings.HasPrefix(line, "FIPS"),
		}
	}
	return nil
}
