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

// Package validate provides JSON Schema validation for gohai output.
package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/osapi-io/gohai/schemas"
)

// schemaResourceURL is the URL that the compiler gives the schema.
const schemaResourceURL = "gohai.schema.json"

// CompileSchema compiles the JSON Schema schemaJSON, such as the embedded
// schemas.SchemaJSON, and returns it ready for validation.
func CompileSchema(
	schemaJSON []byte,
) (*jsonschema.Schema, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	// AddResource fails only for a URL that is not valid or that is added
	// two times. Compile then cannot find the schema, and its error has
	// the error of AddResource.
	c := jsonschema.NewCompiler()
	addErr := c.AddResource(schemaResourceURL, schemaDoc)
	sch, err := c.Compile(schemaResourceURL)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", errors.Join(addErr, err))
	}

	return sch, nil
}

// JSON validates raw JSON bytes against the embedded gohai schema.
// Returns nil if valid, or a structured validation error.
func JSON(
	data []byte,
) error {
	return JSONWith(schemas.SchemaJSON, data)
}

// JSONWith validates raw JSON bytes against the JSON Schema schemaJSON.
// Returns nil if valid, or a structured validation error.
func JSONWith(
	schemaJSON []byte,
	data []byte,
) error {
	sch, err := CompileSchema(schemaJSON)
	if err != nil {
		return err
	}

	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	return sch.Validate(instance)
}
