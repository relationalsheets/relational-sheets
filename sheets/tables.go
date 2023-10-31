package sheets

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"golang.org/x/exp/maps"
)

type TableNames struct {
	SchemaName string `db:"schemaname"`
	TableName  string `db:"tablename"`
}

type ForeignKey struct {
	sourceTableName string
	targetTableName string
	sourceColNames  []string
	targetColNames  []string
}

type Column struct {
	Name         string
	IsNullable   bool
	DataType     string
	IsPrimaryKey bool
	Index        int
	Cells        []Cell
}

type Table struct {
	TableNames
	HasPrimaryKey bool
	Cols          map[string]Column
	Fkeys         map[int64]ForeignKey
	RowCount      int
	Oid           int64
}

type Cell struct {
	Value   string
	NotNull bool
}

var TableMap = make(map[string]*Table)

func (table Table) FullName() string {
	return fmt.Sprintf("%s.%s", table.SchemaName, table.TableName)
}

func (fkey ForeignKey) ToString() string {
	return fkey.sourceTableName + "." + strings.Join(fkey.sourceColNames, ",") + "->" + fkey.targetTableName + "." + strings.Join(fkey.targetColNames, ",")
}

func (fkey ForeignKey) toJoinClause(tableName string) string {
	pairs := make([]string, len(fkey.sourceColNames))
	for i, sourceCol := range fkey.sourceColNames {
		pairs[i] = fkey.sourceTableName + "." + sourceCol + " = " + fkey.targetTableName + "." + fkey.targetColNames[i]
	}
	return "LEFT JOIN " + tableName + " ON " + strings.Join(pairs, ",")
}

func (sheet Sheet) OrderedCols(tx *sqlx.Tx) [][]Column {
	log.Printf("Prefs: %v", sheet.PrefsMap)
	cols := make([][]Column, len(sheet.TableNames))

	for i, tableName := range sheet.TableNames {
		table := TableMap[tableName]
		table.loadCols(tx)
		cols[i] = make([]Column, 0, len(table.Cols))
		for _, col := range table.Cols {
			if !sheet.PrefsMap[tableName+"."+col.Name].Hide {
				cols[i] = append(cols[i], table.Cols[col.Name])
			}
		}
		sort.SliceStable(cols[i], func(j, k int) bool {
			indexJ := sheet.PrefsMap[table.FullName()+"."+cols[i][j].Name].Index | cols[i][j].Index
			indexK := sheet.PrefsMap[table.FullName()+"."+cols[i][k].Name].Index | cols[i][k].Index
			return indexJ < indexK
		})
		table.loadConstraints(tx)
	}

	return cols
}

func (sheet *Sheet) sortedTablesAndReqCols(tx *sqlx.Tx) ([]string, map[string]map[string]map[string]string, error) {
	outgoingEdges := make(map[string]map[string]bool)
	incomingEdges := make(map[string]map[string]bool)
	// {tableName: {colNameNeeded: {otherTableName: colOnOtherSideOfFkey}}}}
	requiredCols := make(map[string]map[string]map[string]string)

	table := *sheet.Table
	for i, joinOid := range sheet.JoinOids {
		var join ForeignKey
		var joinFound bool
		for _, potentialJoinTableName := range sheet.TableNames[:i+1] {
			potentialJoinTable := TableMap[potentialJoinTableName]
			join, joinFound = potentialJoinTable.Fkeys[joinOid]
			if joinFound {
				break
			}
		}
		if !joinFound {
			panic(fmt.Sprintf("Missing join %d on table %s (have %v)", joinOid, table.FullName(), table.Fkeys))
		}
		log.Printf("Join info: %v", join)
		addToNestedMap(outgoingEdges, join.targetTableName, join.sourceTableName, true)
		addToNestedMap(incomingEdges, join.sourceTableName, join.targetTableName, true)
		for i, colName := range join.targetColNames {
			addToNestedMap2(requiredCols, join.targetTableName, colName, join.sourceTableName, join.sourceColNames[i])
		}
	}

	log.Printf("Nodes: %v", sheet.TableNames)
	log.Printf("Outgoing: %v", outgoingEdges)
	log.Printf("Incoming: %v", incomingEdges)
	tableNames, err := topoSort(sheet.TableNames, outgoingEdges, incomingEdges)
	return tableNames, requiredCols, err
}

