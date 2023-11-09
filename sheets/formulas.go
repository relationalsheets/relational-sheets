package sheets

import (
	"acb/db-interface/escape"
	"acb/db-interface/fkeys"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xuri/efp"
)

type Token struct {
	efp.Token
	IsNumeric bool
	IsBool    bool
	TFloat    float64
	TBool     bool
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
	//log.Println(parser.PrettyPrint())

	tokens := make([]Token, len(parsed))
	for i, token := range parsed {
		if token.TSubType == efp.TokenSubTypeNumber {
			tokens[i] = fromString(token.TValue)
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
		false,
		float,
		false,
	}
}

func fromFloat(val float64) Token {
	return Token{
		efp.Token{formatFloat(val), efp.TokenTypeOperand, efp.TokenSubTypeNumber},
		true,
		false,
		val,
		false,
	}
}

func fromBool(val bool) Token {
	return Token{
		efp.Token{fmt.Sprintf("%t", val), efp.TokenTypeOperand, efp.TokenSubTypeNumber},
		false,
		true,
		0,
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
		colName1, start, err := parseColumnAndIndex(split[0], 1)
		if err != nil {
			return "", 0, 0, err
		}
		colName2, end, err := parseColumnAndIndex(split[1], math.MaxInt)
		if err != nil {
			return "", 0, 0, err
		}
		if colName1 != colName2 {
			return "", 0, 0, errors.New("ranges must be for a single column")
		}
		return colName1, start, end, nil
	}
	colName, index, err := parseColumnAndIndex(r, 0)
	return colName, index, index, err
}

func unparseRange(colName string, start, end int) string {
	if start == end {
		return colName + strconv.Itoa(start)
	}
	return strings.Join([]string{
		colName,
		strconv.Itoa(start),
		":",
		colName,
		strconv.Itoa(end),
	}, "")
}

func parseColumnAndIndex(r string, defaultIndex int) (string, int, error) {
	colName := strings.TrimRight(r, "0123456789")
	if colName == r && defaultIndex > 0 {
		return colName, defaultIndex, nil
	}
	index, err := strconv.Atoi(r[len(colName):])
	if err != nil {
		return "", 0, errors.New("invalid row index in " + r)
	}
	return colName, index, nil
}

func toFormula(tokens []Token) string {
	if len(tokens) == 1 {
		return tokens[0].TValue
	}

	values := make([]string, len(tokens)+1)
	values[0] = "="
	for i, token := range tokens {
		if token.TType == efp.TokenTypeSubexpression && token.TSubType == efp.TokenSubTypeStart {
			values[i+1] = "("
		} else if token.TType == efp.TokenTypeFunction && token.TSubType == efp.TokenSubTypeStart {
			values[i+1] = token.TValue + "("
		} else if token.TSubType == efp.TokenSubTypeStop {
			values[i+1] = ")"
		} else if token.TSubType == efp.TokenSubTypeText {
			values[i+1] = "\"" + token.TValue + "\""
		} else {
			values[i+1] = token.TValue
		}
	}
	return strings.Join(values, "")
}

func translateTokens(tokens []Token, offset int) ([]Token, error) {
	newTokens := make([]Token, len(tokens))
	for i, token := range tokens {
		if token.TSubType == efp.TokenSubTypeRange {
			colName, start, end, err := parseRange(token.TValue)
			if err != nil {
				return []Token{}, err
			}
			rangeStr := unparseRange(colName, start+offset, end+offset)
			newTokens[i] = Token{
				Token: efp.Token{rangeStr, efp.TokenTypeOperand, efp.TokenSubTypeRange},
			}
		} else {
			newTokens[i] = token
		}
	}
	return newTokens, nil
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

func (s *Sheet) tableAndColIndex(colName string) (int, int, error) {
	split := strings.Split(colName, ".")
	if len(split) != 1 && len(split) != 3 {
		return -1, -1, errors.New("columns must be specified as <col> or <schema>.<table>.<col>")
	}
	if len(split) == 3 {
		tableName := split[0] + "." + split[1]
		tableIndex := slices.Index(s.TableNames, tableName)
		if tableIndex < 0 {
			return -1, -1, errors.New("no such table " + tableName)
		}
		for i, col := range s.OrderedCols(nil)[tableIndex] {
			if split[2] == col.Name {
				return tableIndex, i, nil
			}
		}
		return -1, -1, fmt.Errorf("no column %s on table %s", split[2], tableName)
	} else {
		for i, cols := range s.OrderedCols(nil) {
			for j, col := range cols {
				if split[0] == col.Name {
					return i, j, nil
				}
			}
		}
		for i, col := range s.ExtraCols {
			if split[0] == col.Name {
				return -1, i, nil
			}
		}
		return -1, -1, fmt.Errorf("no such column %s", split[0])
	}
}

func (s *Sheet) evalToken(token Token) (Token, error) {
	if token.TType != efp.TokenTypeOperand {
		return Token{}, errors.New("not an operand")
	}
	if token.TSubType == efp.TokenSubTypeNumber || token.TSubType == efp.TokenSubTypeLogical || token.TSubType == efp.TokenSubTypeText {
		return token, nil
	}
	if token.TSubType == efp.TokenSubTypeRange {
		colName, index, err := parseColumnAndIndex(token.TValue, 0)
		if err != nil {
			return Token{}, err
		}
		index-- // formulas index from 1
		if index < 0 {
			return Token{}, errors.New("negative indices not allowed")
		}

		tableIndex, colIndex, err := s.tableAndColIndex(colName)
		if err != nil {
			return Token{}, err
		}
		if tableIndex >= 0 {
			if index >= s.RowCount {
				return Token{}, fmt.Errorf("row index out of range: %d", index)
			}
			return fromString(s.Cells[tableIndex][colIndex][index].Value), nil
		} else {
			extraCol := s.ExtraCols[colIndex]
			if index >= len(extraCol.Cells) {
				// Not an error to reference beyond the sheet
				return Token{}, nil
			}
			return fromString(extraCol.Cells[index].Value), nil
		}
	}
	return Token{}, errors.New("invalid formula " + token.TValue)
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
			tableIndex, colIndex, err := s.tableAndColIndex(colName)
			if err != nil {
				return Token{}, err
			}
			if tableIndex >= 0 {
				alias, err := escape.MakeCast(colName, fDefs.sqlCast, "val")
				subquery, err := escape.MakeSelectStmt(
					s.TableNames,
					s.joins(),
					[]escape.SafeSQL{alias},
					[]escape.SafeSQL{},
					[]escape.SafeSQL{},
					true)
				query := fmt.Sprintf(
					"SELECT %s(sq.val) FROM (%s) sq",
					fDefs.sqlName,
					subquery)
				log.Printf("Executing %s (%d, %d)", query, end-start+1, start-1)
				row := conn.QueryRow(query, end-start+1, start-1)
				err = row.Scan(&argVal)
				Check(err)
			} else {
				for _, cell := range s.ExtraCols[colIndex].Cells {
					if cell.NotNull {
						cellVal, err := strconv.ParseFloat(cell.Value, 64)
						if err != nil {
							return Token{}, err
						}
						argVal = fDefs.goFunc(argVal, cellVal)
					}
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
	//log.Printf("Evaluating logical expression: %+v", tokens)

	if len(tokens) == 1 {
		if tokens[0].IsBool {
			return tokens[0].TBool, nil
		}
		return false, errors.New("not a logical expression")
	}

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

	if operator == "=" {
		numericEqual := firstVal.IsNumeric && secondVal.IsNumeric && firstVal.TFloat == secondVal.TFloat
		stringEqual := !firstVal.IsNumeric && !secondVal.IsNumeric && firstVal.TValue == secondVal.TValue
		return numericEqual || stringEqual, nil
	}

	if !firstVal.IsNumeric {
		return false, fmt.Errorf("non-numeric arugment to %s: %s", operator, firstVal.TValue)
	}
	if !secondVal.IsNumeric {
		return false, fmt.Errorf("non-numeric arugment to %s: %s", operator, secondVal.TValue)
	}

	switch operator {
	case ">":
		return firstVal.TFloat > secondVal.TFloat, nil
	case ">=":
		return firstVal.TFloat >= secondVal.TFloat, nil
	case "<":
		return firstVal.TFloat < secondVal.TFloat, nil
	case "<=":
		return firstVal.TFloat <= secondVal.TFloat, nil
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

func (s *Sheet) evalAverage(arguments [][]Token) (Token, error) {
	sum := 0.0
	count := 0

	for _, arg := range arguments {
		argVal := 0.0
		argCount := 0
		if len(arg) == 1 && arg[0].TSubType == efp.TokenSubTypeRange {
			colName, start, end, err := parseRange(arg[0].TValue)
			if err != nil {
				return Token{}, err
			}
			tableIndex, colIndex, err := s.tableAndColIndex(colName)
			if err != nil {
				return Token{}, err
			}
			if tableIndex >= 0 {
				alias, err := escape.MakeCast(colName, "", "val")
				if err != nil {
					return Token{}, err
				}
				subquery, err := escape.MakeSelectStmt(
					[]string{s.TableNames[tableIndex]},
					[]fkeys.ForeignKey{},
					[]escape.SafeSQL{alias},
					[]escape.SafeSQL{},
					[]escape.SafeSQL{},
					true)
				query := fmt.Sprintf("SELECT SUM(sq.val), COUNT(*) FROM (%s) sq", subquery)
				log.Printf("Executing %s (%d, %d)", query, end-start+1, start-1)
				row := conn.QueryRow(query, end-start+1, start-1)
				err = row.Scan(&argVal, &argCount)
				Check(err)
			} else {
				for _, cell := range s.ExtraCols[colIndex].Cells {
					if cell.NotNull {
						cellVal, err := strconv.ParseFloat(cell.Value, 64)
						if err != nil {
							return Token{}, err
						}
						argVal += cellVal
						argCount += 1
					}
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
			argCount = 1
		}

		sum += argVal
		count += argCount
	}

	return fromFloat(sum / float64(count)), nil
}

func (s *Sheet) evalRegexMatch(arguments [][]Token) (Token, error) {
	if len(arguments) != 2 {
		return Token{}, errors.New("wrong number of arguments for REGEXMATCH")
	}

	textToken, err := s.evalTokens(arguments[0])
	if err != nil {
		return Token{}, err
	}
	if textToken.IsNumeric {
		return Token{}, errors.New("only text can be searched with REGEXMATCH")
	}
	text := textToken.Token.TValue

	regexToken, err := s.evalTokens(arguments[1])
	if err != nil {
		return Token{}, err
	}
	if regexToken.IsNumeric {
		return Token{}, errors.New("invalid regex")
	}
	regex := regexToken.Token.TValue

	matched, err := regexp.MatchString(regex, text)
	if err != nil {
		return Token{}, err
	}

	return fromBool(matched), nil
}

func (s *Sheet) evalAggIf(fName string, arguments [][]Token) (Token, error) {
	if len(arguments) < 2 || len(arguments) > 3 {
		return Token{}, fmt.Errorf("wrong number of arguments for %s", fName)
	}
	if len(arguments[0]) != 1 || arguments[0][0].TSubType != efp.TokenSubTypeRange {
		return Token{}, errors.New("invalid range in SUMIF")
	}
	conditionRange := arguments[0][0].TValue
	criteriaToken, err := s.evalTokens(arguments[1])
	if err != nil {
		return Token{}, err
	}
	// TODO: interpret as = if no operator is specific
	criteria := criteriaToken.TValue
	sumRange := conditionRange
	if len(arguments) > 2 {
		if len(arguments[2]) != 1 || arguments[2][0].TSubType != efp.TokenSubTypeRange {
			return Token{}, errors.New("invalid range in SUMIF")
		}
		sumRange = arguments[2][0].TValue
	}
	conditionColName, start, end, err := parseRange(conditionRange)
	conditionTableIndex, conditionColIndex, err := s.tableAndColIndex(conditionColName)
	if err != nil {
		return Token{}, err
	}
	sumColName, sumStart, sumEnd, err := parseRange(sumRange)
	sumTableIndex, sumColIndex, err := s.tableAndColIndex(sumColName)
	if err != nil {
		return Token{}, err
	}
	if sumStart != start || sumEnd != end {
		return Token{}, fmt.Errorf("condition rage (%s) and sum range (%s) must be aligned", conditionRange, sumRange)
	}
	if conditionTableIndex != sumTableIndex && (conditionTableIndex < 0 || sumTableIndex < 0) {
		return Token{}, fmt.Errorf("cannot combine database and spreadsheet ranges in %s", fName)
	}

	sum := 0.0
	count := 0
	if conditionTableIndex >= 0 {
		alias, err := escape.MakeCast(sumColName, "", "val")
		if err != nil {
			return Token{}, err
		}
		filterClause, err := escape.MakeFilterClause(conditionColName, criteria)
		subquery, err := escape.MakeSelectStmt(
			s.TableNames,
			s.joins(),
			[]escape.SafeSQL{alias},
			[]escape.SafeSQL{filterClause},
			[]escape.SafeSQL{},
			true)
		query := fmt.Sprintf(
			"SELECT COALESCE(SUM(sq.val), 0), COUNT(*) FROM (%s) sq",
			subquery)
		log.Printf("Executing %s (%d, %d)", query, end-start+1, start-1)
		row := conn.QueryRow(query, end-start+1, start-1)
		err = row.Scan(&sum, &count)
		Check(err)
	} else {
		for i := start - 1; i < min(end, len(s.ExtraCols[conditionColIndex].Cells)); i++ {
			conditionExpression := "=" + s.ExtraCols[conditionColIndex].Cells[i].Value + criteria
			conditionVal, err := s.evalLogicalExpression(parseFormula(conditionExpression))
			if err != nil {
				return Token{}, fmt.Errorf("error evaluating %s condition %s: %w", fName, conditionExpression, err)
			}
			log.Printf("Evaluated condition %s: %t", conditionExpression, conditionVal)
			if conditionVal {
				count += 1
				if fName == "COUNTIF" {
					continue
				}

				sumString := s.ExtraCols[sumColIndex].Cells[i].Value
				sumVal, err := strconv.ParseFloat(sumString, 64)
				if err != nil {
					return Token{}, fmt.Errorf("non-numeric value in sum: %s (from %s%d)", sumString, sumColName, i+1)
				}
				sum += sumVal
			}
		}
	}

	if fName == "SUMIF" {
		return fromFloat(sum), nil
	} else if fName == "COUNTIF" {
		return fromFloat(float64(count)), nil
	} else if fName == "AVERAGEIF" {
		if count == 0 {
			return Token{}, errors.New("no rows match condition")
		}
		return fromFloat(sum / float64(count)), nil
	}
	return Token{}, fmt.Errorf("unrecognized function %s", fName)
}

func (s *Sheet) evalFunction(fName string, arguments [][]Token) (Token, error) {
	fName = strings.ToUpper(fName)
	//log.Printf("Evaluating: %s(%+v)", fName, arguments)

	fDefs, isAssociativeFunc := associativeFuncs[fName]
	if isAssociativeFunc {
		return s.evalAssociativeFunc(fDefs, arguments)
	}

	if fName == "IF" {
		return s.evalIf(arguments)
	}

	if fName == "AVERAGE" {
		return s.evalAverage(arguments)
	}

	if fName == "REGEXMATCH" {
		return s.evalRegexMatch(arguments)
	}

	if fName == "SUMIF" || fName == "COUNTIF" || fName == "AVERAGEIF" {
		return s.evalAggIf(fName, arguments)
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
			//log.Printf("Subexpression: %d:%d", i+1, end)
			if err != nil {
				return Token{}, err
			}

			tokens[i] = val
			for j, nt := range tokens[end+1:] {
				tokens[i+j+1] = nt
			}
			tokens = tokens[:len(tokens)+i-end]
			//log.Printf("After evaluating subexpression: %+v", tokens)
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
			//log.Printf("After evaluating function: %+v", tokens)
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

func (s *Sheet) evalTokensToCell(formula string, tokens []Token) (SheetCell, error) {
	token, err := s.evalTokens(tokens)
	if err != nil {
		return SheetCell{Cell{}, formula}, err
	}
	return SheetCell{Cell{token.TValue, token.TValue != ""}, formula}, nil
}

func (s *Sheet) evalFormula(formula string) (SheetCell, error) {
	return s.evalTokensToCell(formula, parseFormula(formula))
}
