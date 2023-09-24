package main

func EvalFormula(s Sheet, formula string) SheetCell {
	// TODO: support non-literal formulas
	return SheetCell{Cell{formula, false}, formula}
}
