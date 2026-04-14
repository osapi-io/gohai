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

package collector

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Hooks are optional per-run observability callbacks. Both fields may
// be nil.
type Hooks struct {
	// OnError is invoked when a collector's Collect returns an error.
	// Called from the collector's goroutine, concurrent with other
	// collectors in the same level.
	OnError func(name string, err error)
	// OnComplete is invoked after every collector's Collect returns,
	// regardless of success or failure, with the wall-clock duration
	// of that call. Useful for --debug timing output.
	OnComplete func(name string, dur time.Duration, err error)
}

// Registry holds the set of registered collectors.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry. Returns an error if the name is
// empty or already registered.
func (r *Registry) Register(
	c Collector,
) error {
	if c.Name() == "" {
		return errors.New("collector name must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[c.Name()]; exists {
		return fmt.Errorf("collector %q already registered", c.Name())
	}
	r.collectors[c.Name()] = c
	return nil
}

// Get returns the collector with the given name, if registered.
func (r *Registry) Get(
	name string,
) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

// NamesInCategory returns the names of every collector whose Category
// matches cat. Returns an empty slice (never nil) when no collector
// has that category — callers should treat that as a user error
// ("unknown category") rather than silently running nothing.
func (r *Registry) NamesInCategory(
	cat string,
) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []string{}
	for name, c := range r.collectors {
		if c.Category() == cat {
			out = append(out, name)
		}
	}
	return out
}

// Names returns the names of all registered collectors.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.collectors))
	for n := range r.collectors {
		names = append(names, n)
	}
	return names
}

// Selected returns the collectors that should run treating every
// DefaultEnabled() collector as on, plus anything in `enable`, minus
// anything in `disable`. Kept for backward compatibility; new callers
// should use SelectedWith and decide explicitly whether to honour
// DefaultEnabled().
func (r *Registry) Selected(
	enable []string,
	disable []string,
) ([]Collector, error) {
	return r.SelectedWith(true, enable, disable)
}

// SelectedWith returns the collectors that should run. When
// useDefaults is true, every collector with DefaultEnabled()==true is
// included. The enable list adds names regardless of DefaultEnabled.
// The disable list subtracts names. Unknown names in enable/disable
// return an error.
func (r *Registry) SelectedWith(
	useDefaults bool,
	enable []string,
	disable []string,
) ([]Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, n := range enable {
		if _, ok := r.collectors[n]; !ok {
			return nil, fmt.Errorf("unknown collector %q in enable list", n)
		}
	}
	for _, n := range disable {
		if _, ok := r.collectors[n]; !ok {
			return nil, fmt.Errorf("unknown collector %q in disable list", n)
		}
	}

	enableSet := make(map[string]bool, len(enable))
	for _, n := range enable {
		enableSet[n] = true
	}
	disableSet := make(map[string]bool, len(disable))
	for _, n := range disable {
		disableSet[n] = true
	}

	out := make([]Collector, 0, len(r.collectors))
	for name, c := range r.collectors {
		if disableSet[name] {
			continue
		}
		if (useDefaults && c.DefaultEnabled()) || enableSet[name] {
			out = append(out, c)
		}
	}
	return out, nil
}

// Run executes the named collectors in dependency order. Dependencies are
// auto-included even if not in the names list. Collectors at the same level
// (no inter-dependencies) run concurrently. Returns a map of collector name
// to result. Failed collectors are omitted from the result map. hooks may
// be zero-valued; any non-nil field is invoked per collector.
func (r *Registry) Run(
	ctx context.Context,
	names []string,
	hooks Hooks,
) (map[string]any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted, err := r.expandWithDeps(names)
	if err != nil {
		return nil, err
	}

	levels, err := r.topoLevels(wanted)
	if err != nil {
		return nil, err
	}

	results := make(map[string]any, len(wanted))
	var resultsMu sync.Mutex

	for _, level := range levels {
		// Snapshot prior results once per level. Collectors at the
		// same level have no inter-dependencies by definition, so
		// they all see the same prior state — matches the
		// topological contract.
		resultsMu.Lock()
		prior := make(PriorResults, len(results))
		for k, v := range results {
			prior[k] = v
		}
		resultsMu.Unlock()

		var wg sync.WaitGroup
		for _, name := range level {
			c := r.collectors[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				out, cerr := c.Collect(ctx, prior)
				dur := time.Since(start)
				if hooks.OnComplete != nil {
					hooks.OnComplete(c.Name(), dur, cerr)
				}
				if cerr != nil {
					if hooks.OnError != nil {
						hooks.OnError(c.Name(), cerr)
					}
					return
				}
				resultsMu.Lock()
				results[c.Name()] = out
				resultsMu.Unlock()
			}()
		}
		wg.Wait()
	}
	return results, nil
}

func (r *Registry) expandWithDeps(
	names []string,
) (map[string]bool, error) {
	wanted := make(map[string]bool)
	var visit func(string) error
	visit = func(n string) error {
		if wanted[n] {
			return nil
		}
		c, ok := r.collectors[n]
		if !ok {
			return fmt.Errorf("unknown collector %q", n)
		}
		wanted[n] = true
		for _, dep := range c.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return wanted, nil
}

func (r *Registry) topoLevels(
	wanted map[string]bool,
) ([][]string, error) {
	indeg := make(map[string]int, len(wanted))
	for n := range wanted {
		indeg[n] = 0
	}
	for n := range wanted {
		for range r.collectors[n].Dependencies() {
			indeg[n]++
		}
	}

	var levels [][]string
	remaining := len(indeg)
	for remaining > 0 {
		var level []string
		for n, d := range indeg {
			if d == 0 {
				level = append(level, n)
			}
		}
		if len(level) == 0 {
			return nil, fmt.Errorf("dependency cycle detected among collectors")
		}
		sort.Strings(level)
		levels = append(levels, level)
		for _, n := range level {
			delete(indeg, n)
			remaining--
			for m := range indeg {
				for _, dep := range r.collectors[m].Dependencies() {
					if dep == n {
						indeg[m]--
					}
				}
			}
		}
	}
	return levels, nil
}
