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
// identity: concatenate machinename + DMI serial + DMI uuid, hash it
// with SHA-256, and take the first 7 hex chars as an integer. The same
// host always maps to the same bucket. Ohai's shard plugin uses MD5,
// which is not a FIPS 140-3 approved algorithm, so the seeds differ
// from Ohai's.
package shard

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds the derived shard seed.
type Info struct {
	Seed int `json:"seed"`
	// Inputs names the sources that went into Seed: machinename, and
	// serial and uuid when the collector could read them. DMI's serial and
	// UUID are readable only by root on Linux, so a seed made by a
	// non-root run has fewer inputs, and differs from a root run's on the
	// same host. Inputs shows which one a seed is.
	Inputs []string `json:"inputs"`
}

// newInfo is the Info of a seed made from machinename, serial and uuid.
func newInfo(
	machinename, serial, uuid string,
) *Info {
	inputs := []string{"machinename"}
	if serial != "" {
		inputs = append(inputs, "serial")
	}
	if uuid != "" {
		inputs = append(inputs, "uuid")
	}

	return &Info{Seed: computeSeed(machinename, serial, uuid), Inputs: inputs}
}

// Collector is the public interface every shard variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "shard" }
func (base) Category() collector.Category { return collector.CategorySystem }

func (base) DefaultEnabled() bool { return true }

func (base) Dependencies() []string { return []string{"hostname", "dmi"} }

// New returns the shard variant for the host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// ErrNoMachineName is the error of Collect when the hostname collector gave
// no machine name, such as when it failed. A seed without the machine name is
// the same on each host that lost it, so shard gives none.
var ErrNoMachineName = errors.New("shard: the hostname collector gave no machine name")

// getMachineName extracts the raw hostname from the hostname prior. It is
// ErrNoMachineName when the prior has none.
func getMachineName(
	prior collector.PriorResults,
) (string, error) {
	info, ok := collector.GetDep[*hostname.Info](prior, "hostname")
	if !ok || info == nil || strings.TrimSpace(info.MachineName) == "" {
		return "", ErrNoMachineName
	}
	return info.MachineName, nil
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

// prodSerial returns the serial number of the product, or "" when sysfs has none.
func prodSerial(
	p *dmi.Product,
) string {
	if p == nil {
		return ""
	}
	return p.SerialNumber
}

// bbSerial returns the serial number of the baseboard, or "" when sysfs has none.
func bbSerial(
	b *dmi.Baseboard,
) string {
	if b == nil {
		return ""
	}
	return b.SerialNumber
}

// chSerial returns the serial number of the chassis, or "" when sysfs has none.
func chSerial(
	c *dmi.Chassis,
) string {
	if c == nil {
		return ""
	}
	return c.SerialNumber
}

// computeSeed concatenates the sources (no separator), hashes them
// with SHA-256, and reads the first 7 hex chars as a base-16 integer.
func computeSeed(
	machinename, serial, uuid string,
) int {
	data := machinename + serial + uuid
	h := sha256.Sum256([]byte(data))
	hex := fmt.Sprintf("%x", h)
	val, _ := strconv.ParseInt(hex[:7], 16, 64)
	return int(val)
}
