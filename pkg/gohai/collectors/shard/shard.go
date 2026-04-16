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

// Package shard derives a deterministic shard seed from stable host
// identity. Matches Ohai's shard plugin: concatenate machinename +
// DMI serial + DMI uuid, MD5 hash, take the first 7 hex chars as an
// integer. The same host always maps to the same bucket.
package shard

import (
	//nolint:gosec // MD5 is not used for security — it's Ohai's hash for shard distribution.
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

// Info holds the derived shard seed.
type Info struct {
	Seed int `json:"seed"`
}

// Collector is the public interface every shard variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string     { return "shard" }
func (base) Category() string { return collector.CategorySystem }

func (base) DefaultEnabled() bool { return true }

func (base) Dependencies() []string { return []string{"hostname", "dmi"} }

// New returns the shard variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// getMachineName extracts the raw hostname from the hostname prior.
func getMachineName(
	prior collector.PriorResults,
) string {
	info, ok := collector.GetDep[*hostname.Info](prior, "hostname")
	if !ok || info == nil {
		return ""
	}
	return info.MachineName
}

// getDMISerial mirrors Ohai's get_dmi_property for serial_number —
// cascades system → baseboard → chassis, takes first non-blank.
func getDMISerial(
	info *dmi.Info,
) string {
	if info == nil {
		return ""
	}
	for _, s := range []string{
		prodSerial(info.Product),
		bbSerial(info.Baseboard),
		chSerial(info.Chassis),
	} {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// getDMIUUID mirrors Ohai's get_dmi_property for uuid — only
// Product carries it in sysfs.
func getDMIUUID(
	info *dmi.Info,
) string {
	if info == nil || info.Product == nil {
		return ""
	}
	return info.Product.UUID
}

func prodSerial(p *dmi.Product) string {
	if p == nil {
		return ""
	}
	return p.SerialNumber
}

func bbSerial(b *dmi.Baseboard) string {
	if b == nil {
		return ""
	}
	return b.SerialNumber
}

func chSerial(c *dmi.Chassis) string {
	if c == nil {
		return ""
	}
	return c.SerialNumber
}

// computeSeed matches Ohai's shard algorithm: concatenate sources
// (no separator), MD5 hash, first 7 hex chars interpreted as base-16
// integer.
func computeSeed(
	machinename, serial, uuid string,
) int {
	data := machinename + serial + uuid
	//nolint:gosec // MD5 is Ohai's spec — not a security context.
	h := md5.Sum([]byte(data))
	hex := fmt.Sprintf("%x", h)
	val, _ := strconv.ParseInt(hex[:7], 16, 64)
	return int(val)
}
