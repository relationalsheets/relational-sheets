package sheets

import (
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"log"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"
)

type TableNames struct {
	SchemaName string `db:"schemaname"`
	TableName  string `db:"tablename"`
}

type ForeignKey struct {
	isFrom         bool
	otherTableName string
	sourceColNames []string
	targetColNames []string
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
	if fkey.isFrom {
		return strings.Join(fkey.sourceColNames, ",") + "->" + fkey.otherTableName + "." + strings.Join(fkey.targetColNames, ",")
	} else {
		return fkey.otherTableName + "." + strings.Join(fkey.sourceColNames, ",") + "->" + strings.Join(fkey.targetColNames, ",")
	}
}

func (fkey ForeignKey) toJoinClause(tableName string) string {
	pairs := make([]string, len(fkey.sourceColNames))
	for i, sourceCol := range fkey.sourceColNames {
		if fkey.isFrom {
			pairs[i] = fkey.otherTableName + "." + sourceCol + " = " + tableName + "." + fkey.targetColNames[i]
		} else {
			pairs[i] = tableName + "." + sourceCol + " = " + fkey.otherTableName + "." + fkey.targetColNames[i]
		}
	}
	return "JOIN " + tableName + " ON " + strings.Join(pairs, ",")
}

func (sheet Sheet) OrderedTablesAndCols(tx *sqlx.Tx) ([]string, [][]Column) {
	tableNames := make([]string, 1+len(sheet.JoinOids))
	cols := make([][]Column, len(tableNames))
	table := sheet.Table
	for i := 0; i <= len(sheet.JoinOids); i++ {
		tableNames[i] = table.FullName()
		table.loadCols(tx)
		cols[i] = make([]Column, 0, len(table.Cols))
		for _, col := range table.Cols {
			if !sheet.prefsMap[col.Name].Hide {
				cols[i] = append(cols[i], table.Cols[col.Name])
			}
		}
		table.loadConstraints(tx)
		if i < len(sheet.JoinOids) {
			joinOid := sheet.JoinOids[i]
			join, ok := table.Fkeys[joinOid]
			if !ok {
				panic(fmt.Sprintf("Missing join %d on table %s (have %v)", joinOid, table.FullName(), table.Fkeys))
			}
			table = TableMap[join.otherTableName]
		}
		if len(sheet.prefsMap) > 0 {
			sort.SliceStable(cols[i], func(j, k int) bool {
				indexJ := sheet.prefsMap[table.FullName()+"."+cols[i][j].Name].Index | cols[i][j].Index
				indexK := sheet.prefsMap[table.FullName()+"."+cols[i][k].Name].Index | cols[i][k].Index
				return indexJ < indexK
			})
		}
	}
	return tableNames, cols
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

	// Populate t.FkeysFrom and t.FkeysTo
	for _, rawFkey := range rawFkeys {
		fkey := ForeignKey{}
		var otherTableOid int64
		if rawFkey.Conrelid == t.Oid {
			fkey.isFrom = true
			otherTableOid = rawFkey.Confrelid
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

			otherTableOid = rawFkey.Conrelid
			for _, attnum := range rawFkey.Conkey {
				fkey.targetColNames = append(fkey.targetColNames, idsToNames[[2]int64{rawFkey.Conrelid, attnum}])
			}
			for _, attnum := range rawFkey.Confkey {
				fkey.sourceColNames = append(fkey.sourceColNames, idsToNames[[2]int64{rawFkey.Confrelid, attnum}])
			}
		}

		for _, t2 := range TableMap {
			if t2.Oid == otherTableOid {
				fkey.otherTableName = t2.FullName()
			}
		}
		if fkey.otherTableName == "" {
			panic(fmt.Sprintf("Unexpected table oid %d", otherTableOid))
		}

		t.Fkeys[rawFkey.Oid] = fkey
	}
}

