package sheets

import (
	"github.com/lib/pq"
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
	Id        int
	Table     Table
	JoinOids  pq.Int64Array
	prefsMap  map[string]Pref
	ExtraCols []SheetColumn
}

var SheetMap = make(map[int]Sheet)
var GlobalSheet Sheet

const defaultColNameChars string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (s Sheet) VisibleName() string {
	if s.Name == "" {
		return "Untitled Sheet"
	}
	return s.Name
}

func (s Sheet) RowCount() int {
	return s.Table.RowCount
}

func initSheetsTable() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.sheets (
			id SERIAL PRIMARY KEY
			, "name" VARCHAR(255) NOT NULL
			, schemaname VARCHAR(255) NOT NULL
			, tablename VARCHAR(255) NOT NULL
		    , joinoids INTEGER ARRAY
		)`)
	log.Println("Sheets table exists")
}

func InitSheetsTables() {
	initSheetsTable()
	initExtraColsTables()
}

func (s Sheet) TableFullName() string {
	return s.Table.FullName()
}

func (s *Sheet) SaveSheet() {
	if s.Id == 0 {
		row := conn.QueryRow(`
			INSERT INTO db_interface.sheets (
				"name"
				, schemaname
				, tablename
				, joinoids
			) VALUES (
				$1, $2, $3, $4
			) RETURNING id`,
			s.Name,
			s.Table.SchemaName,
			s.Table.TableName,
			s.JoinOids)
		err := row.Scan(&s.Id)
		Check(err)
		log.Printf("Inserted sheet %d", s.Id)
	} else {
		conn.MustExec(`
			UPDATE db_interface.sheets SET
				"name" = $1
				, schemaname = $2
				, tablename = $3
			    , joinoids = $4
			WHERE id = $5`,
			s.Name,
			s.Table.SchemaName,
			s.Table.TableName,
			s.JoinOids,
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
			 , joinoids
		FROM db_interface.sheets`)
	Check(err)
	for rows.Next() {
		sheet := Sheet{}
		err = rows.Scan(&sheet.Id, &sheet.Name, &sheet.Table.TableName, &sheet.Table.SchemaName, &sheet.JoinOids)
		Check(err)
		SheetMap[sheet.Id] = sheet
		log.Printf("Loaded sheet: %+v", sheet)
	}
	log.Printf("Loaded %d sheets", len(SheetMap))
}

func (s *Sheet) LoadSheet() {
	s.Table = tableMap[s.TableFullName()]
	s.Table.loadCols()
	s.Table.loadConstraints()
	s.loadPrefs()
	s.LoadCells(100, 0)
	s.loadExtraCols()
}

func (s *Sheet) SetTable(name string) {
	s.Table = tableMap[name]
	s.SaveSheet()
	s.LoadSheet()
}
