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

// Restrict collection to specific collectors.
//
// WithCollectors overrides the default set entirely: only the named
// collectors and their dependencies run. Use this when you need a few
// known facts and do not want to pay for the rest.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	g, err := gohai.New(gohai.WithCollectors("cpu", "memory"))
	if err != nil {
		log.Fatalf("building gohai: %v", err)
	}

	facts, err := g.Collect(context.Background())
	if err != nil {
		log.Fatalf("collecting facts: %v", err)
	}

	if facts.CPU != nil {
		fmt.Printf("cpu:    %d logical cores (%s)\n",
			facts.CPU.Count, facts.CPU.ModelName)
	}
	if facts.Memory != nil {
		fmt.Printf("memory: %d bytes total, %.1f%% used\n",
			facts.Memory.Total, facts.Memory.UsedPercent)
	}
}
