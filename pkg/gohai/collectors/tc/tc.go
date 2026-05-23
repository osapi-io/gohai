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

// Package tc reports Linux traffic control (qdisc) configuration via
// `tc -s qdisc show`. macOS does not have tc; Darwin returns nil.
// Mirrors Ohai's linux/tc.rb methodology — parse per-interface qdisc
// entries from the tc command output.
package tc

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// QDisc holds metadata for a single queuing discipline entry.
type QDisc struct {
	Kind   string `json:"kind"`
	Handle string `json:"handle,omitempty"`
	Parent string `json:"parent,omitempty"`
}

// Interface holds the list of qdiscs for a single network interface.
type Interface struct {
	Name   string  `json:"name"`
	QDiscs []QDisc `json:"qdiscs"`
}

// Info holds traffic-control configuration indexed by interface.
type Info struct {
	Interfaces []Interface `json:"interfaces"`
}

// Collector is the public interface every tc variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields shared by every OS variant.
type base struct{}

// Name returns "tc".
func (base) Name() string { return "tc" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — tc requires iproute2 and is
// Linux-specific.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the tc collector variant for the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseTCOutput parses `tc -s qdisc show` output. Each qdisc stanza
// starts with a line matching:
//
//	qdisc <kind> <handle>: dev <iface> [parent <parent>] [root] ...
//
// Mirrors Ohai's linux/tc.rb parsing strategy.
func parseTCOutput(
	output string,
) *Info {
	info := &Info{Interfaces: []Interface{}}
	// Build a map to accumulate qdiscs per interface.
	ifaceMap := map[string]*Interface{}
	// Track insertion order so output is deterministic.
	var order []string

	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.HasPrefix(line, "qdisc ") {
			continue
		}
		fields := strings.Fields(line)
		// Minimum: "qdisc" <kind> <handle> "dev" <iface>
		if len(fields) < 5 {
			continue
		}
		kind := fields[1]
		handle := strings.TrimSuffix(fields[2], ":")
		// Find "dev" keyword.
		devIdx := -1
		for i, f := range fields {
			if f == "dev" {
				devIdx = i
				break
			}
		}
		if devIdx < 0 || devIdx+1 >= len(fields) {
			continue
		}
		iface := fields[devIdx+1]

		// Extract optional parent.
		parent := ""
		for i, f := range fields {
			if f == "parent" && i+1 < len(fields) {
				parent = fields[i+1]
				break
			}
		}

		if _, exists := ifaceMap[iface]; !exists {
			ifaceMap[iface] = &Interface{Name: iface, QDiscs: []QDisc{}}
			order = append(order, iface)
		}
		ifaceMap[iface].QDiscs = append(ifaceMap[iface].QDiscs, QDisc{
			Kind:   kind,
			Handle: handle,
			Parent: parent,
		})
	}
	for _, name := range order {
		info.Interfaces = append(info.Interfaces, *ifaceMap[name])
	}
	return info
}
