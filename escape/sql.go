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
package escape

import (
	"acb/db-interface/fkeys"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/exp/maps"
)

type SafeSQL struct {
	raw string
}

var operators = []string{
	"=", "<", "<=", ">", ">=", "LIKE",
}
var sqlTypes = []string{
	"text", "numeric",
}

func isValidForLabel(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || r == '_') {
			return false
		}
	}
	return true
}

func escapeIdentifier(identifier string) (string, error) {
	parts := strings.Split(identifier, ".")
	escaped := []string{}
	for _, part := range parts {
		if strings.Contains(part, "\"") {
			return "", fmt.Errorf("Illegal identifier: %s", identifier)
		}
		escaped = append(escaped, fmt.Sprintf("\"%s\"", part))
	}
	return strings.Join(escaped, "."), nil
}

func EscapeIdentifier(identifier string) (SafeSQL, error) {
	safe, err := escapeIdentifier(identifier)
	return SafeSQL{safe}, err
}

func isConstant(raw string) bool {
	_, err := strconv.Atoi(raw)
	if err == nil {
		return true
	}
	unSingleQuoted := strings.Trim(raw, "'")
	if raw == "'"+unSingleQuoted+"'" && !strings.Contains(unSingleQuoted, "'") {
		return true
	}
	unParenthesized := strings.TrimLeft(strings.TrimRight(raw, ")"), "(")
	if raw == "("+unParenthesized+")" {
		for _, item := range strings.Split(unParenthesized, ",") {
			if !isConstant(item) {
				return false
			}
		}
		return true
	}
	return false
}

func escapeIdentifierOrConstant(raw string) (string, error) {
	if isConstant(raw) {
		return raw, nil
	}
	return escapeIdentifier(raw)
}

func MakeCast(identifier, castType, alias string) (SafeSQL, error) {
	safe, err := escapeIdentifier(identifier)
	if err != nil {
		return SafeSQL{}, err
	}
	if castType != "" {
		if !slices.Contains(sqlTypes, castType) {
			return SafeSQL{}, fmt.Errorf("Unsupported SQL type: %s", castType)
		}
		safe = fmt.Sprintf("CAST(%s AS %s)", safe, castType)
	}
	if alias != "" {
		alias, err = escapeIdentifier(alias)
		if err != nil {
			return SafeSQL{}, err
		}
		safe = fmt.Sprintf("%s AS %s", safe, alias)
	}
	return SafeSQL{safe}, nil
}

func MakeNotNull(identifer string) (SafeSQL, error) {
	safe, err := escapeIdentifier(identifer)
	if err != nil {
		return SafeSQL{}, err
	}
	return SafeSQL{fmt.Sprintf("%s IS NOT NULL", safe)}, nil
}

func MakeClause(lhs, operator, rhs string) (SafeSQL, error) {
	lhsSafe, err := escapeIdentifierOrConstant(lhs)
	if err != nil {
		return SafeSQL{}, fmt.Errorf("Illegal left-hand side of clause: %s (%w)", lhs, err)
	}
	rhsSafe, err := escapeIdentifierOrConstant(rhs)
	if err != nil {
		return SafeSQL{}, fmt.Errorf("Illegal right-hand side of clause: %s (%w)", rhs, err)
	}
	if !slices.Contains(operators, operator) {
		return SafeSQL{}, fmt.Errorf("Illegal operator: %s", operator)
	}
	return SafeSQL{fmt.Sprintf("%s %s %s", lhsSafe, operator, rhsSafe)}, nil
}

func MakeFilterClause(lhs, filter string) (SafeSQL, error) {
	for _, operator := range operators {
		suffix, found := strings.CutPrefix(filter, operator)
		if found {
			return MakeClause(lhs, operator, strings.TrimLeft(suffix, " "))
		}
	}
	return SafeSQL{}, fmt.Errorf("No supported filter operator in: %s", filter)
}

func MakeOrderExpr(identifier string, ascending bool) (SafeSQL, error) {
	safe, err := escapeIdentifier(identifier)
	if err != nil {
		return SafeSQL{}, err
	}
	orderDirection := " DESC"
	if ascending {
		orderDirection = " ASC"
	}
	return SafeSQL{safe + orderDirection}, nil
}

