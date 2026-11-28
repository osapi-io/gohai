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

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/ocsf"
)

// WriteOutput writes facts to out in the requested format.
func WriteOutput(
	out io.Writer,
	facts *gohai.Facts,
	pretty bool,
	flat bool,
) error {
	if flat {
		return WriteFlat(out, facts)
	}

	return WriteJSON(out, facts, pretty)
}

// WriteFlat writes facts as sorted dot-separated key=value pairs.
func WriteFlat(
	out io.Writer,
	facts *gohai.Facts,
) error {
	flatMap, err := facts.Flat()
	if err != nil {
		return fmt.Errorf("write flat output: %w", err)
	}

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

// WriteJSON writes facts as JSON, optionally pretty-printed.
func WriteJSON(
	out io.Writer,
	facts *gohai.Facts,
	pretty bool,
) error {
	marshal := facts.JSON
	if pretty {
		marshal = facts.PrettyJSON
	}
	b, err := marshal()
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}

	if _, err := out.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

// WriteOCSF converts facts to an OCSF inventory_info event and writes
// the JSON to out.
func WriteOCSF(
	out io.Writer,
	facts *gohai.Facts,
	pretty bool,
) error {
	event := ocsf.FromFacts(facts)
	var (
		b   []byte
		err error
	)
	if pretty {
		b, err = json.MarshalIndent(event, "", "  ")
	} else {
		b, err = json.Marshal(event)
	}
	if err != nil {
		return fmt.Errorf("encode ocsf output: %w", err)
	}

	if _, err := out.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write ocsf output: %w", err)
	}

	return nil
}

// WriteCollectorList writes the collectors of reg grouped by category.
func WriteCollectorList(
	out io.Writer,
	reg gohai.Registry,
) error {
	byCat := map[string][]string{}

	for _, n := range reg.Names() {
		cat := reg.CategoryOf(n)
		byCat[cat] = append(byCat[cat], n)
	}

	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}

	sort.Strings(cats)

	for _, cat := range cats {
		names := byCat[cat]
		sort.Strings(names)

		if _, err := fmt.Fprintf(out, "[%s]\n", cat); err != nil {
			return fmt.Errorf("write collector list: %w", err)
		}

		for _, n := range names {
			if _, err := fmt.Fprintf(out, "  %s\n", n); err != nil {
				return fmt.Errorf("write collector list: %w", err)
			}
		}
	}

	return nil
}

// Print writes a line to w.
func Print(
	w io.Writer,
	s string,
) {
	_, _ = fmt.Fprintln(w, s)
}

// Printf writes a formatted string to w.
func Printf(
	w io.Writer,
	format string,
	a ...any,
) {
	_, _ = fmt.Fprintf(w, format, a...)
}
