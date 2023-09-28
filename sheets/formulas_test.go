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
