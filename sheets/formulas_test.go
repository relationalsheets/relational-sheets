package sheets

import "testing"

func TestEvalWithLiterals(t *testing.T) {
	sheet := Sheet{}
	formulasAndValues := map[string]string{
		"2":         "2",
		"(2)":       "2",
		"2+3":       "5",
		"2+2.5":     "4.500000",
		"2*3":       "6",
		"2+2+2":     "6",
		"2+(2+3)":   "7",
		"(2+2)+3":   "7",
		"2+2*3":     "8",
		"(2+2)*3":   "12",
		"2+(2+2)+2": "8",
	}
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
		"A2":               "2",
		"SUM(A1:A2)":       "3.000000",
		"A1+B1":            "4",
		"A1+4":             "5",
		"SUM(A1:A2,B1:B2)": "6.000000",
	}
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
