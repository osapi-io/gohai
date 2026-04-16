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
// one entry per SCSI address, with transport, type, make+model name,
// firmware revision, and the backing device node. macOS has no
// equivalent interface so the Darwin variant returns an empty Info.
package scsi

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds the enumerated SCSI devices keyed by address.
type Info struct {
	Devices map[string]Device `json:"devices"`
}

// Device is one SCSI device entry parsed from an `lsscsi` row.
// Matches the shape Ohai's linux/scsi.rb emits under
// node['scsi'][<address>].
type Device struct {
	SCSIAddr  string `json:"scsi_addr"`
	Type      string `json:"type,omitempty"`
	Transport string `json:"transport,omitempty"`
	Name      string `json:"name,omitempty"`
	Revision  string `json:"revision,omitempty"`
	Device    string `json:"device,omitempty"`
}

// Collector is the public interface every scsi variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "scsi" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the scsi variant appropriate for the host OS.
func New() Collector {
	if platform.IsDarwin() {
		return NewDarwin()
	}
	return NewLinux()
}

// parseLsscsi parses `lsscsi` output into a map keyed by SCSI address.
// Matches Ohai's linux/scsi.rb algorithm: split on whitespace, first
// bracketed token is the address (brackets stripped), second is type,
// third is transport, last is device node, second-to-last is firmware
// revision, everything between transport and revision joined with
// single spaces is the vendor+model "name".
//
// Lines without at least 5 tokens (the minimum to carry addr, type,
// transport, revision, device) are skipped — leaves the map entry
// absent rather than half-populated.
func parseLsscsi(
	out []byte,
) map[string]Device {
	result := map[string]Device{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 {
			continue
		}
		addr := strings.Trim(fields[0], "[]")
		if addr == "" {
			continue
		}
		d := Device{
			SCSIAddr:  addr,
			Type:      fields[1],
			Transport: fields[2],
			Device:    fields[len(fields)-1],
			Revision:  fields[len(fields)-2],
		}
		if len(fields) > 5 {
			d.Name = strings.Join(fields[3:len(fields)-2], " ")
		}
		result[addr] = d
	}
	return result
}
