package sheets

import (
	"log"
)

type SheetCell struct {
	Cell
	Formula string
}

type SheetColumn struct {
	Name  string
	Cells []SheetCell
}

type Sheet struct {
	Name      string
	Id        int64
	table     Table
	prefsMap  map[string]Pref
	ExtraCols []SheetColumn
}

var SheetMap = make(map[int64]Sheet)
var GlobalSheet Sheet

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
			, Formula INTEGER NOT NULL
			, UNIQUE (sheet_id, i, j)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("SheetCells table exists")
}

func (s Sheet) TableFullName() string {
	return s.table.FullName()
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
		Check(err)
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
	SheetMap[s.Id] = *s
}

func (s *Sheet) loadCells() {
	for i, col := range s.ExtraCols {
		col.Cells = make([]SheetCell, 100)
		s.ExtraCols[i] = col
	}

	rows, err := conn.Query(`
		SELECT i, j, Formula
		FROM db_interface.sheetcells
		WHERE sheet_id = $1
		ORDER BY i, j`,
		s.Id)
	Check(err)

	var formula string
	var i, j int
	for rows.Next() {
		err = rows.Scan(&i, &j, &formula)
		s.ExtraCols[i].Cells[j] = s.EvalFormula(formula)
	}

	log.Println("Loaded custom column Cells")
}

func (s *Sheet) loadCols() {
	s.ExtraCols = make([]SheetColumn, 0, 20)
	err := conn.Select(&s.ExtraCols, `
		SELECT colname AS "name"
		FROM db_interface.sheetcols
		WHERE sheet_id = $1
		ORDER BY i`,
		s.Id)
	Check(err)
	log.Printf("Loaded %d custom columns", len(s.ExtraCols))

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
		s.ExtraCols[i].Name)
}

func LoadSheets() {
	rows, err := conn.Query(`
		SELECT id
		     , "name"
		     , tableName
		     , schemaname
		FROM db_interface.sheets`)
	Check(err)
	for rows.Next() {
		sheet := Sheet{}
		err = rows.Scan(&sheet.Id, &sheet.Name, &sheet.table.TableName, &sheet.table.SchemaName)
		Check(err)
		SheetMap[sheet.Id] = sheet
		log.Printf("Loaded sheet: %+v", sheet)
	}
	log.Printf("Loaded %d sheets", len(SheetMap))
}

func (s *Sheet) LoadSheet() {
	SetCols(&s.table)
	SetConstraints(&s.table)
	s.loadPrefs()
	s.loadCols()
}

func (s *Sheet) SetCell(i, j int, formula string) SheetCell {
	column := s.ExtraCols[i]
	column.Cells[j] = s.EvalFormula(formula)
	conn.MustExec(`
		INSERT INTO db_interface.sheetcells (
		    sheet_id
		    , i
		    , j
		    , Formula
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (sheet_id, i, j) DO
		UPDATE SET Formula = $4`,
		s.Id,
		i,
		j,
		formula)
	return column.Cells[j]
}

func (s *Sheet) AddColumn(name string) {
	if name == "" {
		i := len(GlobalSheet.ExtraCols)
		for i >= 0 {
			name += defaultColNameChars[i%len(defaultColNameChars) : i%len(defaultColNameChars)+1]
			i -= len(defaultColNameChars)
		}
	}
	log.Printf("Adding column %s to sheet %d", name, s.Id)

	cells := make([]SheetCell, 100)
	s.ExtraCols = append(s.ExtraCols, SheetColumn{name, cells})
	s.SaveCol(len(GlobalSheet.ExtraCols) - 1)
}

func (s *Sheet) RenameCol(i int, name string) {
	col := GlobalSheet.ExtraCols[i]
	col.Name = name
	s.ExtraCols[i] = col
	s.SaveCol(i)
}

func (s *Sheet) SetTable(name string) {
	s.table = tableMap[name]
	s.SaveSheet()
	s.LoadSheet()
}
