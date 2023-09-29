package sheets

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

type TableNames struct {
	SchemaName string `db:"schemaname"`
	TableName  string `db:"tablename"`
}

type Constraint struct {
	Name           string
	ConstraintType string
	Def            string
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
	Constraints   []Constraint
	RowCount      int
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

func (sheet Sheet) OrderedColNames() []string {
	colNames := make([]string, 0, len(sheet.table.Cols))
	indices := make([]int, 0, len(sheet.table.Cols))
	for _, col := range sheet.table.Cols {
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
		cols[i] = sheet.table.Cols[name]
	}
	return cols
}

func (sheet Sheet) GetCol(name string) Column {
	return sheet.table.Cols[name]
}

func LoadTables() {
	err := conn.Select(&Tables, `
		SELECT COALESCE(tablename, '') tablename
			, COALESCE(schemaname, '') schemaname
		FROM pg_catalog.pg_tables
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

func SetCols(table *Table) {
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

func SetConstraints(table *Table) {
	table.Constraints = make([]Constraint, 0, 10)
	err := conn.Select(&table.Constraints, `
		SELECT x.conname "name"
			, x.contype constrainttype
			, pg_get_constraintdef(x.oid) def
		FROM pg_catalog.pg_constraint x
		INNER JOIN pg_catalog.pg_class pk ON x.conrelid!=0 AND x.conrelid=pk.oid
		INNER JOIN pg_catalog.pg_class fk ON x.confrelid!=0 AND x.confrelid=fk.oid
		WHERE pk.relname = $1
			AND pk.relnamespace = $2::regnamespace`,
		table.TableName,
		table.SchemaName)
	Check(err)
	log.Printf("Retrieved %d constraints from %s", len(table.Constraints), table.FullName())

	for _, constraint := range table.Constraints {
		if constraint.ConstraintType == "p" {
			columnPart := strings.Replace(constraint.Def, "PRIMARY KEY", "", 1)
			primaryKeyColNames := strings.Split(strings.Trim(columnPart, "() "), ", ")
			for key, col := range table.Cols {
				for _, name := range primaryKeyColNames {
					if col.Name == name {
						col.IsPrimaryKey = true
						table.Cols[key] = col
						log.Printf("Primary key found: %s", col.Name)
					}
				}
			}
			break
		}
	}
}

func (sheet *Sheet) LoadCells(limit int, offset int) {
	colNames := sheet.OrderedColNames()
	// TODO: Check if table.TableName and column names are valid somewhere
	casts := make([]string, 0, len(colNames))
	for _, name := range colNames {
		col := sheet.table.Cols[name]
		col.Cells = make([]Cell, limit)
		sheet.table.Cols[name] = col

		cast := fmt.Sprintf("\"%s\"::text, \"%s\" IS NOT NULL", name, name)
		casts = append(casts, cast)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s LIMIT $1 OFFSET $2",
		strings.Join(casts, ", "),
		sheet.table.FullName())
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	Check(err)

	sheet.table.RowCount = 0
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		Check(err)
		for i := 0; i < len(casts); i++ {
			val, _ := scanResult[2*i].(string)
			sheet.table.Cols[colNames[i]].Cells[sheet.table.RowCount] = Cell{
				val, scanResult[2*i+1].(bool),
			}
		}
		sheet.table.RowCount++
	}
	log.Printf("Retrieved %d rows from %s", sheet.table.RowCount, sheet.table.FullName())
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
		sheet.table.FullName(),
		strings.Join(maps.Keys(nonEmptyValues), ", "),
		strings.Join(valueLabels, ", "))
	log.Println("Executing:", query)
	log.Println("Values:", nonEmptyValues)
	_, err := conn.NamedExec(query, nonEmptyValues)
	return err
}
