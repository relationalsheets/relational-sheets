package main

func (s *Sheet) EvalFormula(formula string) SheetCell {
	// TODO: support non-literal formulas
	return SheetCell{Cell{formula, formula != ""}, formula}
}