func (sheet Sheet) GetCol(name string) Column {
	return sheet.Table.Cols[name]
}

func loadTables() {
	tables := make([]Table, 0)
	err := conn.Select(&tables, `
		SELECT COALESCE(tablename, '') tablename
			, COALESCE(schemaname, '') schemaname
			, oid
		FROM pg_catalog.pg_tables
		LEFT JOIN pg_catalog.pg_class
			ON relname = tablename AND relkind = 'r'
		WHERE schemaname != 'pg_catalog'
			AND schemaname != 'information_schema'
			AND schemaname != 'db_interface'
		ORDER BY schemaname, tablename DESC`)
	Check(err)
	for i, table := range tables {
		//log.Printf("Loading table %s", table.FullName())
		TableMap[table.FullName()] = &tables[i]
	}
	log.Printf("Retrieved %d Tables", len(TableMap))
}

func (table *Table) loadCols(tx *sqlx.Tx) {
	if len(table.Cols) > 0 {
		return
	}
	table.Cols = make(map[string]Column)

	if tx == nil {
		tx = Begin()
		defer Commit(tx)
	}

	cols := make([]Column, 0, 100)
	err := tx.Select(&cols, `
		SELECT column_name "name"
			   , is_nullable = 'YES' isnullable
			   , data_type datatype
			   , ordinal_position "index"
		FROM information_schema.columns
		WHERE table_name = $1
		AND table_schema = $2`,
		table.TableName,
		table.SchemaName)
	Check(err)
	log.Printf("Retrieved %d columns from %s", len(cols), table.FullName())
	for _, col := range cols {
		table.Cols[col.Name] = col
	}
}

