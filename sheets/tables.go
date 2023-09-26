package sheets

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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
}

type Table struct {
	TableNames
	HasPrimaryKey bool
	Cols          map[string]Column
	Constraints   []Constraint
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

func (sheet Sheet) OrderedCols() []Column {
	cols := make([]Column, 0, len(sheet.table.Cols))
	for _, col := range sheet.table.Cols {
		if !sheet.prefsMap[col.Name].Hide {
			cols = append(cols, col)
		}
	}
	sort.SliceStable(cols, func(i, j int) bool {
		indexI := sheet.prefsMap[cols[i].Name].Index | cols[i].Index
		indexJ := sheet.prefsMap[cols[j].Name].Index | cols[j].Index
		return indexI < indexJ
	})
	return cols
}

func GetTables() {
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
		SELECT conname "name"
			, contype constrainttype
     		, pg_get_constraintdef(oid) def
		FROM pg_constraint
		WHERE conrelid = $1::regclass
			AND connamespace = $2::regnamespace`,
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

func GetRows(sheet Sheet, limit int, offset int) [][]Cell {
	cells := make([][]Cell, 0, limit)
	cols := sheet.OrderedCols()
	// TODO: Check if table.TableName and column names are valid somewhere
	casts := make([]string, 0, len(cols))
	for i := 0; i < len(cols); i++ {
		col := cols[i]
		cast := fmt.Sprintf(
			"\"%s\"::text, \"%s\" IS NOT NULL",
			col.Name,
			col.Name)
		casts = append(casts, cast)
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s LIMIT $1 OFFSET $2",
		strings.Join(casts, ", "),
		sheet.table.FullName())
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	Check(err)
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		Check(err)
		row := make([]Cell, len(casts))
		for i := 0; i < len(casts); i++ {
			val, _ := scanResult[2*i].(string)
			row[i] = Cell{val, scanResult[2*i+1].(bool)}
		}
		cells = append(cells, row)
	}
	log.Printf("Retrieved %d rows from %s", len(cells), sheet.table.FullName())
	Check(rows.Close())
	return cells
}

func InsertRow(sheet Sheet, values map[string]string) error {
	cols := sheet.OrderedCols()
	colNames := make([]string, 0, len(cols))
	valueLabels := make([]string, 0, len(cols))
	nonEmptyValues := make(map[string]interface{})
	for _, col := range cols {
		if values[col.Name] != "" {
			nonEmptyValues[col.Name] = values[col.Name]
			colNames = append(colNames, col.Name)
			valueLabels = append(valueLabels, ":"+col.Name)
		}
	}
	if len(nonEmptyValues) == 0 {
		return errors.New("All fields are empty")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		sheet.table.FullName(),
		strings.Join(colNames, ", "),
		strings.Join(valueLabels, ", "))
	log.Println("Executing:", query)
	log.Println("Values:", nonEmptyValues)
	_, err := conn.NamedExec(query, nonEmptyValues)
	return err
}
