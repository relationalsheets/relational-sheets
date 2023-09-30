package sheets

import (
	"errors"
	"fmt"
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
	FkeysFrom     map[int]ForeignKey
	FkeysTo       map[int]ForeignKey
	RowCount      int
	Oid           int64
}

type Cell struct {
	Value   string
	NotNull bool
}

var Tables = make([]Table, 0, 20)
var tableMap = make(map[string]Table)

func (table Table) FullName() string {
	return fmt.Sprintf("%s.%s", table.SchemaName, table.TableName)
}

func (fkey ForeignKey) ToString(isSource bool) string {
	if isSource {
		return strings.Join(fkey.sourceColNames, ",") + "->" + fkey.otherTableName + "." + strings.Join(fkey.targetColNames, ",")
	} else {
		return fkey.otherTableName + "." + strings.Join(fkey.sourceColNames, ",") + "->" + strings.Join(fkey.targetColNames, ",")
	}
}

func (sheet Sheet) OrderedColNames() []string {
	colNames := make([]string, 0, len(sheet.Table.Cols))
	indices := make([]int, 0, len(sheet.Table.Cols))
	for _, col := range sheet.Table.Cols {
		if !sheet.prefsMap[col.Name].Hide {
			colNames = append(colNames, col.Name)
			indices = append(indices, col.Index)
		}
	}
	sort.SliceStable(colNames, func(i, j int) bool {
		indexI := sheet.prefsMap[colNames[i]].Index | indices[i]
		indexJ := sheet.prefsMap[colNames[j]].Index | indices[j]
		return indexI < indexJ
	})
	return colNames
}

func (sheet Sheet) OrderedCols() []Column {
	colNames := sheet.OrderedColNames()
	cols := make([]Column, len(colNames))
	for i, name := range colNames {
		cols[i] = sheet.Table.Cols[name]
	}
	return cols
}

func (sheet Sheet) GetCol(name string) Column {
	return sheet.Table.Cols[name]
}

