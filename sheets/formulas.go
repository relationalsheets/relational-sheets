package sheets

import (
	"errors"
	"fmt"
	"github.com/xuri/efp"
	"log"
	"math"
	"strconv"
	"strings"
)

type Token struct {
	efp.Token
	IsNumeric bool
	TFloat    float64
}

func CreateAggregates() {
	conn.MustExec(
		`CREATE OR REPLACE AGGREGATE db_interface.mul(numeric) (
			STYPE = numeric,
			INITCOND = 1,
			SFUNC = numeric_mul,
			COMBINEFUNC = numeric_mul,
			PARALLEL = SAFE
		)`)
}

func parseFormula(formula string) []Token {
	if !strings.HasPrefix(formula, "=") {
		return []Token{fromString(formula)}
	}
	parser := efp.ExcelParser()
	parsed := parser.Parse(formula[1:])
	log.Println(parser.PrettyPrint())

	tokens := make([]Token, len(parsed))
	for i, token := range parsed {
		if token.TSubType == efp.TokenSubTypeNumber {
			float, err := strconv.ParseFloat(token.TValue, 64)
			tokens[i] = Token{token, err == nil, float}
		} else {
			tokens[i] = Token{Token: token}
		}
	}
	return tokens
}

func fromString(val string) Token {
	float, err := strconv.ParseFloat(val, 64)
	return Token{
		efp.Token{val, efp.TokenTypeOperand, efp.TokenSubTypeNumber},
		err == nil,
		float,
	}
}

func fromFloat(val float64) Token {
	return Token{
		efp.Token{formatFloat(val), efp.TokenTypeOperand, efp.TokenSubTypeNumber},
		true,
		val,
	}
}

