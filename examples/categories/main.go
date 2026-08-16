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

// Select collectors by category, then subtract one.
//
// WithCategory adds every collector in a category. It stacks with the
// other selection options, and WithDisabled still subtracts — so this
// is how you say "all the hardware facts except the expensive one".
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	g, err := gohai.New(
		gohai.WithCategory("hardware"),
		gohai.WithDisabled("disk"),
	)
	if err != nil {
		log.Fatalf("building gohai: %v", err)
	}

	facts, err := g.Collect(context.Background())
	if err != nil {
		log.Fatalf("collecting facts: %v", err)
	}

	// Disk was subtracted, so its field stays nil while the rest of the
	// hardware category is populated.
	fmt.Printf("cpu collected:    %t\n", facts.CPU != nil)
	fmt.Printf("memory collected: %t\n", facts.Memory != nil)
	fmt.Printf("disk collected:   %t (disabled)\n", facts.Disk != nil)
}
