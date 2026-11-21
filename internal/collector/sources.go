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

package collector

import (
	"reflect"
	"time"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// IsDarwin reports whether a collector built from src takes its macOS
// variant: src's family when it names one, else the host's, through
// platform.Detect, which a test can swap.
func IsDarwin(
	src source.Sources,
) bool {
	f := src.Family
	if f == "" {
		f = platform.Detect()
	}

	return f == family.Darwin
}

// Family is src's family, or the host's through platform.Detect when src
// names none.
func Family(
	src source.Sources,
) string {
	if src.Family != "" {
		return src.Family
	}

	return platform.Detect()
}

var (
	fsType       = reflect.TypeFor[avfs.VFS]()
	execType     = reflect.TypeFor[source.Runner]()
	nowType      = reflect.TypeFor[func() time.Time]()
	resolverType = reflect.TypeFor[source.Resolver]()
)

// Wire gives the collector variant c the sources of src, and returns c. It
// sets the exported fields of the variant that hold a source: FS (an
// avfs.VFS), Exec (a source.Runner) and Now (a func() time.Time). A field
// the variant does not have is not set. A zero source in src is the real
// host's.
func Wire[C any](
	c C,
	src source.Sources,
) C {
	v := reflect.ValueOf(c)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return c
	}
	src.Family = Family(src)
	src = src.Defaults()
	s := v.Elem()
	set := func(name string, t reflect.Type, value any) {
		f := s.FieldByName(name)
		if f.IsValid() && f.CanSet() && f.Type() == t {
			f.Set(reflect.ValueOf(value))
		}
	}
	set("FS", fsType, src.FS)
	set("Exec", execType, src.Exec)
	set("Now", nowType, src.Now)
	set("Resolver", resolverType, src.Resolver)

	return c
}
