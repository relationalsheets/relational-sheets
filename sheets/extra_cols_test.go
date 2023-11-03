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
	sheet.SetTable("db_interface_test.customers")
	sheet.AddColumn("")
	sheet.AddColumn("")
	sheet.AddColumn("")
	for i := range sheet.ExtraCols[0].Cells {
		iStr := strconv.Itoa(i)
		sheet.SetCell(0, i, iStr)
		sheet.SetCell(1, i, iStr)
	}
	sheet.FillColumnDown(2, 0, "=A1+B1")
	for i, sheetCell := range sheet.ExtraCols[2].Cells {
		value := sheetCell.Cell.Value
		expectedValue := strconv.Itoa(2*i)
		if value != expectedValue {
			t.Errorf("row %d: %s != %s", i, value, expectedValue)
		}
	}
}
