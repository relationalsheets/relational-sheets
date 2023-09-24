package main

import (
	"log"
	"net/http"
)

type SheetCell struct {
	Cell
	formula string
}

type SheetColumn struct {
	name  string
	cells []SheetCell
}

type Sheet struct {
	name         string
	id           int
	table        Table
	prefsMap     map[string]Pref
	extraColumns []SheetColumn
}

func InitSheetsTables() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheets (
			id SERIAL PRIMARY KEY
			, name VARCHAR(255) NOT NULL
			, schemaname VARCHAR(255) NOT NULL
			, tablename VARCHAR(255) NOT NULL
		)`)
	log.Println("Sheets table exists")

	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheetcells (
			int SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, i INTEGER NOT NULL
			, j INTEGER NOT NULL
			, formula INTEGER NOT NULL
			, UNIQUE (sheet_id, i, j)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("SheetCells table exists")
}

func CreateSheet(name string, table Table) {
	conn.MustExec(`
		INSERT INTO db_interface.sheets (
			name
			, schemaname
			, tablename
		) VALUES (
			$1, $2, $3
		)`,
		name,
		table.SchemaName,
		table.TableName)
}

func GetSheets() []Sheet {
	sheets := []Sheet{}
	err := conn.Select(&sheets, "SELECT name FROM db_interface.sheets")
	check(err)
	return sheets
}

var globalSheet = Sheet{}

func GetSheet(name string) Sheet {
	// TODO
	return globalSheet
}

func (s *Sheet) SetCell(i, j int, formula string) {
	column := s.extraColumns[i]
	column.cells[j] = EvalFormula(*s, formula)
}

func (s *Sheet) AddColumn(name string) {
	cells := make([]SheetCell, 0, 100)
	s.extraColumns = append(s.extraColumns, SheetColumn{name, cells})
}

func handleAddColumn(w http.ResponseWriter, r *http.Request) {
	sheetName := r.FormValue("sheet_name")
	colName := r.FormValue("new_column_name")
	sheet := GetSheet(sheetName)
	sheet.AddColumn(colName)
}