func (t *Table) loadConstraints(tx *sqlx.Tx) {
	if len(t.Fkeys) > 0 {
		return
	}
	log.Printf("Loading constraints for table %s (%d)", t.FullName(), t.Oid)
	t.Fkeys = make(map[int64]ForeignKey)

	if tx == nil {
		tx = Begin()
		defer Commit(tx)
	}

	// Query the primary keys, but returns IDs instead of column names
	pKeyAttNums := pq.Int64Array{}
	err := tx.Get(&pKeyAttNums, `
		SELECT conkey
		FROM pg_catalog.pg_constraint
		WHERE conrelid = $1
			AND contype = 'p'`,
		t.Oid)
	if err != nil && err.Error() != "sql: no rows in result set" {
		panic(err)
	}

	// Query the foreign keys, but returns IDs instead of column names
	rawFkeys := make([]struct {
		Oid       int64
		Conrelid  int64
		Confrelid int64
		// PostgreSQL integers are 64-bit, so there is no IntArray type
		// This is why we use int64 for most ints here
		Conkey  pq.Int64Array
		Confkey pq.Int64Array
	}, 0, 20)
	err = tx.Select(&rawFkeys, `
		SELECT oid
		    , conrelid
			, confrelid
			, conkey
			, confkey
		FROM pg_catalog.pg_constraint
		WHERE (conrelid = $1 OR confrelid = $1)
			AND contype = 'f'`,
		t.Oid)
	Check(err)
	log.Printf("Retrieved %d fkeys relating to %s (%d): %+v", len(rawFkeys), t.FullName(), t.Oid, rawFkeys)

	// Get the names for each column referenced in an fkey
	idsToNames := make(map[[2]int64]string)
	relIds := make([]string, 1, 20)
	relIds[0] = strconv.FormatInt(t.Oid, 10)
	for _, rawFkey := range rawFkeys {
		// may add duplicates but that's OK
		relIds = append(
			append(relIds,
				strconv.FormatInt(rawFkey.Conrelid, 10)),
			strconv.FormatInt(rawFkey.Confrelid, 10))
	}
	query := fmt.Sprintf(`
		SELECT attrelid
		    , attnum
		    , attname
		FROM pg_catalog.pg_attribute
		WHERE attrelid IN (%s)`,
		strings.Join(relIds, ","))
	log.Println("Executing: " + query)
	rows, err := tx.Query(query)
	Check(err)
	for rows.Next() {
		var relId, attNum int64
		var colName string
		err := rows.Scan(&relId, &attNum, &colName)
		Check(err)
		idsToNames[[2]int64{relId, attNum}] = colName
	}

	// Flag the primary keys in t.Cols
	t.loadCols(tx)
	if len(pKeyAttNums) > 0 {
		for _, attNum := range pKeyAttNums {
			colName := idsToNames[[2]int64{t.Oid, attNum}]
			col := t.Cols[colName]
			col.IsPrimaryKey = true
			t.Cols[colName] = col
		}
		t.HasPrimaryKey = true
		log.Printf("Retrieved primary key for %s", t.FullName())
	} else {
		log.Printf("No primary key for %s", t.FullName())
	}

	// Populate t.Fkeys
	for _, rawFkey := range rawFkeys {
		fkey := ForeignKey{}
		if rawFkey.Conrelid == t.Oid {
			fkey.sourceTableName = t.FullName()
			for _, t2 := range TableMap {
				if t2.Oid == rawFkey.Confrelid {
					fkey.targetTableName = t2.FullName()
				}
			}
			for _, attnum := range rawFkey.Conkey {
				fkey.sourceColNames = append(fkey.sourceColNames, idsToNames[[2]int64{rawFkey.Conrelid, attnum}])
			}
			for _, attnum := range rawFkey.Confkey {
				fkey.targetColNames = append(fkey.targetColNames, idsToNames[[2]int64{rawFkey.Confrelid, attnum}])
			}
		} else {
			if rawFkey.Confrelid != t.Oid {
				panic("SQL WHERE clause violated -- unexpected table oid")
			}

			fkey.targetTableName = t.FullName()
			for _, t2 := range TableMap {
				if t2.Oid == rawFkey.Conrelid {
					fkey.sourceTableName = t2.FullName()
				}
			}
			for _, attnum := range rawFkey.Conkey {
				fkey.sourceColNames = append(fkey.sourceColNames, idsToNames[[2]int64{rawFkey.Conrelid, attnum}])
			}
			for _, attnum := range rawFkey.Confkey {
				fkey.targetColNames = append(fkey.targetColNames, idsToNames[[2]int64{rawFkey.Confrelid, attnum}])
			}
		}

		if fkey.sourceTableName == "" || fkey.targetTableName == "" {
			panic(fmt.Sprintf("Unexpected table oid in %v", rawFkey))
		}

		t.Fkeys[rawFkey.Oid] = fkey
	}
}

func (sheet *Sheet) loadJoins() {
	tx := Begin()
	defer Commit(tx)

	table := sheet.Table
	sheet.TableNames = make([]string, 1+len(sheet.JoinOids))
	sheet.TableNames[0] = table.FullName()
	table.loadConstraints(tx)
	for i, joinOid := range sheet.JoinOids {
		join, ok := table.Fkeys[joinOid]
		if !ok {
			panic(fmt.Sprintf("Unable to load join %d on %s (have %v)", joinOid, table.FullName(), table.Fkeys))
		}
		if join.sourceTableName == table.FullName() {
			table = TableMap[join.targetTableName]
		} else {
			table = TableMap[join.sourceTableName]
		}
		sheet.TableNames[i+1] = table.FullName()
		table.loadConstraints(tx)
	}
}