func LoadTables() {
	err := conn.Select(&Tables, `
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
	log.Printf("Retrieved %d Tables", len(Tables))
	for _, table := range Tables {
		tableMap[table.FullName()] = table
	}
}

func (table *Table) loadCols() {
	cols := make([]Column, 0, 100)
	err := conn.Select(&cols, `
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
	table.Cols = make(map[string]Column)
	for _, col := range cols {
		table.Cols[col.Name] = col
	}
}

func (t *Table) loadConstraints() {
	t.FkeysFrom = make(map[int]ForeignKey)
	t.FkeysTo = make(map[int]ForeignKey)
	tablesByOid := make(map[int64]Table)
	for _, t := range Tables {
		tablesByOid[t.Oid] = t
	}

	// Query the primary keys, but returns IDs instead of column names
	pKeyAttNums := []int64{}
	err := conn.Get(&pKeyAttNums, `
		SELECT conkey
		FROM pg_catalog.pg_constraint
		WHERE conrelid = $1
			AND contype = 'p'`,
		t.Oid)

	// Query the foreign keys, but returns IDs instead of column names
	rawFkeys := make([]struct {
		Oid       int
		Conrelid  int64
		Confrelid int64
		// PostgreSQL integers are 64-bit, so there is no IntArray type
		// This is why we use int64 for most ints here
		Conkey  pq.Int64Array
		Confkey pq.Int64Array
	}, 0, 20)
	err = conn.Select(&rawFkeys, `
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
	rows, err := conn.Query(query)
	Check(err)
	for rows.Next() {
		var relId, attNum int64
		var colName string
		err := rows.Scan(&relId, &attNum, &colName)
		Check(err)
		idsToNames[[2]int64{relId, attNum}] = colName
	}

	// Flag the primary keys in t.Cols
	if len(pKeyAttNums) > 0 {
		for _, attNum := range pKeyAttNums {
			colName := idsToNames[[2]int64{t.Oid, attNum}]
			col := t.Cols[colName]
			col.IsPrimaryKey = true
			t.Cols[colName] = col
		}
		log.Printf("Retrieved primary key for %s", t.FullName())
	} else {
		log.Printf("No primary key for %s", t.FullName())
	}

	// Populate t.FkeysFrom and t.FkeysTo
	for _, rawFkey := range rawFkeys {
		var sourceColNames, targetColNames []string
		var otherTableOid int64
		if rawFkey.Conrelid == t.Oid {
			otherTableOid = rawFkey.Confrelid
			for _, attnum := range rawFkey.Conkey {
				sourceColNames = append(sourceColNames, idsToNames[[2]int64{rawFkey.Conrelid, attnum}])
			}
			for _, attnum := range rawFkey.Confkey {
				targetColNames = append(sourceColNames, idsToNames[[2]int64{rawFkey.Confrelid, attnum}])
			}
		} else {
			if rawFkey.Confrelid != t.Oid {
				panic("SQL WHERE clause violated -- unexpected table oid")
			}

			otherTableOid = rawFkey.Conrelid
			for _, attnum := range rawFkey.Conkey {
				targetColNames = append(targetColNames, idsToNames[[2]int64{rawFkey.Conrelid, attnum}])
			}
			for _, attnum := range rawFkey.Confkey {
				sourceColNames = append(sourceColNames, idsToNames[[2]int64{rawFkey.Confrelid, attnum}])
			}
		}

		otherTableName := ""
		for _, t2 := range Tables {
			if t2.Oid == otherTableOid {
				otherTableName = t2.FullName()
			}
		}
		if otherTableName == "" {
			panic(fmt.Sprintf("Unexpected table oid %d", otherTableOid))
		}

		fkey := ForeignKey{
			otherTableName,
			sourceColNames,
			targetColNames,
		}
		if rawFkey.Conrelid == t.Oid {
			t.FkeysFrom[rawFkey.Oid] = fkey
		} else {
			t.FkeysTo[rawFkey.Oid] = fkey
		}
	}
}

func (sheet *Sheet) LoadCells(limit int, offset int) {
	colNames := sheet.OrderedColNames()
	// TODO: Check if table.TableName and column names are valid somewhere
	casts := make([]string, 0, len(colNames))
	for _, name := range colNames {
		col := sheet.Table.Cols[name]
		col.Cells = make([]Cell, limit)
		sheet.Table.Cols[name] = col

		cast := fmt.Sprintf("\"%s\"::text, \"%s\" IS NOT NULL", name, name)
		casts = append(casts, cast)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s LIMIT $1 OFFSET $2",
		strings.Join(casts, ", "),
		sheet.Table.FullName())
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	Check(err)

	sheet.Table.RowCount = 0
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		Check(err)
		for i := 0; i < len(casts); i++ {
			val, _ := scanResult[2*i].(string)
			sheet.Table.Cols[colNames[i]].Cells[sheet.Table.RowCount] = Cell{
				val, scanResult[2*i+1].(bool),
			}
		}
		sheet.Table.RowCount++
	}
	log.Printf("Retrieved %d rows from %s", sheet.Table.RowCount, sheet.Table.FullName())
	Check(rows.Close())
}

func (sheet *Sheet) InsertRow(values map[string]string) error {
	colNames := sheet.OrderedColNames()
	valueLabels := make([]string, 0, len(colNames))
	nonEmptyValues := make(map[string]interface{})
	for _, name := range colNames {
		if values[name] != "" {
			nonEmptyValues[name] = values[name]
			valueLabels = append(valueLabels, ":"+name)
		}
	}
	if len(nonEmptyValues) == 0 {
		return errors.New("All fields are empty")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		sheet.Table.FullName(),
		strings.Join(maps.Keys(nonEmptyValues), ", "),
		strings.Join(valueLabels, ", "))
	log.Println("Executing:", query)
	log.Println("Values:", nonEmptyValues)
	_, err := conn.NamedExec(query, nonEmptyValues)
	return err
}
