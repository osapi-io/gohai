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

// Package gohai is the public SDK for collecting system facts.
package gohai

import (
	"encoding/json"
	"fmt"
	"time"
)

// Facts holds the result of a collection run. The Data map is keyed by
// collector name and contains each collector's typed result struct.
type Facts struct {
	Data            map[string]any `json:"-"`
	CollectTime     time.Time      `json:"collect_time"`
	CollectDuration time.Duration  `json:"collect_duration_ns"`
}

// JSON returns the compact JSON representation of the facts.
func (f *Facts) JSON() ([]byte, error) {
	return json.Marshal(f.toMap())
}

// PrettyJSON returns the indented JSON representation of the facts.
func (f *Facts) PrettyJSON() ([]byte, error) {
	return json.MarshalIndent(f.toMap(), "", "  ")
}

// Flat returns a flat dot-separated key map of all facts.
func (f *Facts) Flat() map[string]any {
	return flattenMap("", normalize(f.toMap()).(map[string]any))
}

// Get returns the value at a dot-separated key path, or nil if absent.
func (f *Facts) Get(
	path string,
) any {
	return f.Flat()[path]
}

// String returns a printable summary.
func (f *Facts) String() string {
	return fmt.Sprintf("Facts{%d collectors, took %s}", len(f.Data), f.CollectDuration)
}

func (f *Facts) toMap() map[string]any {
	out := make(map[string]any, len(f.Data)+2)
	for k, v := range f.Data {
		out[k] = v
	}
	out["collect_time"] = f.CollectTime
	out["collect_duration_ns"] = int64(f.CollectDuration)
	return out
}

func flattenMap(
	prefix string,
	in map[string]any,
) map[string]any {
	out := make(map[string]any)
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			for sk, sv := range flattenMap(key, sub) {
				out[sk] = sv
			}
			continue
		}
		out[key] = v
	}
	return out
}

// normalize round-trips through JSON to convert typed structs into
// generic map[string]any so flattenMap can walk them uniformly. If the input
// cannot be marshaled, it is returned unchanged.
func normalize(
	v any,
) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}
