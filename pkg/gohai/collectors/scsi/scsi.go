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

// Package scsi enumerates SCSI devices visible to the host by parsing
// the output of `lsscsi`. Mirrors Ohai's linux/scsi.rb methodology:
// one entry per SCSI address, with type, vendor, make+model name,
// firmware revision, and the backing device node. macOS has no
// equivalent interface so the Darwin variant returns an empty Info.
package scsi

import (
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds the enumerated SCSI devices keyed by address.
type Info struct {
	Devices map[string]Device `json:"devices"`
}

// Device is one SCSI device entry parsed from an `lsscsi` row.
// Matches the shape Ohai's linux/scsi.rb emits under
// node['scsi'][<address>].
type Device struct {
	SCSIAddr string `json:"scsi_addr"`
	Type     string `json:"type,omitempty"`
	// Vendor is the vendor column of `lsscsi`, such as ATA or LSI. Ohai
	// names it transport, but `lsscsi` prints the transport only with -t.
	Vendor   string `json:"vendor,omitempty"`
	Name     string `json:"name,omitempty"`
	Revision string `json:"revision,omitempty"`
	Device   string `json:"device,omitempty"`
}

// Collector is the public interface every scsi variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "scsi" }
func (base) Category() collector.Category { return collector.CategoryHardware }
func (base) DefaultEnabled() bool         { return false }
func (base) Dependencies() []string       { return nil }

// New returns the scsi variant appropriate for the host OS.
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

// parseLsscsi parses `lsscsi` output into a map keyed by SCSI address.
// Matches Ohai's linux/scsi.rb algorithm: split on whitespace, first
// bracketed token is the address (brackets stripped), second is type,
// third is transport, last is device node, second-to-last is firmware
// revision, everything between transport and revision joined with
// single spaces is the vendor+model "name".
//
// Lines without at least 5 tokens (the minimum to carry addr, type,
// transport, revision, device) are skipped. That leaves the map entry
// absent rather than half-populated.
func parseLsscsi(
	out []byte,
) map[string]Device {
	result := map[string]Device{}
	// strings.Lines, not a bufio.Scanner, which stops with no error at a
	// line over 64 KiB and drops every line after it.
	for rawLine := range strings.Lines(string(out)) {
		fields := strings.Fields(strings.TrimRight(rawLine, "\r\n"))
		if len(fields) < 5 {
			continue
		}
		addr := strings.Trim(fields[0], "[]")
		if addr == "" {
			continue
		}
		d := Device{
			SCSIAddr: addr,
			Type:     fields[1],
			Vendor:   fields[2],
			Device:   fields[len(fields)-1],
			Revision: fields[len(fields)-2],
		}
		if len(fields) > 5 {
			d.Name = strings.Join(fields[3:len(fields)-2], " ")
		}
		result[addr] = d
	}
	return result
}
