package sheets

import (
	"testing"
)

func setupDB() func() {
	Open()
	conn.MustExec("CREATE SCHEMA IF NOT EXISTS db_interface_test")
	conn.MustExec(
		`CREATE TABLE db_interface_test.foo (
			bar INT
			, baz FLOAT
		)`)
	conn.MustExec(
		`INSERT INTO db_interface_test.foo VALUES
			(1, 2)
			, (3, 4)
			, (5, 6)
		`)
	LoadTables()

	return func() {
		conn.MustExec("DROP TABLE IF EXISTS db_interface_test.foo")
		Check(conn.Close())
	}
}

func checkFormulas(t *testing.T, sheet Sheet, formulasAndValues map[string]string) {
	for formula, expected := range formulasAndValues {
		actual, err := sheet.EvalFormula("=" + formula)
		if err != nil {
			t.Fatalf("%s: %s", formula, err)
		}
		if actual.Value != expected {
			t.Fatalf("%s: %s != %s", formula, actual.Value, expected)
		}
	}
}

func TestEvalWithLiterals(t *testing.T) {
	formulasAndValues := map[string]string{
		"2":           "2",
		"(2)":         "2",
		"2+3":         "5",
		"2+2.5":       "4.5",
		"2*3":         "6",
		"2+2+2":       "6",
		"2+(2+3)":     "7",
		"(2+2)+3":     "7",
		"2+2*3":       "8",
		"(2+2)*3":     "12",
		"2+(2+2)+2":   "8",
		"IF(1=1,1,2)": "1",
		"IF(1=0,1,2)": "2",
	}
	checkFormulas(t, Sheet{}, formulasAndValues)
}

func TestEvalWithExtraCols(t *testing.T) {
	sheet := Sheet{
		ExtraCols: []SheetColumn{
			{
				Name: "A",
				Cells: []SheetCell{
					{
						Cell:    Cell{Value: "1", NotNull: true},
						Formula: "",
					},
					{
						Cell:    Cell{Value: "2", NotNull: true},
						Formula: "",
					},
				},
			},
			{
				Name: "B",
				Cells: []SheetCell{
					{
						Cell:    Cell{Value: "3", NotNull: true},
						Formula: "",
					},
					{},
				},
			},
		},
	}
	formulasAndValues := map[string]string{
		"A1":               "1",
		"-A1":              "-1",
		"A2":               "2",
		"SUM(A1:A2)":       "3",
		"A1+B1":            "4",
		"A1+4":             "5",
		"SUM(A1:A2,B1:B2)": "6",
		"MAX(A1:A2,B1:B2)": "3",
		"MIN(A1:A2,B1:B2)": "1",
		"SUM(A1:A2,-A1)":   "2",
	}
	checkFormulas(t, sheet, formulasAndValues)
}

func TestEvalWithDB(t *testing.T) {
	teardown := setupDB()
	defer teardown()

	sheet := Sheet{}
	sheet.SetTable("db_interface_test.foo")

	formulasAndValues := map[string]string{
		"SUM(bar1:bar1)":           "1",
		"SUM(bar1:bar3)":           "9",
		"SUM(baz1:baz3)":           "12",
		"SUM(bar1:bar1,2)":         "3",
		"SUM(bar1:bar1,1+2)":       "4",
		"SUM(bar1:bar3,baz1:baz3)": "21",
		"MAX(bar1:bar3)":           "5",
		"MIN(bar1:bar3)":           "1",
	}
	checkFormulas(t, sheet, formulasAndValues)
}
