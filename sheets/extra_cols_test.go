// This file is part of Relational Sheets.
//
// Relational Sheets is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// Relational Sheets is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU General Public License along with Relational Sheets.
// If not, see https://www.gnu.org/licenses/agpl-3.0.html
package sheets

import (
	"strconv"
	"testing"
)

func TestDefaultColumnName(t *testing.T) {
	names := map[int]string{
		0:         "A",
		1:         "B",
		26:        "AA",
		27:        "AB",
		52:        "BA",
		26 * 26:   "ZA",
		27 * 26:   "AAA",
		27*26 + 1: "AAB",
	}
	for i, expectedName := range names {
		name := defaultColumnName(i)
		if name != expectedName {
			t.Errorf("%s != %s", name, expectedName)
		}
	}
}

func TestFillColumnDown(t *testing.T) {
	SetupTablesDB()
	defer teardownTablesDB()

	sheet := Sheet{RowCount: 10}
	sheet.SetTable("test.customers")
	sheet.AddColumn("")
	sheet.AddColumn("")
	sheet.AddColumn("")
	for i := range sheet.ExtraCols[0].Cells {
		iStr := strconv.Itoa(i)
		sheet.SetCell(0, i, iStr)
		sheet.SetCell(1, i, iStr)
	}

	sheet.FillColumnDown(2, 1, "=A1+B1")
	cells := sheet.ExtraCols[2].Cells
	
	value := cells[0].Cell.Value
	if value != "" {
		t.Errorf("first cell should have been skipped")
	}
	for i, sheetCell := range cells[1:] {
		value := sheetCell.Cell.Value
		expectedValue := strconv.Itoa(2*i)
		if value != expectedValue {
			t.Errorf("row %d: %s != %s", i, value, expectedValue)
		}
	}
}
