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
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// QDisc holds metadata for a single queuing discipline entry.
type QDisc struct {
	Kind   string `json:"kind"`
	Handle string `json:"handle,omitempty"`
	Parent string `json:"parent,omitempty"`
	// Stats holds the counters that `tc -s` prints under the qdisc. It is
	// nil when tc prints no statistics for the qdisc.
	Stats *Stats `json:"stats,omitempty"`
}

// Stats holds the counters of one qdisc, from its "Sent" and "backlog"
// lines in `tc -s qdisc show` output.
type Stats struct {
	SentBytes      uint64 `json:"sent_bytes"`
	SentPackets    uint64 `json:"sent_packets"`
	Dropped        uint64 `json:"dropped"`
	Overlimits     uint64 `json:"overlimits"`
	Requeues       uint64 `json:"requeues"`
	BacklogBytes   uint64 `json:"backlog_bytes"`
	BacklogPackets uint64 `json:"backlog_packets"`
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
func (base) Category() collector.Category { return collector.CategoryLinux }

// DefaultEnabled returns false — tc requires iproute2 and is
// Linux-specific.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the tc collector variant for the detected host OS.
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

// parseTCOutput parses `tc -s qdisc show` output. Each qdisc stanza
// starts with a line matching:
//
//	qdisc <kind> <handle>: dev <iface> [parent <parent>] [root] ...
//
// The "Sent" and "backlog" lines that follow a qdisc line hold its
// statistics.
func parseTCOutput(
	output string,
) *Info {
	info := &Info{Interfaces: []Interface{}}
	// Build a map to accumulate qdiscs per interface.
	ifaceMap := map[string]*Interface{}
	// Track insertion order so output is deterministic.
	var order []string
	// last is the qdisc that the statistics lines belong to.
	var last *QDisc

	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if last != nil {
			words := strings.Fields(line)
			if len(words) > 0 && words[0] == "Sent" {
				parseSent(statsOf(last), words)
				continue
			}
			if len(words) > 0 && words[0] == "backlog" {
				parseBacklog(statsOf(last), words)
				continue
			}
		}
		if line == "" || !strings.HasPrefix(line, "qdisc ") {
			continue
		}
		last = nil
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

		// The parent is the handle after "parent", or "root" for a qdisc
		// that tc prints with the word root.
		parent := ""
		for i, f := range fields {
			if f == "root" {
				parent = "root"
				break
			}
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
		qdiscs := ifaceMap[iface].QDiscs
		last = &qdiscs[len(qdiscs)-1]
	}
	for _, name := range order {
		info.Interfaces = append(info.Interfaces, *ifaceMap[name])
	}
	return info
}

// statsOf returns the statistics of q, and makes them when q has none.
func statsOf(
	q *QDisc,
) *Stats {
	if q.Stats == nil {
		q.Stats = &Stats{}
	}

	return q.Stats
}

// parseSent reads a line such as
//
//	Sent 4096 bytes 32 pkt (dropped 1, overlimits 2 requeues 3)
//
// into st. A value that does not parse stays 0.
func parseSent(
	st *Stats,
	fields []string,
) {
	for i := 0; i+1 < len(fields); i++ {
		v := strings.Trim(fields[i+1], "(),")
		switch strings.Trim(fields[i], "(),") {
		case "Sent":
			st.SentBytes = parseCount(v)
		case "bytes":
			st.SentPackets = parseCount(v)
		case "dropped":
			st.Dropped = parseCount(v)
		case "overlimits":
			st.Overlimits = parseCount(v)
		case "requeues":
			st.Requeues = parseCount(v)
		}
	}
}

// parseBacklog reads a line such as
//
//	backlog 1514b 1p requeues 3
//
// into st. tc prints the bytes with a b, Kb, Mb or Gb suffix in powers of
// 1024, and the packets with a p suffix.
func parseBacklog(
	st *Stats,
	fields []string,
) {
	if len(fields) > 1 {
		st.BacklogBytes = parseSize(fields[1])
	}
	if len(fields) > 2 {
		st.BacklogPackets = parseCount(strings.TrimSuffix(fields[2], "p"))
	}
}

// parseSize converts a tc size such as 1514b, 10Kb or 2Mb to bytes. A size
// that does not parse is 0.
func parseSize(
	s string,
) uint64 {
	units := []struct {
		suffix string
		mult   float64
	}{
		{"Gb", 1 << 30},
		{"Mb", 1 << 20},
		{"Kb", 1 << 10},
		{"b", 1},
	}
	for _, u := range units {
		if n, ok := strings.CutSuffix(s, u.suffix); ok {
			f, err := strconv.ParseFloat(n, 64)
			if err != nil || f < 0 {
				return 0
			}

			return uint64(f * u.mult)
		}
	}

	return 0
}

// parseCount converts a decimal counter to a number. A counter that does
// not parse is 0.
func parseCount(
	s string,
) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}

	return n
}
