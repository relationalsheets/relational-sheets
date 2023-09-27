package sheets

import (
	"errors"
	"fmt"
	"github.com/xuri/efp"
	"strconv"
	"strings"
)

func parseFormula(formula string) []efp.Token {
	if !strings.HasPrefix(formula, "=") {
		return []efp.Token{literalToken(formula)}
	}
	parser := efp.ExcelParser()
	return parser.Parse(formula[1:])
}

func literalToken(val string) efp.Token {
	return efp.Token{val, efp.TokenTypeOperand, efp.TokenSubTypeNumber}
}

func (s *Sheet) infixOperator(t1, t2 efp.Token, operator string) (string, error) {
	a, err := s.evalToken(t1)
	if err != nil {
		return "", err
	}
	b, err := s.evalToken(t2)
	if err != nil {
		return "", err
	}

	aInt, err := strconv.Atoi(a)
	bInt, err2 := strconv.Atoi(b)
	if err == nil && err2 == nil && (operator != "/" || aInt%bInt == 0) {
		var d int
		switch operator {
		case "*":
			d = aInt * bInt
		case "/":
			d = aInt / bInt
		case "+":
			d = aInt + bInt
		case "-":
			d = aInt - bInt
		default:
			return "", errors.New("invalid infix operator")
		}
		return strconv.Itoa(d), nil
	}

	aFloat, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return "", err
	}
	bFloat, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return "", err
	}

	var f float64
	switch operator {
	case "*":
		f = aFloat * bFloat
	case "/":
		f = aFloat / bFloat
	case "+":
		f = aFloat + bFloat
	case "-":
		f = aFloat - bFloat
	default:
		return "", errors.New("invalid infix operator")
	}
	return fmt.Sprintf("%f", f), nil
}

func (s *Sheet) evalToken(token efp.Token) (string, error) {
	if token.TType != efp.TokenTypeOperand {
		return "", errors.New("not an operand")
	}
	if token.TSubType == efp.TokenSubTypeNumber {
		return token.TValue, nil
	}
	if token.TSubType == efp.TokenSubTypeRange {
		if strings.Contains(token.TValue, ":") {
			// TODO
		}
		colName := strings.TrimRight(token.TValue, "0123456789")
		index, err := strconv.Atoi(token.TValue[len(colName):])
		if err != nil {
			return "", errors.New("invalid row index")
		}
		index-- // formulas index from 1
		if index < 0 {
			return "", errors.New("negative indices not allowed")
		}
		col, ok := s.table.Cols[colName]
		if ok {
			if index >= s.table.RowCount {
				return "", errors.New("row index out of range")
			}
			return col.Cells[index].Value, nil
		}
		for _, extraCol := range s.ExtraCols {
			if colName == extraCol.Name {
				if index >= len(extraCol.Cells) {
					// Not an error to reference beyond the sheet
					return "", nil
				}
				return extraCol.Cells[index].Value, nil
			}
		}
		return "", errors.New("invalid column name")
	}
	return "", errors.New("invalid formula")
}

func (s *Sheet) evalTokens(tokens []efp.Token) (string, error) {
	if len(tokens) == 0 {
		return "", errors.New("empty expression")
	}

	if len(tokens) == 1 {
		return s.evalToken(tokens[0])
	}

	for i, t := range tokens {
		if t.TType == efp.TokenTypeSubexpression && t.TSubType == efp.TokenSubTypeStart {
			indent, end := 1, 0
			for j, nt := range tokens[i+1:] {
				if nt.TSubType == efp.TokenSubTypeStart {
					indent++
				}
				if nt.TSubType == efp.TokenSubTypeStop {
					indent--
				}
				if indent == 0 && nt.TType == efp.TokenTypeSubexpression && nt.TSubType == efp.TokenSubTypeStop {
					end = j
					break
				}
			}
			if end == 0 {
				return "", errors.New("unmatched parentheses")
			}

			val, err := s.evalTokens(tokens[i+1 : end])
			if err != nil {
				return "", err
			}

			tokens[i] = literalToken(val)
			for j, nt := range tokens[end+1:] {
				tokens[i+j+1] = nt
			}
			return s.evalTokens(tokens[:len(tokens)+i-end-1])
		}
	}

	// TODO: handle functions

	for _, t := range tokens {
		if t.TType == efp.TokenTypeOperatorPrefix {
			// TODO
		}
	}

	for i, t := range tokens {
		if t.TType == efp.TokenTypeOperatorInfix && (t.TValue == "*" || t.TValue == "/") {
			if i == 0 {
				return "", errors.New("cannot start expression with infix operator")
			}
			val, err := s.infixOperator(tokens[i-1], tokens[i+1], t.TValue)
			if err != nil {
				return "", err
			}

			tokens[i-1] = literalToken(val)
			for j, nt := range tokens[i+2:] {
				tokens[i+j] = nt
			}
			return s.evalTokens(tokens[:len(tokens)-2])
		}
	}

	for i, t := range tokens {
		if t.TType == efp.TokenTypeOperatorInfix && (t.TValue == "+" || t.TValue == "-") {
			if i == 0 {
				return "", errors.New("cannot start expression with infix operator")
			}
			val, err := s.infixOperator(tokens[i-1], tokens[i+1], t.TValue)
			if err != nil {
				return "", err
			}

			tokens[i-1] = literalToken(val)
			for j, nt := range tokens[i+2:] {
				tokens[i+j] = nt
			}
			return s.evalTokens(tokens[:len(tokens)-2])
		}
	}

	return "", errors.New("not implemented")
}

func (s *Sheet) EvalFormula(formula string) (SheetCell, error) {
	tokens := parseFormula(formula)
	val, err := s.evalTokens(tokens)
	if err != nil {
		return SheetCell{Cell{}, formula}, err
	}
	return SheetCell{Cell{val, val != ""}, formula}, nil
}
