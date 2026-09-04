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

// Surface per-collector duration and failure.
//
// Collect returns an error only when collection as a whole fails. An
// individual collector that errors is dropped from the typed output and
// its field stays nil — indistinguishable from "not selected". WithTimings
// is how that failure becomes visible.
package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	g, err := gohai.New(gohai.WithDefaults(), gohai.WithTimings())
	if err != nil {
		log.Fatalf("building gohai: %v", err)
	}

	facts, err := g.Collect(context.Background())
	if err != nil {
		log.Fatalf("collecting facts: %v", err)
	}

	if facts.Timings == nil {
		log.Fatal("no timings recorded; was WithTimings set?")
	}

	names := make([]string, 0, len(facts.Timings.Collectors))
	for name := range facts.Timings.Collectors {
		names = append(names, name)
	}
	slices.Sort(names)

	var failed int
	for _, name := range names {
		t := facts.Timings.Collectors[name]
		if t.Status == "ok" {
			continue
		}
		failed++
		fmt.Printf("failed: %-18s %-10s %s\n",
			name, time.Duration(t.DurationNs), t.Error)
	}

	fmt.Printf("\n%d collectors ran, %d failed, total %s\n",
		len(names), failed, facts.CollectDuration)
}
