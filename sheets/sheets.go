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
	Table     *Table
	JoinOids  pq.Int64Array
	PrefsMap  map[string]Pref
	ExtraCols []SheetColumn
	RowCount  int
}

var SheetMap = make(map[int]Sheet)

const defaultColNameChars string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (s Sheet) VisibleName() string {
	if s.Name == "" {
		return "Untitled Sheet"
	}
	return s.Name
}

func initSheetsTable() {
	conn.MustExec(`
		CREATE SCHEMA IF NOT EXISTS db_interface;
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
	if s.Table == nil {
		return ""
	}
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
	loadTables()
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
		var tableName, schemaName string
		err = rows.Scan(&sheet.Id, &sheet.Name, &tableName, &schemaName, &sheet.JoinOids)
		Check(err)
		sheet.Table = TableMap[schemaName+"."+tableName]
		SheetMap[sheet.Id] = sheet
		log.Printf("Loaded sheet: %+v", sheet)
	}
	log.Printf("Loaded %d sheets", len(SheetMap))
}

func (s *Sheet) LoadSheet() {
	log.Printf("Loading sheet %d for table %s", s.Id, s.TableFullName())
	s.Table = TableMap[s.TableFullName()]
	s.loadPrefs()
	s.LoadRows(100, 0)
	s.loadExtraCols()
	SheetMap[s.Id] = *s
}

func (s *Sheet) SetTable(name string) {
	s.Table = TableMap[name]
	s.SaveSheet()
	s.LoadSheet()
}
