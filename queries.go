package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type TableNames struct {
	SchemaName string
	TableName  string
}

type Constraint struct {
	Name           string
	ConstraintType string
	Def            string
}

type Pref struct {
	ColumnName string // Only used for convenient loading
	Hide       bool
	Editable   bool
	Index      int
}

type Column struct {
	Name         string
	IsNullable   bool
	DataType     string
	IsPrimaryKey bool
	Index        int
	Config       Pref
}

type Table struct {
	*TableNames
	HasPrimaryKey bool
	Cols          map[string]Column
	Constraints   []Constraint
}

func (table Table) FullName() string {
	return fmt.Sprintf("%s.%s", table.SchemaName, table.TableName)
}

func (table Table) OrderedCols() []Column {
	cols := make([]Column, 0, len(table.Cols))
	for _, col := range table.Cols {
		if !col.Config.Hide {
			cols = append(cols, col)
		}
	}
	sort.SliceStable(cols, func(i, j int) bool {
		indexI := cols[i].Config.Index | cols[i].Index
		indexJ := cols[j].Config.Index | cols[j].Index
		return indexI < indexJ
	})
	return cols
}

type Cell struct {
	Value string
	Null  bool
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func stringsToInterfaces(sx []string) []interface{} {
	interfaces := make([]interface{}, len(sx))
	for i, s := range sx {
		interfaces[i] = s
	}
	return interfaces
}

func Open() *sqlx.DB {
	conn, err := sqlx.Open("pgx", os.Getenv("DATABASE_URL"))
	check(err)
	return conn
}

func GetTables(conn *sqlx.DB) []Table {
	tables := []Table{}
	err := conn.Select(&tables, `
		SELECT COALESCE(tablename, '') tablename
			, COALESCE(schemaname, '') schemaname
		FROM pg_catalog.pg_tables
		WHERE schemaname != 'pg_catalog'
			AND schemaname != 'information_schema'
			AND schemaname != 'db_interface'
		ORDER BY schemaname, tablename DESC`)
	check(err)
	log.Printf("Retrieved %d tables", len(tables))
	return tables
}

func InitPrefsTable(conn *sqlx.DB) {
	conn.MustExec(`
		CREATE SCHEMA IF NOT EXISTS db_interface;
		DROP TABLE db_interface.column_prefs;
		CREATE TABLE IF NOT EXISTS db_interface.column_prefs (
			schemaname VARCHAR(255) NOT NULL
			, tablename VARCHAR(255) NOT NULL
			, columnname VARCHAR(255) NOT NULL
			, hide boolean NOT NULL DEFAULT false
			, editable boolean NOT NULL DEFAULT false
			, index int NOT NULL
			, UNIQUE(schemaname, tablename, columnname)
		)`)
	log.Println("Column prefs table exists")
}

func WritePref(conn *sqlx.DB, table Table, pref Pref) {
	conn.MustExec(`
		INSERT INTO db_interface.column_prefs (
			schemaname
			, tablename
			, columnname
			, hide
			, editable
			, index
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT ("schemaname", "tablename", "columnname") DO
		UPDATE SET hide = $4
			, editable = $5
			, index = $6`,
		table.SchemaName,
		table.TableName,
		pref.ColumnName,
		pref.Hide,
		pref.Editable,
		pref.Index)
}

func SetPrefs(conn *sqlx.DB, table Table) {
	prefs := []Pref{}
	err := conn.Select(&prefs, `
		SELECT columnname
			, hide
			, editable
			, index
		FROM db_interface.column_prefs
		WHERE schemaname = $1
			AND tablename = $2`,
		table.SchemaName,
		table.TableName)
	check(err)
	log.Printf("Retrieved %d column prefs", len(prefs))
	prefsMap := make(map[string]Pref)
	for _, pref := range prefs {
		prefsMap[pref.ColumnName] = pref
	}
	for key, col := range table.Cols {
		// Note the key may not be present, but the zero value is
		// the default config, except we need to set ColumnName
		config := prefsMap[col.Name]
		config.ColumnName = col.Name
		col.Config = config
		table.Cols[key] = col
	}
}

func SetCols(conn *sqlx.DB, table *Table) {
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
	check(err)
	log.Printf("Retrieved %d columns from %s", len(cols), table.FullName())
	table.Cols = make(map[string]Column)
	for _, col := range cols {
		table.Cols[col.Name] = col
	}
}

func SetConstraints(conn *sqlx.DB, table *Table) {
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
	check(err)
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

func GetRows(conn *sqlx.DB, table Table, limit int, offset int) [][]Cell {
	cells := make([][]Cell, 0, limit)
	cols := table.OrderedCols()
	// TODO: check if table.TableName and column names are valid somewhere
	casts := make([]string, 0, len(cols))
	for i := 0; i < len(cols); i++ {
		col := cols[i]
		cast := fmt.Sprintf("\"%s\"::text, \"%s\" IS NULL", col.Name, col.Name)
		casts = append(casts, cast)
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s LIMIT $1 OFFSET $2",
		strings.Join(casts, ", "),
		table.FullName())
	log.Printf("Executing: %s", query)
	rows, err := conn.Queryx(query, limit, offset)
	check(err)
	for rows.Next() {
		scanResult, err := rows.SliceScan()
		check(err)
		row := make([]Cell, len(casts))
		for i := 0; i < len(casts); i++ {
			val, _ := scanResult[2*i].(string)
			row[i] = Cell{val, scanResult[2*i+1].(bool)}
		}
		cells = append(cells, row)
	}
	log.Printf("Retrieved %d rows from %s", len(cells), table.FullName())
	check(rows.Close())
	return cells
}

func InsertRow(conn *sqlx.DB, table Table, values map[string]string) error {
	cols := table.OrderedCols()
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
		table.FullName(),
		strings.Join(colNames, ", "),
		strings.Join(valueLabels, ", "))
	log.Println("Executing:", query)
	log.Println("Values:", nonEmptyValues)
	_, err := conn.NamedExec(query, nonEmptyValues)
	return err
}
