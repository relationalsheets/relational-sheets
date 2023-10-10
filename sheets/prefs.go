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
}

func InitPrefsTable() {
	conn.MustExec(`
		DROP TABLE db_interface.column_prefs;
		CREATE TABLE IF NOT EXISTS db_interface.column_prefs (
			id SERIAL PRIMARY KEY
			, sheet_id INT NOT NULL
			, tablename VARCHAR(255) NOT NULL
			, columnname VARCHAR(255) NOT NULL
			, hide boolean NOT NULL DEFAULT false
			, editable boolean NOT NULL DEFAULT false
			, index int NOT NULL
			, UNIQUE(sheet_id, tablename, columnname)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id)
		)`)
	log.Println("Column prefs table exists")
}

func (sheet *Sheet) SetPref(colName string, hide bool) {
	pref := sheet.prefsMap[colName]
	pref.Hide = hide
	sheet.prefsMap[colName] = pref

	conn.MustExec(`
		INSERT INTO db_interface.column_prefs (
			sheet_id
			, tablename
			, columnname
			, hide
			, editable
			, index
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT ("sheet_id", "tablename", "columnname") DO
		UPDATE SET hide = $3
			, editable = $4
			, index = $5`,
		sheet.Id,
		colName,
		pref.Hide,
		pref.Editable,
		pref.Index)
}

func (s *Sheet) loadPrefs() {
	prefs := []Pref{}
	err := conn.Select(&prefs, `
		SELECT tablename
		    , columnname
			, hide
			, editable
			, index
		FROM db_interface.column_prefs
		WHERE sheet_id = $1`,
		s.Id)
	Check(err)
	log.Printf("Retrieved %d column prefs", len(prefs))
	s.prefsMap = make(map[string]Pref)
	for _, pref := range prefs {
		s.prefsMap[pref.TableName+"."+pref.ColumnName] = pref
	}
}
