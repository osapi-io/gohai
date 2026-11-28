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

package schemas_test

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/avfs/avfs/vfs/osfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/schemas"
)

type SchemaPublicTestSuite struct {
	suite.Suite
}

// separatorRE is the row under a markdown table's header.
var separatorRE = regexp.MustCompile(`^\|[\s:|-]+\|$`)

// mappingRows is the number of data rows in the tables of md: each row that
// is not a header or the separator under it.
func mappingRows(
	md string,
) int {
	lines := strings.Split(md, "\n")
	rows := 0
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") || separatorRE.MatchString(l) {
			continue
		}
		if i+1 < len(lines) && separatorRE.MatchString(strings.TrimSpace(lines[i+1])) {
			continue
		}
		rows++
	}

	return rows
}

func (s *SchemaPublicTestSuite) TestSchemaJSON() {
	tests := []struct {
		name         string
		validateFunc func()
	}{
		{
			name: "is a JSON Schema with a Facts definition",
			validateFunc: func() {
				var schema struct {
					Defs map[string]json.RawMessage `json:"$defs"`
				}
				s.Require().NoError(json.Unmarshal(schemas.SchemaJSON, &schema))
				s.Contains(schema.Defs, "Facts")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc()
		})
	}
}

func (s *SchemaPublicTestSuite) TestFieldMappingRows() {
	repo := osfs.New()
	mapping, err := repo.ReadFile(repo.Join("field-mapping.md"))
	s.Require().NoError(err)
	rows := strconv.Itoa(mappingRows(string(mapping)))

	tests := []struct {
		name string
		page string
		// stated finds the row count that page states.
		stated       string
		validateFunc func(m [][]byte)
	}{
		{
			name:   "schemas/README.md states the row count of field-mapping.md",
			page:   "README.md",
			stated: `\((\d+) rows\)`,
			validateFunc: func(m [][]byte) {
				s.Require().NotNil(m)
				s.Equal(rows, string(m[1]))
			},
		},
		{
			name:   "CONTRIBUTING.md states the row count of field-mapping.md",
			page:   repo.Join("..", "CONTRIBUTING.md"),
			stated: `It has (\d+) rows\.`,
			validateFunc: func(m [][]byte) {
				s.Require().NotNil(m)
				s.Equal(rows, string(m[1]))
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			page, err := repo.ReadFile(tt.page)
			s.Require().NoError(err)
			re, err := regexp.Compile(tt.stated)
			s.Require().NoError(err)
			tt.validateFunc(re.FindSubmatch(page))
		})
	}
}

func TestSchemaPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaPublicTestSuite))
}