func toJoinClause(fkey fkeys.ForeignKey, tableName string) (string, error) {
	tableIdent, err := escapeIdentifier(tableName)
	if err != nil {
		return "", err
	}
	pairs := make([]string, len(fkey.SourceColNames))
	for i, sourceCol := range fkey.SourceColNames {
		clause, err := MakeClause(fkey.SourceTableName+"."+sourceCol, "=", fkey.TargetTableName+"."+fkey.TargetColNames[i])
		if err != nil {
			return "", fmt.Errorf("Illegal clause in JOIN: %w", err)
		}
		pairs[i] = clause.raw
	}
	return "LEFT JOIN " + tableIdent + " ON " + strings.Join(pairs, ", "), nil
}

func joinSafeSQL(parts []SafeSQL, sep string) string {
	unwrapped := make([]string, len(parts))
	for i, part := range parts {
		unwrapped[i] = part.raw
	}
	return strings.Join(unwrapped, ", ")
}

func MakeSelectStmt(tableNames []string, joins []fkeys.ForeignKey, columns, filterClauses, orderClauses []SafeSQL, limit bool) (string, error) {
	fromClause := " FROM " + tableNames[0]
	for i, fkey := range joins {
		tableName := tableNames[i+1]
		joinClause, err := toJoinClause(fkey, tableName)
		if err != nil {
			return "", err
		}
		fromClause += " " + joinClause
	}
	query := "SELECT " + joinSafeSQL(columns, ", ") + fromClause
	if len(filterClauses) > 0 {
		query += " WHERE " + joinSafeSQL(filterClauses, " AND ")
	}
	if len(orderClauses) > 0 {
		query += " ORDER BY " + joinSafeSQL(orderClauses, ", ")
	}
	if limit {
		query += " LIMIT $1 OFFSET $2"
	}
	log.Printf("Query: %s", query)
	return query, nil
}

func MakeInsertStmt(tableName string, values map[string]interface{}, returning []string) (string, error) {
	identifier, err := escapeIdentifier(tableName)
	if err != nil {
		return "", err
	}

	if len(values) == 0 {
		return "", fmt.Errorf("All fields are empty")
	}

	colIdentifiers := make([]string, len(values))
	valueLabels := make([]string, len(values))
	for i, key := range maps.Keys(values) {
		if !isValidForLabel(key) {
			return "", fmt.Errorf("Invalid column name: %s", key)
		}
		colIdentifiers[i] = key
		valueLabels[i] = ":" + key
	}

	returningCasts := make([]string, len(returning))
	for i, colName := range returning {
		cast, err := MakeCast(colName, "text", "")
		if err != nil {
			return "", err
		}
		returningCasts[i] = cast.raw
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		identifier,
		strings.Join(colIdentifiers, ", "),
		strings.Join(valueLabels, ", "))
	if len(returningCasts) > 0 {
		query += " RETURNING " + strings.Join(returningCasts, ", ")
	}
	log.Printf("Executing: %s", query)
	return query, nil
}

func MakeUpdateStmt(tableName string, values, primaryKeys map[string]string) (string, error) {
	identifier, err := escapeIdentifier(tableName)
	if err != nil {
		return "", err
	}

	assignments := make([]string, len(values))
	for i, key := range maps.Keys(values) {
		if !isValidForLabel(key) {
			return "", fmt.Errorf("Invalid column name: %s", key)
		}
		assignments[i] = fmt.Sprintf("%s = :%s", key, key)
	}

	whereClauses := make([]string, len(primaryKeys))
	for i, key := range maps.Keys(primaryKeys) {
		colIdentifier, err := escapeIdentifier(key)
		if err != nil {
			return "", err
		}
		whereClauses[i] = fmt.Sprintf("%s = :%s", colIdentifier, key)
		values[key] = primaryKeys[key]
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		identifier,
		strings.Join(assignments, ", "),
		strings.Join(whereClauses, " AND "))
	log.Println("Executing:", query)
	return query, nil
}
