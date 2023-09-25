package main

import (
	"log"
)

type Pref struct {
	ColumnName string // Only used for convenient loading
	Hide       bool
	Editable   bool
	Index      int
}

func InitPrefsTable() {
	conn.MustExec(`
		CREATE SCHEMA IF NOT EXISTS db_interface;
		CREATE TABLE IF NOT EXISTS db_interface.column_prefs (
			id SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, columnname VARCHAR(255) NOT NULL
			, hide boolean NOT NULL DEFAULT false
			, editable boolean NOT NULL DEFAULT false
			, index int NOT NULL
			, UNIQUE(sheet_id, columnname)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("Column prefs table exists")
}

func WritePref(sheet Sheet, colName string) {
	pref := sheet.prefsMap[colName]
	conn.MustExec(`
		INSERT INTO db_interface.column_prefs (
			sheet_id
			, columnname
			, hide
			, editable
			, index
		) VALUES (
			$1, $2, $3, $4, $5
		)
		ON CONFLICT ("sheet_id", "columnname") DO
		UPDATE SET hide = $3
			, editable = $4
			, index = $5`,
		sheet.Id,
		colName,
		pref.Hide,
		pref.Editable,
		pref.Index)
}

func (s *Sheet) LoadPrefs() {
	prefs := []Pref{}
	err := conn.Select(&prefs, `
		SELECT columnname
			, hide
			, editable
			, index
		FROM db_interface.column_prefs
		WHERE sheet_id = $1`,
		s.Id)
	check(err)
	log.Printf("Retrieved %d column prefs", len(prefs))
	s.prefsMap = make(map[string]Pref)
	for _, pref := range prefs {
		s.prefsMap[pref.ColumnName] = pref
	}
}
