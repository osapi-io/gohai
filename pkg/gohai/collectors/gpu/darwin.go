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

package gpu

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin calls `system_profiler SPDisplaysDataType -json` through the
// shared Executor. Apple's output uses "sppci_"/"spdisplays_" prefixes
// on every key; we normalize them onto Card.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect shells out to system_profiler. On any failure (exec missing,
// JSON malformed) we return an empty Info — matches the silent-on-miss
// convention used elsewhere.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	if d.Exec == nil {
		return info, nil
	}
	out, err := d.Exec.Execute(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	if err != nil {
		return info, nil
	}
	var payload struct {
		Displays []darwinDisplay `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return info, nil
	}
	for _, d := range payload.Displays {
		info.Cards = append(info.Cards, d.toCard())
	}
	return info, nil
}

// darwinDisplay mirrors the shape of each entry in the
// SPDisplaysDataType array. Unused keys (displays/connections) are
// intentionally omitted — this collector reports the adapter, not the
// attached displays.
type darwinDisplay struct {
	Name   string `json:"_name"`
	Vendor string `json:"spdisplays_vendor"`
	Bus    string `json:"sppci_bus"`
	Model  string `json:"sppci_model"`
	Cores  string `json:"sppci_cores"`
}

// toCard lifts the darwin display payload onto Card. We strip Apple's
// "sppci_" / "spdisplays_" prefixes so consumers don't have to know
// system_profiler's naming convention.
func (d darwinDisplay) toCard() Card {
	c := Card{
		Vendor:  cleanSPValue(d.Vendor),
		Model:   d.Model,
		Address: d.Bus,
		Bus:     cleanSPValue(d.Bus),
	}
	if c.Model == "" {
		c.Model = d.Name
	}
	if n, err := strconv.Atoi(d.Cores); err == nil {
		c.Cores = n
	}
	return c
}

// cleanSPValue strips the Apple-ish `sppci_` / `spdisplays_` /
// `sppci_vendor_` prefixes system_profiler wraps human-readable values
// with. An unprefixed value passes through untouched.
func cleanSPValue(
	v string,
) string {
	for _, prefix := range []string{"sppci_vendor_", "spdisplays_", "sppci_"} {
		if strings.HasPrefix(v, prefix) {
			return strings.TrimPrefix(v, prefix)
		}
	}
	return v
}
