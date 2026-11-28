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
	"slices"
	"sync"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Gohai is the SDK entry point for collecting system facts.
type Gohai struct {
	registry    *collector.Registry
	selected    []collector.Collector
	withTimings bool
}

// New constructs a Gohai instance with the given options.
func New(
	opts ...Option,
) (*Gohai, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// Each of the collectors picks its variant by the family, so find it
	// once, not once for each collector.
	cfg.sources.Family = collector.Family(cfg.sources)
	g := &Gohai{registry: collector.NewRegistry()}
	if err := registerBuiltins(g.registry, cfg.sources); err != nil {
		return nil, err
	}

	sel, err := selectCollectors(g.registry, cfg)
	if err != nil {
		return nil, err
	}
	g.selected = sel
	g.withTimings = cfg.withTimings
	return g, nil
}

// Collect runs all selected collectors and returns Facts. Each collector's
// typed result is written to the matching field on Facts; collectors that
// error or aren't selected leave their field nil, and Facts.Errors has the
// error of each that failed. A cancelled ctx is an error, and Collect then
// returns the Facts of the collectors that finished with it, so a caller with
// a deadline keeps what was collected. Any other error returns nil Facts. When the instance
// was built with WithTimings, per-collector durations are embedded in
// Facts.Timings.
func (g *Gohai) Collect(
	ctx context.Context,
) (*Facts, error) {
	names := make([]string, 0, len(g.selected))
	for _, c := range g.selected {
		names = append(names, c.Name())
	}
	start := time.Now()

	var (
		samples   map[string]CollectorTiming
		errorsMu  sync.Mutex
		errs      map[string]string
		statusMu  sync.Mutex
		status    = make(map[string]Status, len(names))
		durations = make(map[string]time.Duration, len(names))
	)
	hooks := collector.Hooks{
		OnError: func(name string, err error) {
			errorsMu.Lock()
			defer errorsMu.Unlock()
			if errs == nil {
				errs = make(map[string]string)
			}
			errs[name] = err.Error()
		},
		OnComplete: func(name string, dur time.Duration, _ error) {
			statusMu.Lock()
			defer statusMu.Unlock()
			durations[name] = dur
		},
		OnStatus: func(name string, s collector.Status, _ error) {
			statusMu.Lock()
			defer statusMu.Unlock()
			status[name] = statusFrom(s)
		},
	}

	results, runErr := g.registry.Run(ctx, names, hooks)
	if runErr != nil && results == nil {
		return nil, fmt.Errorf("run collectors: %w", runErr)
	}

	facts := &Facts{
		CollectTime:     start,
		CollectDuration: time.Since(start),
		Errors:          errs,
		Status:          status,
	}
	if len(status) == 0 {
		facts.Status = nil
	}
	for name, result := range results {
		if result == nil || facts.set(name, result) {
			continue
		}
		// A collector that gives a type its Facts field doesn't hold would
		// lose its fact with no word, so the mismatch is its error.
		if facts.Errors == nil {
			facts.Errors = make(map[string]string)
		}
		facts.Errors[name] = fmt.Sprintf("gave a %T, which its Facts field does not hold", result)
		facts.Status[name] = StatusFailed
	}
	if g.withTimings {
		samples = make(map[string]CollectorTiming, len(durations))
		for name, dur := range durations {
			samples[name] = CollectorTiming{
				DurationNs: dur.Nanoseconds(),
				Status:     facts.Status[name],
				Error:      facts.Errors[name],
			}
		}
		facts.Timings = &Timings{Collectors: samples}
	}
	if runErr != nil {
		return facts, fmt.Errorf("run collectors: %w", runErr)
	}
	return facts, nil
}

func selectCollectors(
	reg *collector.Registry,
	cfg config,
) ([]collector.Collector, error) {
	// WithCollectors wins outright: an explicit roster, no defaults. It
	// names each collector, so a disabled list with it is a mistake.
	if len(cfg.only) > 0 {
		if len(cfg.disabled) > 0 {
			return nil, fmt.Errorf(
				"WithCollectors and WithDisabled together: name only the collectors to run",
			)
		}
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
	// Expand WithCategory into an additive enable list. Unknown
	// categories (zero collectors match) error so typos surface
	// immediately rather than silently selecting nothing.
	enabled := cfg.enabled
	for _, cat := range cfg.categories {
		names := reg.NamesInCategory(collector.Category(cat))
		if len(names) == 0 {
			return nil, fmt.Errorf("unknown category %q", cat)
		}
		enabled = append(enabled, names...)
	}
	// Without WithDefaults / WithEnabled / WithCategory, return
	// empty, because gohai.New() is opt-in. Still validate the disabled
	// list so unknown names error consistently.
	if !cfg.useDefaults && len(enabled) == 0 {
		if _, err := reg.SelectedWith(false, nil, cfg.disabled); err != nil {
			return nil, fmt.Errorf("select collectors: %w", err)
		}
		return nil, nil
	}
	sel, err := reg.SelectedWith(cfg.useDefaults, enabled, cfg.disabled)
	if err != nil {
		return nil, fmt.Errorf("select collectors: %w", err)
	}
	if err := disabledDependency(reg, sel, cfg.disabled); err != nil {
		return nil, err
	}
	return sel, nil
}

// disabledDependency is the error of a disabled collector that a selected
// one depends on, directly or through another dependency, or nil. The
// registry adds each dependency to a run, so without this check the disabled
// collector would run and fill its Facts field: the caller asked for no read
// of it. The error names both, so that the caller can change the options.
func disabledDependency(
	reg *collector.Registry,
	sel []collector.Collector,
	disabled []string,
) error {
	if len(disabled) == 0 {
		return nil
	}
	off := make(map[string]bool, len(disabled))
	for _, n := range disabled {
		off[n] = true
	}
	for _, c := range sel {
		seen := map[string]bool{}
		queue := slices.Clone(c.Dependencies())
		for len(queue) > 0 {
			dep := queue[0]
			queue = queue[1:]
			if seen[dep] {
				continue
			}
			seen[dep] = true
			if off[dep] {
				return fmt.Errorf(
					"collector %q is disabled, but %q depends on it: enable it, or disable %q too",
					dep, c.Name(), c.Name(),
				)
			}
			if d, ok := reg.Get(dep); ok {
				queue = append(queue, d.Dependencies()...)
			}
		}
	}

	return nil
}

// registerBuiltins registers every built-in collector. A registration error
// is a duplicate or empty name, or a source field of the wrong type, in
// builtinCollectors: a bug that would drop a collector with no signal, so the
// caller returns it.
func registerBuiltins(
	reg *collector.Registry,
	src source.Sources,
) error {
	for _, c := range builtinCollectorsFn(src) {
		if err := reg.Register(c); err != nil {
			return fmt.Errorf("register built-in collector %q: %w", c.Name(), err)
		}
	}

	return nil
}

// builtinCollectorsFn is builtinCollectors, which a test replaces.
var builtinCollectorsFn = builtinCollectors

// statusFrom is the Status of the registry's status s. Each internal status
// has a Status of the same value. A status that has none, which TestStatus
// stops, is StatusFailed, so a fact never carries a status with no constant.
func statusFrom(
	s collector.Status,
) Status {
	switch s {
	case collector.StatusOK:
		return StatusOK
	case collector.StatusAbsent:
		return StatusAbsent
	case collector.StatusUnsupported:
		return StatusUnsupported
	case collector.StatusPartial:
		return StatusPartial
	default:
		return StatusFailed
	}
}