func formatFloat(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
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

func (s *Sheet) infixOperator(t1, t2 Token, operator string) (Token, error) {
	a, err := s.evalToken(t1)
	if err != nil {
		return Token{}, err
	}
	b, err := s.evalToken(t2)
	if err != nil {
		return Token{}, err
	}

	var f float64
	switch operator {
	case "*":
		f = a.TFloat * b.TFloat
	case "/":
		f = a.TFloat / b.TFloat
	case "+":
		f = a.TFloat + b.TFloat
	case "-":
		f = a.TFloat - b.TFloat
	default:
		return Token{}, errors.New("invalid infix operator")
	}
	return fromFloat(f), nil
}

func (s *Sheet) evalToken(token Token) (Token, error) {
	if token.TType != efp.TokenTypeOperand {
		return Token{}, errors.New("not an operand")
	}
	if token.TSubType == efp.TokenSubTypeNumber {
		return token, nil
	}
	if token.TSubType == efp.TokenSubTypeRange {
		colName, index, err := parseColumnAndIndex(token.TValue)
		if err != nil {
			return Token{}, err
		}
		index-- // formulas index from 1
		if index < 0 {
			return Token{}, errors.New("negative indices not allowed")
		}
		col, ok := s.table.Cols[colName]
		if ok {
			if index >= s.table.RowCount {
				return Token{}, errors.New("row index out of range")
			}
			return fromString(col.Cells[index].Value), nil
		}
		for _, extraCol := range s.ExtraCols {
			if colName == extraCol.Name {
				if index >= len(extraCol.Cells) {
					// Not an error to reference beyond the sheet
					return Token{}, nil
				}
				return fromString(extraCol.Cells[index].Value), nil
			}
		}
		return Token{}, errors.New("invalid column name")
	}
	return Token{}, errors.New("invalid formula")
}

type SQLAndGoFunc struct {
	sqlName    string
	sqlCast    string
	goFunc     func(float64, float64) float64
	initialVal float64
}

var associativeFuncs = map[string]SQLAndGoFunc{
	"SUM": {
		"SUM",
		"",
		func(a, b float64) float64 {
			return a + b
		},
		0,
	},
	"MAX": {
		"MAX",
		"",
		math.Max,
		0,
	},
	"MIN": {
		"MIN",
		"",
		math.Min,
		math.MaxFloat64,
	},
	"PRODUCT": {
		"db_interface.mul",
		"numeric",
		func(a, b float64) float64 {
			return a * b
		},
		1,
	},
}

func (s *Sheet) evalAssociativeFunc(fDefs SQLAndGoFunc, arguments [][]Token) (Token, error) {
	val := fDefs.initialVal

	for _, arg := range arguments {
		argVal := fDefs.initialVal

		if len(arg) == 1 && arg[0].TSubType == efp.TokenSubTypeRange {
			colName, start, end, err := parseRange(arg[0].TValue)
			if err != nil {
				return Token{}, err
			}
			_, colExists := s.table.Cols[colName]
			if colExists {
				colExpression := "\"" + colName + "\""
				if fDefs.sqlCast != "" {
					colExpression = fmt.Sprintf("CAST(%s AS %s)", colExpression, fDefs.sqlCast)
				}
				query := fmt.Sprintf(
					"SELECT %s(sq.val) FROM (SELECT %s AS val FROM %s LIMIT $1 OFFSET $2) sq",
					fDefs.sqlName,
					colExpression,
					s.TableFullName())
				log.Printf("Executing %s (%d, %d)", query, end-start+1, start-1)
				row := conn.QueryRow(query, end-start+1, start-1)
				err = row.Scan(&argVal)
				Check(err)
			} else {
				found := false
				for _, col := range s.ExtraCols {
					if col.Name == colName {
						for _, cell := range col.Cells {
							if cell.NotNull {
								cellVal, err := strconv.ParseFloat(cell.Value, 64)
								if err != nil {
									return Token{}, err
								}
								argVal = fDefs.goFunc(argVal, cellVal)
							}
						}
						found = true
						break
					}
				}
				if !found {
					return Token{}, errors.New("no column named " + colName)
				}
			}
		} else {
			argValToken, err := s.evalTokens(arg)
			if err != nil {
				return Token{}, nil
			}
			if !argValToken.IsNumeric {
				return Token{}, errors.New("invalid non-numeric argument to SUM: " + argValToken.TValue)
			}
			argVal = argValToken.TFloat
		}

		val = fDefs.goFunc(val, argVal)
	}

	return fromFloat(val), nil
}

func (s *Sheet) evalLogicalExpression(tokens []Token) (bool, error) {
	log.Printf("Evaluating logical expression: %+v", tokens)

	operator := ""
	first, second := []Token{}, []Token{}
	for _, t := range tokens {
		if t.TSubType == efp.TokenSubTypeLogical {
			if operator != "" {
				return false, errors.New("multiple logical operators in single expression")
			}
			operator = t.TValue
			continue
		}
		if operator == "" {
			first = append(first, t)
		} else {
			second = append(second, t)
		}
	}
	if operator == "" {
		return false, errors.New("not a logical expression")
	}

	firstVal, err := s.evalTokens(first)
	if err != nil {
		return false, err
	}
	secondVal, err := s.evalTokens(second)
	if err != nil {
		return false, err
	}

	switch operator {
	case "=":
		numericEqual := firstVal.IsNumeric && secondVal.IsNumeric && firstVal.TFloat == secondVal.TFloat
		stringEqual := !firstVal.IsNumeric && !secondVal.IsNumeric && firstVal.TValue == secondVal.TValue
		return numericEqual || stringEqual, nil
	default:
		return false, errors.New("unsupported logical operator: " + operator)
	}
}

func (s *Sheet) evalIf(arguments [][]Token) (Token, error) {
	if len(arguments) != 3 {
		return Token{}, errors.New("wrong number of arguments for IF")
	}

	condition, err := s.evalLogicalExpression(arguments[0])
	if err != nil {
		return Token{}, err
	}

	if condition {
		return s.evalTokens(arguments[1])
	} else {
		return s.evalTokens(arguments[2])
	}
}

func (s *Sheet) evalFunction(fName string, arguments [][]Token) (Token, error) {
	fName = strings.ToUpper(fName)
	log.Printf("Evaluating: %s(%+v)", fName, arguments)

	fDefs, isAssociativeFunc := associativeFuncs[fName]
	if isAssociativeFunc {
		return s.evalAssociativeFunc(fDefs, arguments)
	}

	if fName == "IF" {
		return s.evalIf(arguments)
	}

	return Token{}, errors.New("unsupported function: " + fName)
}

func (s *Sheet) evalTokens(tokens []Token) (Token, error) {
	if len(tokens) == 0 {
		return Token{}, errors.New("empty expression")
	}

	if len(tokens) == 1 {
		return s.evalToken(tokens[0])
	}

	// Prefix
	if tokens[0].TType == efp.TokenTypeOperatorPrefix {
		if tokens[0].TValue != "-" {
			return Token{}, errors.New("invalid prefix operator " + tokens[0].TValue)
		}

		val, err := s.evalTokens(tokens[1:])
		if err != nil {
			return Token{}, err
		}

		if !val.IsNumeric {
			return Token{}, errors.New("attempting to negate non-numeric value")
		}

		return fromFloat(-val.TFloat), nil
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
					end = i + 1 + j
					break
				}
			}
			if end == 0 {
				return Token{}, errors.New("unmatched parentheses")
			}

			val, err := s.evalTokens(tokens[i+1 : end])
			log.Printf("Subexpression: %d:%d", i+1, end)
			if err != nil {
				return Token{}, err
			}

			tokens[i] = val
			for j, nt := range tokens[end+1:] {
				tokens[i+j+1] = nt
			}
			tokens = tokens[:len(tokens)+i-end]
			log.Printf("After evaluating subexpression: %+v", tokens)
			return s.evalTokens(tokens)
		}
	}

	// Functions
	for i, t := range tokens {
		if t.TType == efp.TokenTypeFunction && t.TSubType == efp.TokenSubTypeStart {
			arguments := [][]Token{{}}
			indent, end := 1, 0
			for j, nt := range tokens[i+1:] {
				if nt.TSubType == efp.TokenSubTypeStart {
					indent++
				}
				if nt.TSubType == efp.TokenSubTypeStop {
					indent--
				}
				if indent == 0 && nt.TType == efp.TokenTypeFunction && nt.TSubType == efp.TokenSubTypeStop {
					end = i + 1 + j
					break
				}
				if indent == 1 && nt.TType == efp.TokenTypeArgument {
					arguments = append(arguments, []Token{})
				} else {
					arguments[len(arguments)-1] = append(arguments[len(arguments)-1], nt)
				}
			}
			if end == 0 {
				return Token{}, errors.New("unmatched parentheses for function")
			}

			val, err := s.evalFunction(t.TValue, arguments)
			if err != nil {
				return Token{}, err
			}

			tokens[i] = val
			for j, nt := range tokens[end+1:] {
				tokens[i+j+1] = nt
			}
			tokens = tokens[:len(tokens)+i-end]
			log.Printf("After evaluating function: %+v", tokens)
			return s.evalTokens(tokens)
		}
	}

	// Multiplication/Division
	for i, t := range tokens {
		if t.TType == efp.TokenTypeOperatorInfix && (t.TValue == "*" || t.TValue == "/") {
			if i == 0 {
				return Token{}, errors.New("cannot start expression with infix operator")
			}
			val, err := s.infixOperator(tokens[i-1], tokens[i+1], t.TValue)
			if err != nil {
				return Token{}, err
			}

			tokens[i-1] = val
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
				return Token{}, errors.New("cannot start expression with infix operator")
			}
			val, err := s.infixOperator(tokens[i-1], tokens[i+1], t.TValue)
			if err != nil {
				return Token{}, err
			}

			tokens[i-1] = val
			for j, nt := range tokens[i+2:] {
				tokens[i+j] = nt
			}
			return s.evalTokens(tokens[:len(tokens)-2])
		}
	}

	return Token{}, errors.New("not implemented")
}

func (s *Sheet) EvalFormula(formula string) (SheetCell, error) {
	tokens := parseFormula(formula)
	token, err := s.evalTokens(tokens)
	if err != nil {
		return SheetCell{Cell{}, formula}, err
	}
	return SheetCell{Cell{token.TValue, token.TValue != ""}, formula}, nil
}