func (sheet *Sheet) LoadRows(limit int, offset int) [][][]Cell {
	tableNames, cols := sheet.OrderedTablesAndCols(nil)
	cells := make([][][]Cell, len(tableNames))
	// TODO: Check if table.TableName and column names are valid somewhere
	casts := make([]string, 0)
	for i, tableName := range tableNames {
		table := TableMap[tableName]
		cells[i] = make([][]Cell, len(cols[i]))
		for j, col := range cols[i] {
			cells[i][j] = make([]Cell, limit)
			name := table.FullName() + "." + col.Name
			cast := fmt.Sprintf("%s::text, %s IS NOT NULL", name, name)
			casts = append(casts, cast)
		}
	}

	fromClause := "FROM " + tableNames[0]
	for i, joinOid := range sheet.JoinOids {
		tableName := tableNames[i+1]
		fkey := TableMap[tableName].Fkeys[joinOid]
		fromClause += " " + fkey.toJoinClause(tableName)
	}
	query := fmt.Sprintf(
		"SELECT %s %s LIMIT $1 OFFSET $2",
		strings.Join(casts, ", "),
		fromClause)
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	Check(err)

	sheet.RowCount = 0
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		log.Printf("row: %v", scanResult)
		Check(err)
		index := 0
		for i, _ := range tableNames {
			for j, _ := range cols[i] {
				val := ""
				isNotNull := scanResult[2*index+1].(bool)
				if isNotNull {
					val = scanResult[2*index].(string)
				}
				cells[i][j][sheet.RowCount] = Cell{val, isNotNull}
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

	valueLabels := make([]string, len(nonEmptyValues))
	for i, key := range maps.Keys(nonEmptyValues) {
		valueLabels[i] = ":" + key
	}

	returningCasts := make([]string, len(returning))
	for i, colName := range returning {
		returningCasts[i] = fmt.Sprintf("CAST(%s AS TEXT)", colName)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(maps.Keys(nonEmptyValues), ", "),
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

func topoSort(nodes map[string]bool, outgoingEdges, incomingEdges map[string]map[string]bool) ([]string, error) {
	// Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
	s := make([]string, 0, len(nodes))
	for n, _ := range nodes {
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

func (sheet *Sheet) sortedTablesAndReqCols() ([]string, map[string]map[string]map[string]string, error) {
	nodes := make(map[string]bool)
	outgoingEdges := make(map[string]map[string]bool)
	incomingEdges := make(map[string]map[string]bool)
	// {tableName: {colName: {otherTableName: colName}}}}
	requiredCols := make(map[string]map[string]map[string]string)
	table := sheet.Table
	nodes[table.FullName()] = true
	log.Printf("Join oids: %v", sheet.JoinOids)
	for _, joinOid := range sheet.JoinOids {
		join := table.Fkeys[joinOid]
		log.Printf("Join info: %v", join)
		if join.isFrom {
			addToNestedMap(outgoingEdges, join.otherTableName, table.FullName(), true)
			addToNestedMap(incomingEdges, table.FullName(), join.otherTableName, true)
			for i, colName := range join.targetColNames {
				addToNestedMap2(requiredCols, join.otherTableName, colName, table.FullName(), join.sourceColNames[i])
			}
		} else {
			addToNestedMap(outgoingEdges, table.FullName(), join.otherTableName, true)
			addToNestedMap(incomingEdges, join.otherTableName, table.FullName(), true)
			for i, colName := range join.sourceColNames {
				addToNestedMap2(requiredCols, table.FullName(), colName, join.otherTableName, join.targetColNames[i])
			}
		}
		table = TableMap[join.otherTableName]
		nodes[join.otherTableName] = true
	}

	log.Printf("Nodes: %v", nodes)
	log.Printf("Outgoing: %v", outgoingEdges)
	log.Printf("Incoming: %v", incomingEdges)
	tableNames, err := topoSort(nodes, outgoingEdges, incomingEdges)
	return tableNames, requiredCols, err
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

func (sheet *Sheet) InsertMultipleRows(values map[string]map[string]string) error {
	tableNames, requiredCols, err := sheet.sortedTablesAndReqCols()
	log.Printf("Sorted tables: %v", tableNames)
	log.Printf("Required cols: %v", requiredCols)
	if err != nil {
		return err
	}

	tx := Begin()
	defer Commit(tx)
	for _, tableName := range tableNames {
		// Note that prior insertions may have made values[tableName] non-empty, e.g. for join
		// tables for many-to-many relationships.
		if isEmpty(values[tableName]) {
			continue
		}

		tableRequiredCols := maps.Keys(requiredCols[tableName])
		row, err := sheet.InsertRow(tx, tableName, values[tableName], tableRequiredCols)
		if err != nil {
			Check(tx.Rollback())
			return err
		}
		for i, colName := range tableRequiredCols {
			for otherTableName, colToSet := range requiredCols[tableName][colName] {
				addToNestedMap(values, otherTableName, colToSet, row[i].(string))
			}
		}
	}
	return nil
}

func (table *Table) updateRow(values map[string]string) error {
	table.loadConstraints(nil)
	if !table.HasPrimaryKey {
		return errors.New("Cannot update table without primary key: " + table.FullName())
	}

	assignments := make([]string, len(values))
	for i, key := range maps.Keys(values) {
		assignments[i] = key + " = :" + key
	}
	query := fmt.Sprintf(
		"UPDATE %s SET %s",
		table.FullName(),
		strings.Join(assignments, ", "))
	prepared := prepareValues(values, true)
	log.Println("Executing:", query)
	log.Println("Values:", prepared)
	_, err := conn.NamedExec(query, prepared)
	return err
}

func (sheet *Sheet) UpdateRows(values map[string]map[string]string) error {
	for tableName, tableValues := range values {
		if isEmpty(tableValues) {
			continue
		}
		table := TableMap[tableName]
		err := table.updateRow(tableValues)
		if err != nil {
			return err
		}
	}
	return nil
}
