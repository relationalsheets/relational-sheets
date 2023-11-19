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
	"testing"
)

func setupFormulasDB() func() {
	Open()
	InitSheetsTables()
	conn.MustExec("CREATE SCHEMA IF NOT EXISTS test")
	conn.MustExec(
		`CREATE TABLE IF NOT EXISTS test.foo (
			bar INT
			, baz FLOAT
		)`)
	conn.MustExec(
		`INSERT INTO test.foo VALUES
			(1, 2)
			, (3, 4)
			, (5, 6)
		`)
	CreateAggregates()
	loadTables()

	return func() {
		conn.MustExec("DROP TABLE IF EXISTS test.foo CASCADE")
		Check(conn.Close())
	}
}

func checkFormulas(t *testing.T, sheet Sheet, formulasAndValues map[string]string) {
	for formula, expected := range formulasAndValues {
		actual, err := sheet.evalFormula("=" + formula)
		if err != nil {
			t.Errorf("%s: %s", formula, err)
		} else if actual.Value != expected {
			t.Errorf("%s: %s != %s", formula, actual.Value, expected)
		}
	}
}

func checkFormulaErrors(t *testing.T, sheet Sheet, formulasAndErrors map[string]string) {
	for formula, expected := range formulasAndErrors {
		_, err := sheet.evalFormula("=" + formula)
		if err == nil {
			t.Errorf("Unexpected success: %s", formula)
		} else if err.Error() != expected {
			t.Errorf("Wrong error: %s != %s", err.Error(), expected)
		}
	}
}

func TestEvalWithLiterals(t *testing.T) {
	formulasAndValues := map[string]string{
		"2":                           "2",
		"(2)":                         "2",
		"2+3":                         "5",
		"2+2.5":                       "4.5",
		"2*3":                         "6",
		"2+2+2":                       "6",
		"2+(2+3)":                     "7",
		"(2+2)+3":                     "7",
		"2+2*3":                       "8",
		"(2+2)*3":                     "12",
		"2+(2+2)+2":                   "8",
		"IF(1=1,1,2)":                 "1",
		"IF(1=0,1,2)":                 "2",
		"AVERAGE(10,1)":               "5.5",
		"REGEXMATCH(\"foo\",\"bar\")": "false",
		"REGEXMATCH(\"foo\",\"f.*\")": "true",
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
						Formula: "1",
					},
					{
						Cell:    Cell{Value: "2", NotNull: true},
						Formula: "2",
					},
				},
			},
			{
				Name: "B",
				Cells: []SheetCell{
					{
						Cell:    Cell{Value: "3", NotNull: true},
						Formula: "3",
					},
					{},
				},
			},
		},
	}
	formulasAndValues := map[string]string{
		"A1":                        "1",
		"-A1":                       "-1",
		"A2":                        "2",
		"SUM(A1:A2)":                "3",
		"A1+B1":                     "4",
		"A1+4":                      "5",
		"SUM(A1:A2,B1:B2)":          "6",
		"MAX(A1:A2,B1:B2)":          "3",
		"MIN(A1:A2,B1:B2)":          "1",
		"SUM(A1:A2,-A1)":            "2",
		"PRODUCT(A1:A2,B1)":         "6",
		"AVERAGE(A1:A2,B1)":         "2",
		"COUNTIF(A1:A2,\"<1\")":     "0",
		"COUNTIF(A1:A2,\">1\")":     "1",
		"COUNTIF(A1:A2,\">=1\")":    "2",
		"SUMIF(A1:A2,\">1\")":       "2",
		"SUMIF(B1:B1,\">1\",A1:A1)": "1",
		"AVERAGEIF(A1:A2,\">=1\")":  "1.5",
	}
	checkFormulas(t, sheet, formulasAndValues)
}

func TestEvalWithDB(t *testing.T) {
	teardown := setupFormulasDB()
	defer teardown()

	sheet := Sheet{}
	sheet.SetTable("test.foo")
	sheet.LoadRows(100, 0)

	formulasAndValues := map[string]string{
		"bar1":                              "1",
		"test.foo.bar1":        "1",
		"SUM(bar1:bar1)":                    "1",
		"SUM(bar1:bar3)":                    "9",
		"SUM(baz1:baz3)":                    "12",
		"SUM(baz:baz)":                      "12",
		"SUM(bar1:bar1,2)":                  "3",
		"SUM(bar1:bar1,1+2)":                "4",
		"SUM(bar1:bar3,baz1:baz3)":          "21",
		"MAX(bar1:bar3)":                    "5",
		"MIN(bar1:bar3)":                    "1",
		"PRODUCT(bar1:bar3)":                "15",
		"AVERAGE(bar1:bar3)":                "3",
		"AVERAGE(bar1:bar3,7)":              "4",
		"COUNTIF(bar:bar,\">0\")":           "3",
		"COUNTIF(bar1:bar2,\">0\")":         "2",
		"COUNTIF(bar:bar,\"<2\")":           "1",
		"SUMIF(bar:bar,\">0\",baz:baz)":     "12",
		"SUMIF(bar:bar,\"<0\",baz:baz)":     "0",
		"SUMIF(bar:bar,\"=3\",baz:baz)":     "4",
		"AVERAGEIF(bar:bar,\">0\",baz:baz)": "4",
	}
	checkFormulas(t, sheet, formulasAndValues)
}

func TestRoundTripSerialization(t *testing.T) {
	formulas := []string{
		"=SUM(A:A)",
		"1+(2)",
		"1+(2+3)",
		"=REGEXMATCH(A,\"foo\")",
	}
	for _, formula := range formulas {
		tokens := parseFormula(formula)
		roundTripped := toFormula(tokens)
		if roundTripped != formula {
			t.Errorf("%s != %s parsed as: %v", roundTripped, formula, tokens)
		}
	}
}

func TestParsingErrors(t *testing.T) {
	formulasAndErrors := map[string]string{
		"1+": "missing second operand for +",
	}
	checkFormulaErrors(t, Sheet{}, formulasAndErrors)
}
