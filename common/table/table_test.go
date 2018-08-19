// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package table

import (
	"log"
	"testing"
)

func TestTable(t *testing.T) {
	table := NewTable(2, 3)
	table.HasHeading()
	table.Fill("heading 1", "heading 2")
	table.SpaceAllColumns()
	table.NoRowBorders()
	table.Fill("a", "b", "c", "d")
	log.Println("\n" + table.Format().String())
}
