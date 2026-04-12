// Copyright (c) 2024 John Dewey

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

package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func writeOutput(
	out io.Writer,
	facts *gohai.Facts,
	pretty, flat bool,
) error {
	if flat {
		return writeFlat(out, facts)
	}
	return writeJSON(out, facts, pretty)
}

func writeFlat(
	out io.Writer,
	facts *gohai.Facts,
) error {
	flatMap := facts.Flat()
	keys := make([]string, 0, len(flatMap))
	for k := range flatMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(out, "%s=%v\n", k, flatMap[k]); err != nil {
			return fmt.Errorf("write flat output: %w", err)
		}
	}
	return nil
}

func writeJSON(
	out io.Writer,
	facts *gohai.Facts,
	pretty bool,
) error {
	var (
		b   []byte
		err error
	)
	if pretty {
		b, err = facts.PrettyJSON()
	} else {
		b, err = facts.JSON()
	}
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	if _, err := out.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
