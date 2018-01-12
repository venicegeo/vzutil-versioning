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

import "strings"

type Table struct {
	table      [][]string
	spaceColum []bool
	nextRow    int
	nextColumn int
	drawBorder bool
}

func NewTable(width, height int) *Table {
	table := &Table{[][]string{}, []bool{}, 0, 0, true}
	for i := 0; i < height; i++ {
		temp := []string{}
		for j := 0; j < width; j++ {
			temp = append(temp, "")
		}
		table.table = append(table.table, temp)
	}
	for i := 0; i < width; i++ {
		table.spaceColum = append(table.spaceColum, false)
	}
	return table
}

func (t *Table) SpaceColumn(i int) *Table {
	t.spaceColum[i] = true
	return t
}
func (t *Table) UnspaceColumn(i int) *Table {
	t.spaceColum[i] = false
	return t
}
func (t *Table) SpaceAllColumns() *Table {
	for i := 0; i < len(t.spaceColum); i++ {
		t.spaceColum[i] = true
	}
	return t
}
func (t *Table) UnspaceAllColumns() *Table {
	for i := 0; i < len(t.spaceColum); i++ {
		t.spaceColum[i] = false
	}
	return t
}
func (t *Table) Fill(toFill string) {
	t.table[t.nextRow][t.nextColumn] = toFill
	if t.nextColumn == len(t.table[0])-1 {
		t.nextRow++
		t.nextColumn = 0
	} else {
		t.nextColumn++
	}
}
func (t *Table) Format() *Table {
	if len(t.table) == 0 {
		return t
	}
	for c := 0; c < len(t.table[0]); c++ {
		max := 0
		for r := 0; r < len(t.table); r++ {
			max = t.max(max, len(t.table[r][c]))
		}
		for r := 0; r < len(t.table); r++ {
			for len(t.table[r][c]) < max {
				t.table[r][c] += " "
			}
		}
	}
	return t
}
func (t *Table) DrawBorders() *Table {
	t.drawBorder = true
	return t
}
func (t *Table) NoBorders() *Table {
	t.drawBorder = false
	return t
}
func (t *Table) String() string {
	res := ""
	pipe := "|"
	if !t.drawBorder {
		pipe = ""
	}
	for r := 0; r < len(t.table); r++ {
		line := ""
		for c := 0; c < len(t.table[r]); c++ {
			if t.spaceColum[c] {
				line += " " + t.table[r][c] + " " + pipe
			} else {
				line += t.table[r][c] + pipe
			}
		}
		res += pipe + line + "\n"
		if t.drawBorder {
			for i := 0; i < len(line)+1; i++ {
				res += "-"
			}
			res += "\n"
		}
	}
	temp := ""
	tmpS := len(strings.SplitN(res, "\n", 2)[0])
	if tmpS == 0 {
		tmpS = 10
	}
	for i := 0; i < tmpS; i++ {
		temp += "_"
	}
	return temp + "\n" + res + temp
}
func (t *Table) max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
