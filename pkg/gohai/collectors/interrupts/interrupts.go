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

// Package interrupts reports IRQ statistics and per-CPU event counts as
// found in /proc/interrupts on Linux.
package interrupts

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// IRQ holds the statistics for a single interrupt line.
type IRQ struct {
	// Number is the IRQ identifier. Numeric for hardware IRQs ("0", "9"),
	// or a string label for architecture-defined entries ("NMI", "LOC",
	// "ERR", "MIS", etc.).
	Number string `json:"number"`
	// Type is the interrupt controller type (e.g. "IO-APIC", "PCI-MSI").
	// Empty for non-numeric IRQs that lack this field.
	Type string `json:"type,omitempty"`
	// Device is the driver/device name associated with this IRQ.
	// Empty for non-numeric IRQs.
	Device string `json:"device,omitempty"`
	// CountsPerCPU contains the event count for each CPU in CPU-index
	// order. The slice length equals the number of CPUs reported in
	// the /proc/interrupts header.
	CountsPerCPU []int64 `json:"counts_per_cpu"`
}

// Info holds all interrupt lines parsed from /proc/interrupts.
type Info struct {
	IRQs []IRQ `json:"irqs"`
}

// Collector is the public interface every interrupts variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "interrupts".
func (base) Name() string { return "interrupts" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — interrupts is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the interrupts collector variant appropriate to the
// detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseInterrupts parses the content of /proc/interrupts and returns an
// Info struct. The first line is the CPU header; subsequent lines are
// IRQ entries in the format:
//
//	<IRQ>: <count0> <count1> ... [type] [vector] [device]
//
// Non-numeric IRQ labels (NMI, LOC, ERR, MIS, …) may carry a type string
// in the trailing field but no vector or device.
func parseInterrupts(
	content []byte,
) (*Info, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// First line: "           CPU0       CPU1 ..."
	if !scanner.Scan() {
		return &Info{IRQs: []IRQ{}}, nil
	}
	header := scanner.Text()
	cpuCount := len(strings.Fields(header))

	irqs := []IRQ{}
	for scanner.Scan() {
		line := scanner.Text()

		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		number := strings.TrimSpace(line[:colonIdx])
		rest := line[colonIdx+1:]

		fields := strings.Fields(rest)

		counts := make([]int64, cpuCount)
		for i := range cpuCount {
			if i < len(fields) {
				v, err := strconv.ParseInt(fields[i], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parse interrupt count cpu%d irq %s: %w", i, number, err)
				}
				counts[i] = v
			}
		}

		irq := IRQ{
			Number:       number,
			CountsPerCPU: counts,
		}

		// Fields after the CPU counts are type, optional vector, and device.
		// For non-numeric IRQs the trailing content is just a type label.
		// Guard against lines with fewer fields than cpuCount (e.g. ERR on
		// a multi-CPU host that only reports one count).
		extraStart := cpuCount
		if extraStart > len(fields) {
			extraStart = len(fields)
		}
		extra := fields[extraStart:]
		isNumeric := isNumericIRQ(number)
		if isNumeric && len(extra) >= 1 {
			irq.Type = extra[0]
			// extra[1] is the vector (skipped), extra[2:] joined is the device.
			if len(extra) >= 3 {
				irq.Device = strings.Join(extra[2:], " ")
			}
		} else if !isNumeric && len(extra) >= 1 {
			irq.Type = strings.Join(extra, " ")
		}

		irqs = append(irqs, irq)
	}

	return &Info{IRQs: irqs}, nil
}

// isNumericIRQ reports whether the IRQ number string is a decimal integer.
func isNumericIRQ(
	number string,
) bool {
	_, err := strconv.ParseInt(number, 10, 64)
	return err == nil
}
