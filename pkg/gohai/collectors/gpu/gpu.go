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

// Package gpu reports graphics / compute cards present on the host.
// On Linux we wrap ghw/gpu which walks /sys/class/drm and resolves
// vendor/product names via the bundled pci.ids database. macOS has no
// DRM equivalent so the darwin variant parses
// `system_profiler SPDisplaysDataType -json`, which covers integrated
// Apple Silicon GPUs (reporting `cores`) and any discrete card.
//
// Ohai has no corresponding plugin — this collector is native to gohai
// and maps its data directly onto the typed Info below. Since there's
// no OCSF or OpenTelemetry schema for GPU devices either, field names
// follow Go-idiomatic snake_case with fields chosen to cover both
// Linux PCI data and darwin's system_profiler shape.
package gpu

import (
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info is the top-level GPU view. Empty `cards` is the normal result
// on headless servers, VMs without a virtual display, or containers
// that don't expose /sys/class/drm.
type Info struct {
	Cards []Card `json:"cards,omitempty"`
}

// Card is a single GPU / graphics device. Fields are populated
// opportunistically — Linux fills PCI metadata; macOS fills Model /
// Vendor / Cores / Bus from system_profiler. A field left empty
// simply means that signal isn't available for that backend.
type Card struct {
	Vendor   string `json:"vendor,omitempty"`    // "Apple", "NVIDIA Corporation", "Advanced Micro Devices, Inc."
	Model    string `json:"model,omitempty"`     // "Apple M1 Pro", "GP107 [GeForce GTX 1050 Ti]"
	Address  string `json:"address,omitempty"`   // Linux: PCI address "0000:03:00.0". Darwin: sppci_bus descriptor.
	VendorID string `json:"vendor_id,omitempty"` // Linux PCI vendor hex ("10de"). Darwin: not populated.
	DeviceID string `json:"device_id,omitempty"` // Linux PCI device hex ("1c82"). Darwin: not populated.
	Cores    int    `json:"cores,omitempty"`     // Darwin sppci_cores (integrated Apple GPUs). Linux: 0.
	Bus      string `json:"bus,omitempty"`       // Darwin: "builtin" / "pcie". Linux: not populated.
}

// Collector is the public interface every gpu variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "gpu" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return false }
func (base) Dependencies() []string { return nil }

// New returns the gpu variant appropriate for the host OS.
func New() Collector {
	if platform.IsDarwin() {
		return NewDarwin()
	}
	return NewLinux()
}
