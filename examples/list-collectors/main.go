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

// Enumerate the available collectors without collecting anything.
//
// The registry answers "what could I ask for?" — useful for building
// flags, help output, or validating configuration before a run.
package main

import (
	"fmt"
	"sort"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	registry := gohai.NewRegistry()

	byCategory := map[string][]string{}
	for _, name := range registry.Names() {
		category := registry.CategoryOf(name)
		byCategory[category] = append(byCategory[category], name)
	}

	categories := make([]string, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		names := byCategory[category]
		sort.Strings(names)
		fmt.Printf("%s (%d)\n", category, len(names))
		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
	}
}
