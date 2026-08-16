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

// Collect the default set of facts and print them as JSON.
//
// This is the shortest path through the SDK: build with the default
// collector set, collect, and marshal. Every other example varies one
// dimension of this.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	g, err := gohai.New(gohai.WithDefaults())
	if err != nil {
		log.Fatalf("building gohai: %v", err)
	}

	facts, err := g.Collect(context.Background())
	if err != nil {
		log.Fatalf("collecting facts: %v", err)
	}

	out, err := facts.PrettyJSON()
	if err != nil {
		log.Fatalf("encoding facts: %v", err)
	}

	fmt.Println(string(out))
}