func (sheet *Sheet) LoadRows(limit int, offset int) [][][]Cell {
	cols := sheet.OrderedCols(nil)
	cells := make([][][]Cell, len(sheet.TableNames))
	// TODO: Check if table.TableName and column names are valid somewhere
	casts := []string{}
	orderClauses := []string{}
	for i, tableName := range sheet.TableNames {
		table := TableMap[tableName]
		cells[i] = make([][]Cell, len(cols[i]))
		for j, col := range cols[i] {
			cells[i][j] = make([]Cell, 0, limit)
			name := table.FullName() + "." + col.Name
			cast := fmt.Sprintf("%s::text, %s IS NOT NULL", name, name)
			casts = append(casts, cast)

			pref := sheet.PrefsMap[tableName+"."+col.Name]
			if pref.SortOn {
				orderDirection := "DESC"
				if pref.Ascending {
					orderDirection = "ASC"
				}
				colOrder := tableName + "." + col.Name + " " + orderDirection
				orderClauses = append(orderClauses, colOrder)
			}
		}
	}

	fromClause := "FROM " + sheet.TableNames[0]
	for i, joinOid := range sheet.JoinOids {
		tableName := sheet.TableNames[i+1]
		fkey := TableMap[tableName].Fkeys[joinOid]
		fromClause += " " + fkey.toJoinClause(tableName)
	}
	query := "SELECT " + strings.Join(casts, ", ") + " " + fromClause
	if len(orderClauses) > 0 {
		query += " ORDER BY " + strings.Join(orderClauses, ", ")
	}
	query += " LIMIT $1 OFFSET $2"
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	Check(err)

	sheet.RowCount = 0
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		Check(err)
		index := 0
		for i, _ := range sheet.TableNames {
			for j, _ := range cols[i] {
				val := ""
				isNotNull := scanResult[2*index+1].(bool)
				if isNotNull {
					val = scanResult[2*index].(string)
				}
				cells[i][j] = append(cells[i][j], Cell{val, isNotNull})
				index++
			}
		}
		sheet.RowCount++
	}
	log.Printf("Retrieved %d rows from %s", sheet.RowCount, sheet.Table.FullName())
	Check(rows.Close())
	return cells
}

func (sheet *Sheet) InsertRow(tx *sqlx.Tx, tableName string, values map[string]string, returning []string) ([]interface{}, error) {
	nonEmptyValues := prepareValues(values, false)
	if len(nonEmptyValues) == 0 {
		return nil, errors.New("All fields are empty")
	}

	keys := maps.Keys(nonEmptyValues)
	valueLabels := make([]string, len(nonEmptyValues))
	for i, key := range keys {
		valueLabels[i] = ":" + key
	}

	returningCasts := make([]string, len(returning))
	for i, colName := range returning {
		returningCasts[i] = fmt.Sprintf("CAST(%s AS TEXT)", colName)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(keys, ", "),
		strings.Join(valueLabels, ", "))
	if len(returningCasts) > 0 {
		query += " RETURNING " + strings.Join(returningCasts, ", ")
	}
	log.Println("Executing:", query)
	log.Println("Values:", nonEmptyValues)
	rows, err := tx.NamedQuery(query, nonEmptyValues)
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if len(returningCasts) > 0 {
		if !rows.Next() {
			panic("No ID returned by insert")
		}
		result, err = rows.SliceScan()
	}
	Check(rows.Close())
	return result, err
}

func addToNestedMap[V any](m map[string]map[string]V, k1, k2 string, v V) {
	_, ok := m[k1]
	if !ok {
		m[k1] = make(map[string]V)
	}
	m[k1][k2] = v
}

func addToNestedMap2[V any](m map[string]map[string]map[string]V, k1, k2, k3 string, v V) {
	_, ok := m[k1]
	if !ok {
		m[k1] = make(map[string]map[string]V)
	}
	addToNestedMap(m[k1], k2, k3, v)
}

