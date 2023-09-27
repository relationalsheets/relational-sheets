package sheets

import (
	"errors"
	"fmt"
	"github.com/xuri/efp"
	"log"
	"strconv"
	"strings"
)

func parseFormula(formula string) []efp.Token {
	if !strings.HasPrefix(formula, "=") {
		return []efp.Token{literalToken(formula)}
	}
	parser := efp.ExcelParser()
	parsed := parser.Parse(formula[1:])
	log.Println(parser.PrettyPrint())
	return parsed
}

func literalToken(val string) efp.Token {
	return efp.Token{val, efp.TokenTypeOperand, efp.TokenSubTypeNumber}
}

func parseRange(r string) (string, int, int, error) {
	if strings.Contains(r, ":") {
		split := strings.Split(r, ":")
		if len(split) != 2 {
			return "", 0, 0, errors.New("invalid range")
		}
		colName1, start, err := parseColumnAndIndex(split[0])
		if err != nil {
			return "", 0, 0, err
		}
		colName2, end, err := parseColumnAndIndex(split[1])
		if err != nil {
			return "", 0, 0, err
		}
		if colName1 != colName2 {
			return "", 0, 0, errors.New("ranges must be for a single column")
		}
		return colName1, start, end, nil
	}
	colName, index, err := parseColumnAndIndex(r)
	return colName, index, index, err
}

func parseColumnAndIndex(r string) (string, int, error) {
	colName := strings.TrimRight(r, "0123456789")
	index, err := strconv.Atoi(r[len(colName):])
	if err != nil {
		return "", 0, errors.New("invalid row index")
	}
	return colName, index, nil
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
		colName, index, err := parseColumnAndIndex(token.TValue)
		if err != nil {
			return "", err
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

func (s *Sheet) evalFunction(fName string, arguments [][]efp.Token) (string, error) {
	log.Printf("Evaluating: %s(%+v)", fName, arguments)
	fName = strings.ToUpper(fName)
	if fName == "SUM" {
		sum := float64(0)
		for _, arg := range arguments {
			argVal := float64(0)
			if len(arg) == 1 && arg[0].TSubType == efp.TokenSubTypeRange {
				colName, start, end, err := parseRange(arg[0].TValue)
				if err != nil {
					return "", err
				}

				query := fmt.Sprintf(
					"SELECT SUM(sq.val) FROM (SELECT \"%s\" AS val FROM %s LIMIT $1 OFFSET $2) sq",
					colName,
					s.TableFullName())
				log.Printf("Executing %s (%d, %d)", query, end-start+1, start)
				row := conn.QueryRow(query, end-start+1, start)
				err = row.Scan(&argVal)
				Check(err)
			} else {
				argValStr, err := s.evalTokens(arg)
				if err != nil {
					return "", nil
				}
				argVal, err = strconv.ParseFloat(argValStr, 64)
				if err != nil {
					return "", errors.New("invalid non-numeric argument to SUM: " + argValStr)
				}
			}

			sum += argVal
		}

		return fmt.Sprintf("%f", sum), nil
	}

	return "", errors.New("unsupported function: " + fName)
}

func (s *Sheet) evalTokens(tokens []efp.Token) (string, error) {
	if len(tokens) == 0 {
		return "", errors.New("empty expression")
	}

	if len(tokens) == 1 {
		return s.evalToken(tokens[0])
	}

	// Parentheses
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

	// Functions
	for i, t := range tokens {
		if t.TType == efp.TokenTypeFunction && t.TSubType == efp.TokenSubTypeStart {
			arguments := [][]efp.Token{{}}
			indent, end := 1, 0
			for j, nt := range tokens[i+1:] {
				if nt.TSubType == efp.TokenSubTypeStart {
					indent++
				}
				if nt.TSubType == efp.TokenSubTypeStop {
					indent--
				}
				if indent == 0 && nt.TType == efp.TokenTypeFunction && nt.TSubType == efp.TokenSubTypeStop {
					end = j
					break
				}
				if indent == 1 && nt.TType == efp.TokenTypeArgument {
					arguments = append(arguments, []efp.Token{})
				} else {
					arguments[len(arguments)-1] = append(arguments[len(arguments)-1], nt)
				}
			}
			if end == 0 {
				return "", errors.New("unmatched parentheses for function")
			}

			val, err := s.evalFunction(t.TValue, arguments)
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

	// Prefix
	for _, t := range tokens {
		if t.TType == efp.TokenTypeOperatorPrefix {
			// TODO
		}
	}

	// Multiplication/Division
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

	// Addition/Subtraction
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
