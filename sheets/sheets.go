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

func (s Sheet) RowCount() int {
	return s.table.RowCount
}

func initSheetsTable() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheets (
			id SERIAL PRIMARY KEY
			, "name" VARCHAR(255) NOT NULL
			, schemaname VARCHAR(255) NOT NULL
			, tablename VARCHAR(255) NOT NULL
		)`)
	log.Println("Sheets table exists")
}

func InitSheetsTables() {
	initSheetsTable()
	initExtraColsTables()
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
	s.table = tableMap[s.TableFullName()]
	s.table.loadCols()
	s.table.loadConstraints()
	s.loadPrefs()
	s.LoadCells(100, 0)
	s.loadExtraCols()
}

func (s *Sheet) SetTable(name string) {
	s.table = tableMap[name]
	s.SaveSheet()
	s.LoadSheet()
}
