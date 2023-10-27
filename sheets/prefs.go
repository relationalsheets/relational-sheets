package sheets

import (
	"log"
)

type Pref struct {
	TableName  string // Only used for convenient loading
	ColumnName string // Only used for convenient loading
	Hide       bool
	Editable   bool
	Index      int
	SortOn     bool
	Ascending  bool
}

func InitPrefsTable() {
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS db_interface.column_prefs (
			id SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, tablename VARCHAR(255) NOT NULL
			, columnname VARCHAR(255) NOT NULL
			, hide boolean NOT NULL DEFAULT false
			, editable boolean NOT NULL DEFAULT false
			, index int NOT NULL
		    , sorton boolean NOT NULL
		    , ascending boolean NOT NULL
			, UNIQUE(sheet_id, tablename, columnname)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("Column prefs table exists")
}

func (sheet *Sheet) SetPref(pref Pref) {
	sheet.PrefsMap[pref.TableName+"."+pref.ColumnName] = pref

	conn.MustExec(`
		INSERT INTO db_interface.column_prefs (
			sheet_id
			, tablename
			, columnname
			, hide
			, editable
			, index
			, sorton
			, ascending
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT ("sheet_id", "tablename", "columnname") DO
		UPDATE SET hide = $4
			, editable = $5
			, index = $6
			, sorton = $7
			, ascending = $8`,
		sheet.Id,
		pref.TableName,
		pref.ColumnName,
		pref.Hide,
		pref.Editable,
		pref.Index,
		pref.SortOn,
		pref.Ascending)
}

func (s *Sheet) loadPrefs() {
	prefs := []Pref{}
	err := conn.Select(&prefs, `
		SELECT tablename
		    , columnname
			, hide
			, editable
			, index
			, sorton
			, ascending
		FROM db_interface.column_prefs
		WHERE sheet_id = $1`,
		s.Id)
	Check(err)
	log.Printf("Retrieved %d column prefs", len(prefs))
	s.PrefsMap = make(map[string]Pref)
	for _, pref := range prefs {
		s.PrefsMap[pref.TableName+"."+pref.ColumnName] = pref
	}
}