func topoSort(nodes []string, outgoingEdges, incomingEdges map[string]map[string]bool) ([]string, error) {
	// Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
	s := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if len(incomingEdges[n]) == 0 {
			s = append(s, n)
		}
	}

	l := make([]string, 0, len(nodes))
	for len(s) > 0 {
		n := s[0]
		s = s[1:]
		l = append(l, n)
		for m, _ := range outgoingEdges[n] {
			delete(outgoingEdges[n], m)
			delete(incomingEdges[m], n)
			if len(incomingEdges[m]) == 0 {
				s = append(s, m)
			}
		}
	}

	for _, outgoing := range outgoingEdges {
		if len(outgoing) > 0 {
			return l, fmt.Errorf("cycle detected: %v", outgoingEdges)
		}
	}

	return l, nil
}

func isEmpty[K comparable](m map[K]string) bool {
	for _, value := range m {
		if value != "" {
			return false
		}
	}
	return true
}

func prepareValues(values map[string]string, allowEmpty bool) map[string]interface{} {
	nonEmptyValues := make(map[string]interface{})
	for key, value := range values {
		if value != "" {
			nonEmptyValues[key] = value
		} else if allowEmpty {
			nonEmptyValues[key] = nil
		}
	}
	return nonEmptyValues
}

func (sheet *Sheet) InsertMultipleRows(values map[string]map[string]string, referencedValues map[string]map[string]string) error {
	tx := Begin()
	defer Commit(tx)

	tableNames, requiredCols, err := sheet.sortedTablesAndReqCols(tx)
	log.Printf("Sorted tables: %v", tableNames)
	log.Printf("Required cols: %v", requiredCols)
	if err != nil {
		return err
	}

	for _, tableName := range tableNames {
		tableValues := values[tableName]
		if isEmpty(tableValues) {
			continue
		}

		log.Printf("requiredCols: %v", requiredCols)
		for otherTableName, otherColMapping := range requiredCols {
			for otherColName, tableMapping := range otherColMapping {
				colName, ok := tableMapping[tableName]
				referencedValue, ok2 := referencedValues[otherTableName][otherColName]
				if ok && ok2 {
					tableValues[colName] = referencedValue
				}
			}
		}

		tableRequiredCols := maps.Keys(requiredCols[tableName])
		row, err := sheet.InsertRow(tx, tableName, tableValues, tableRequiredCols)
		for i, colName := range tableRequiredCols {
			log.Printf("Setting %s.%s to %s", tableName, colName, row[i].(string))
			addToNestedMap(referencedValues, tableName, colName, row[i].(string))
		}
		Check(err)
	}
	return nil
}

func (table *Table) updateRow(values map[string]string, primaryKeys map[string]string) error {
	if len(primaryKeys) == 0 {
		return errors.New("Cannot update table without primary key: " + table.FullName())
	}

	assignments := make([]string, len(values))
	for i, key := range maps.Keys(values) {
		assignments[i] = key + " = :" + key
	}
	whereClauses := make([]string, len(primaryKeys))
	for i, key := range maps.Keys(primaryKeys) {
		whereClauses[i] = key + " = :" + key
		values[key] = primaryKeys[key]
	}
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table.FullName(),
		strings.Join(assignments, ", "),
		strings.Join(whereClauses, " AND "))
	prepared := prepareValues(values, true)
	log.Println("Executing:", query)
	log.Println("Values:", prepared)
	_, err := conn.NamedExec(query, prepared)
	return err
}

func (sheet *Sheet) UpdateRows(values map[string]map[string]string, primaryKeys map[string]map[string]string) error {
	log.Printf("UpdateRows(%v, %v)", values, primaryKeys)
	for tableName, tableValues := range values {
		if isEmpty(tableValues) {
			continue
		}
		table := TableMap[tableName]
		err := table.updateRow(tableValues, primaryKeys[tableName])
		if err != nil {
			return err
		}
	}
	return nil
}
