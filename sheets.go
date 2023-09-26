package main

import (
	"github.com/a-h/templ"
	"log"
	"net/http"
	"strconv"
)

type SheetCell struct {
	Cell
	formula string
}

type SheetColumn struct {
	Name  string
	cells []SheetCell
}

type Sheet struct {
	Name      string
	Id        int64
	table     Table
	prefsMap  map[string]Pref
	extraCols []SheetColumn
}

var sheetMap = make(map[int64]Sheet)
var globalSheet Sheet

const defaultColNameChars string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (s Sheet) VisibleName() string {
	if s.Name == "" {
		return "Untitled Sheet"
	}
	return s.Name
}

func InitSheetsTables() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheets (
			id SERIAL PRIMARY KEY
			, "name" VARCHAR(255) NOT NULL
			, schemaname VARCHAR(255) NOT NULL
			, tablename VARCHAR(255) NOT NULL
		)`)
	log.Println("Sheets table exists")

	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheetcols (
			id SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, i INTEGER NOT NULL
			, colname VARCHAR(255) NOT NULL
			, UNIQUE (sheet_id, i)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("SheetCols table exists")

	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheetcells (
			id SERIAL PRIMARY KEY
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

func (s *Sheet) SaveSheet() {
	if s.Id == 0 {
		row := conn.QueryRow(`
			INSERT INTO db_interface.sheets (
				"name"
				, schemaname
				, tablename
			) VALUES (
				$1, $2, $3
			) RETURNING id`,
			s.Name,
			s.table.SchemaName,
			s.table.TableName)
		err := row.Scan(&s.Id)
		check(err)
		log.Printf("Inserted sheet %d", s.Id)
	} else {
		conn.MustExec(`
			UPDATE db_interface.sheets SET
				"name" = $1
				, schemaname = $2
				, tablename = $3
			WHERE id = $4`,
			s.Name,
			s.table.SchemaName,
			s.table.TableName,
			s.Id)
		log.Printf("Updated sheet %d", s.Id)
	}
}

func (s *Sheet) loadCells() {
	for i, col := range s.extraCols {
		col.cells = make([]SheetCell, 100)
		s.extraCols[i] = col
	}

	rows, err := conn.Query(`
		SELECT i, j, formula
		FROM db_interface.sheetcells
		WHERE sheet_id = $1
		ORDER BY i, j`,
		s.Id)
	check(err)

	var formula string
	var i, j int
	for rows.Next() {
		err = rows.Scan(&i, &j, &formula)
		s.extraCols[i].cells[j] = s.EvalFormula(formula)
	}

	log.Println("Loaded custom column cells")
}

func (s *Sheet) LoadCols() {
	s.extraCols = make([]SheetColumn, 0, 20)
	err := conn.Select(&s.extraCols, `
		SELECT colname AS "name"
		FROM db_interface.sheetcols
		WHERE sheet_id = $1
		ORDER BY i`,
		s.Id)
	check(err)
	log.Printf("Loaded %d custom columns", len(s.extraCols))

	s.loadCells()
}

func (s *Sheet) SaveCol(i int) {
	conn.MustExec(`
		INSERT INTO db_interface.sheetcols (
			sheet_id
			, i
			, colname
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (sheet_id, i) DO
		UPDATE SET colname = $3`,
		s.Id,
		i,
		s.extraCols[i].Name)
}

func LoadSheets() {
	rows, err := conn.Query(`
		SELECT id
		     , "name"
		     , tableName
		     , schemaname
		FROM db_interface.sheets`)
	check(err)
	for rows.Next() {
		sheet := Sheet{}
		err = rows.Scan(&sheet.Id, &sheet.Name, &sheet.table.TableName, &sheet.table.SchemaName)
		check(err)
		sheetMap[sheet.Id] = sheet
		log.Printf("Loaded sheet: %+v", sheet)
	}
	log.Printf("Loaded %d sheets", len(sheetMap))
}

func (s *Sheet) LoadSheet() {
	SetCols(&s.table)
	SetConstraints(&s.table)
	s.LoadPrefs()
	s.LoadCols()
}

func (s *Sheet) SetCell(i, j int, formula string) SheetCell {
	column := s.extraCols[i]
	column.cells[j] = s.EvalFormula(formula)
	conn.MustExec(`
		INSERT INTO db_interface.sheetcells (
		    sheet_id
		    , i
		    , j
		    , formula
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (sheet_id, i, j) DO
		UPDATE SET formula = $4`,
		s.Id,
		i,
		j,
		formula)
	return column.cells[j]
}

func (s *Sheet) AddColumn(name string) {
	cells := make([]SheetCell, 100)
	s.extraCols = append(s.extraCols, SheetColumn{name, cells})
	log.Printf("Adding column to sheet %d", s.Id)
	conn.MustExec(`
		INSERT INTO db_interface.sheetcols (
			sheet_id
			, i
			, colname 
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (sheet_id, i) DO
		UPDATE SET colname=$3`,
		s.Id,
		len(s.extraCols)-1,
		name)
}

func handleSheet(w http.ResponseWriter, r *http.Request) {
	sheetName := r.Header.Get("HX-Prompt")
	sheet := Sheet{Name: sheetName}
	sheet.SaveSheet()
	sheetMap[sheet.Id] = sheet
	globalSheet = sheet
	templ.Handler(sheetSelect(sheetMap, globalSheet.Id)).ServeHTTP(w, r)
}

func handleAddCol(w http.ResponseWriter, r *http.Request) {
	colName := ""
	i := len(globalSheet.extraCols)
	for i >= 0 {
		colName += defaultColNameChars[i%len(defaultColNameChars) : i%len(defaultColNameChars)+1]
		i -= len(defaultColNameChars)
	}
	globalSheet.AddColumn(colName)

	cells := GetRows(globalSheet, 100, 0)
	handler := templ.Handler(RenderSheet(globalSheet, cells))
	handler.ServeHTTP(w, r)
}

func handleRenameCol(w http.ResponseWriter, r *http.Request) {
	colIndex, err := strconv.Atoi(r.FormValue("col_index"))
	check(err)
	col := globalSheet.extraCols[colIndex]
	col.Name = r.FormValue("col_name")
	globalSheet.extraCols[colIndex] = col
	globalSheet.SaveCol(colIndex)
	w.WriteHeader(http.StatusNoContent)
}

func handleSetCell(w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(r.FormValue("i"))
	check(err)
	j, err := strconv.Atoi(r.FormValue("j"))
	check(err)
	formula := r.FormValue("formula")

	cell := globalSheet.SetCell(i, j, formula)

	handler := templ.Handler(extraCell(i, j, cell))
	handler.ServeHTTP(w, r)
}
