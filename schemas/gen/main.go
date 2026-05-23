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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func qualifiedNamer(t reflect.Type) string {
	pkg := t.PkgPath()
	name := t.Name()

	if !strings.Contains(pkg, "gohai/pkg/gohai/collectors/") {
		return name
	}

	parts := strings.Split(pkg, "/")
	collector := parts[len(parts)-1]
	collector = strings.ReplaceAll(collector, "_", "")
	collector = strings.ToUpper(collector[:1]) + collector[1:]

	return collector + name
}

func main() {
	out := flag.String("out", "gohai.schema.json", "output file path")
	flag.Parse()

	r := &jsonschema.Reflector{
		Namer: qualifiedNamer,
	}
	schema := r.Reflect(&gohai.Facts{})

	schema.ID = "https://gohai.dev/schemas/gohai.schema.json"
	schema.Title = "gohai Facts"
	schema.Description = "System facts collected by gohai (https://github.com/osapi-io/gohai)"

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	data = append(data, '\n')

	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *out, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s (%d bytes)\n", *out, len(data))
}
