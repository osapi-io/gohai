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
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/avfs/avfs"

	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// IsDarwin reports whether a collector built from src takes its macOS
// variant, by Family.
func IsDarwin(
	src source.Sources,
) bool {
	return Family(src) == family.Darwin
}

// Family is src's family. When src names none, it is the family of src.FS's
// /etc/os-release, so that a fake filesystem and the variant agree. With no
// FS either, it is the real host's, through platform.Detect, which a test
// can swap.
func Family(
	src source.Sources,
) string {
	switch {
	case src.Family != "":
		return src.Family
	case src.FS != nil:
		return family.FromFS(src.FS)
	default:
		return platform.Detect()
	}
}

var (
	fsType       = reflect.TypeFor[avfs.VFS]()
	execType     = reflect.TypeFor[source.Runner]()
	nowType      = reflect.TypeFor[func() time.Time]()
	resolverType = reflect.TypeFor[source.Resolver]()
	httpType     = reflect.TypeFor[http.RoundTripper]()
)

// sourceFields are the exported fields of a collector variant that hold a
// source, and the type each must have.
var sourceFields = []struct {
	name string
	typ  reflect.Type
}{
	{"FS", fsType},
	{"Exec", execType},
	{"Now", nowType},
	{"Resolver", resolverType},
	{"HTTP", httpType},
}

// Wire gives the collector variant c the sources of src, and returns c. It
// sets the exported fields of the variant that hold a source: FS (an
// avfs.VFS), Exec (a source.Runner), Now (a func() time.Time), Resolver
// (a source.Resolver) and HTTP (an http.RoundTripper). A field the variant
// does not have is not set. A zero source in src is the real host's, except
// a zero HTTP, which stays nil: the cloud collectors' shared transport. A field with one of those names and a
// different type is not set: CheckSources reports it, and Registry.Register
// refuses the collector, so the collector cannot read the real host with
// nothing to say so.
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
	values := map[string]any{
		"FS": src.FS, "Exec": src.Exec, "Now": src.Now, "Resolver": src.Resolver,
		"HTTP": src.HTTP,
	}
	for _, sf := range sourceFields {
		f := s.FieldByName(sf.name)
		if !f.IsValid() || f.Type() != sf.typ || !f.CanSet() {
			continue
		}
		// A zero HTTP stays zero: the cloud collectors' shared transport.
		if values[sf.name] == nil {
			continue
		}
		f.Set(reflect.ValueOf(values[sf.name]))
	}

	return c
}

// HasHTTP reports whether c is a variant with an HTTP source field: a
// collector that reads over the network, such as a cloud's metadata
// service, as well as the host.
func HasHTTP(
	c any,
) bool {
	v := reflect.ValueOf(c)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return false
	}
	f, ok := v.Elem().Type().FieldByName("HTTP")

	return ok && f.Type == httpType
}

// CheckSources returns an error when c is a pointer to a struct with a
// source field of the wrong type, which Wire cannot set.
func CheckSources(
	c any,
) error {
	v := reflect.ValueOf(c)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return nil
	}
	s := v.Elem()
	for _, sf := range sourceFields {
		f := s.FieldByName(sf.name)
		if f.IsValid() && (f.Type() != sf.typ || !f.CanSet()) {
			return fmt.Errorf("%s.%s is %s, not %s", s.Type(), sf.name, f.Type(), sf.typ)
		}
	}

	return nil
}
