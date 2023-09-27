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

func (s *Sheet) infixOperator(t1, t2 efp.Token, operand string) (string, error) {
	a, err := s.evalToken(t1)
	if err != nil {
		return "", err
	}
	b, err := s.evalToken(t2)
	if err != nil {
		return "", err
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
	switch operand {
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
	// TODO: handle non-literals
	return token.TValue, nil
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
