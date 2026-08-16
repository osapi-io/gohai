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

// Read facts as typed fields, by key path, and as a flat map.
//
// The SDK is the product: Facts carries a typed field per collector, so
// the compiler checks what you read. Get and Flat exist for the cases
// where the key is only known at runtime.
package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	g, err := gohai.New(gohai.WithCollectors("platform", "cpu"))
	if err != nil {
		log.Fatalf("building gohai: %v", err)
	}

	facts, err := g.Collect(context.Background())
	if err != nil {
		log.Fatalf("collecting facts: %v", err)
	}

	// Typed access — checked at compile time. Collectors that did not
	// run, or that failed, leave their field nil.
	if p := facts.Platform; p != nil {
		fmt.Printf("typed:  %s %s (%s, %s)\n",
			p.Name, p.Version, p.Family, p.CPUArchitecture)
	}

	// Key-path access — for keys chosen at runtime.
	fmt.Printf("by key: platform.family = %v\n", facts.Get("platform.family"))

	// Flat map — every leaf as a dotted key, for export or filtering.
	flat := facts.Flat()
	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Printf("flat:   %d keys, first five:\n", len(keys))
	for _, k := range keys[:min(5, len(keys))] {
		fmt.Printf("          %s = %v\n", k, flat[k])
	}
}
