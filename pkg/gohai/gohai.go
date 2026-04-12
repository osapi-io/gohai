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

package gohai

import (
	"context"
	"fmt"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

// Gohai is the SDK entry point for collecting system facts.
type Gohai struct {
	registry *collector.Registry
	selected []collector.Collector
}

// New constructs a Gohai instance with the given options.
func New(
	opts ...Option,
) (*Gohai, error) {
	g := &Gohai{registry: collector.NewRegistry()}
	registerBuiltins(g.registry)

	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	sel, err := selectCollectors(g.registry, cfg)
	if err != nil {
		return nil, err
	}
	g.selected = sel
	return g, nil
}

// Collect runs all selected collectors and returns Facts. Each collector's
// typed result is written to the matching field on Facts; collectors that
// error or aren't selected leave their field nil.
func (g *Gohai) Collect(
	ctx context.Context,
) (*Facts, error) {
	names := make([]string, 0, len(g.selected))
	for _, c := range g.selected {
		names = append(names, c.Name())
	}
	start := time.Now()
	results, err := g.registry.Run(ctx, names, nil)
	if err != nil {
		return nil, fmt.Errorf("run collectors: %w", err)
	}
	facts := &Facts{
		CollectTime:     start,
		CollectDuration: time.Since(start),
	}
	for name, result := range results {
		facts.set(name, result)
	}
	return facts, nil
}

func selectCollectors(
	reg *collector.Registry,
	cfg config,
) ([]collector.Collector, error) {
	if len(cfg.only) > 0 {
		out := make([]collector.Collector, 0, len(cfg.only))
		for _, n := range cfg.only {
			c, ok := reg.Get(n)
			if !ok {
				return nil, fmt.Errorf("unknown collector %q", n)
			}
			out = append(out, c)
		}
		return out, nil
	}
	sel, err := reg.Selected(cfg.enabled, cfg.disabled)
	if err != nil {
		return nil, fmt.Errorf("select collectors: %w", err)
	}
	return sel, nil
}

// registerBuiltins registers every built-in collector. Registration errors
// would only occur from programmer bugs (duplicate names, empty names),
// which are caught by tests — callers can rely on registration succeeding.
func registerBuiltins(
	reg *collector.Registry,
) {
	for _, c := range builtinCollectors() {
		_ = reg.Register(c)
	}
}

// builtinCollectors returns the list of built-in collectors to register.
func builtinCollectors() []collector.Collector {
	return []collector.Collector{
		platform.New(),
		hostname.New(),
		kernel.New(),
		uptime.New(),
		virtualization.New(),
		machineid.New(),
		cpu.New(),
		memory.New(),
		filesystem.New(),
		disk.New(),
		network.New(),
		process.New(),
		users.New(),
		timezone.New(),
	}
}
