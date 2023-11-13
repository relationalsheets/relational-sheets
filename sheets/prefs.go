// This file is part of Relational Sheets.
//
// Relational Sheets is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// Relational Sheets is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU General Public License along with Relational Sheets.
// If not, see https://www.gnu.org/licenses/agpl-3.0.html
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
	Filter	   string
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
			, filter VARCHAR(255) NOT NULL
			, UNIQUE(sheet_id, tablename, columnname)
			, CONSTRAINT fk_sheets
				FOREIGN KEY (sheet_id)
					REFERENCES db_interface.sheets(id) ON DELETE CASCADE
		)`)
	log.Println("Column prefs table exists")
}

func (sheet *Sheet) SavePref(pref Pref) {
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
			, filter
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT ("sheet_id", "tablename", "columnname") DO
		UPDATE SET hide = $4
			, editable = $5
			, index = $6
			, sorton = $7
			, ascending = $8
			, filter = $9`,
		sheet.Id,
		pref.TableName,
		pref.ColumnName,
		pref.Hide,
		pref.Editable,
		pref.Index,
		pref.SortOn,
		pref.Ascending,
		pref.Filter)
}

func (s *Sheet) LoadPrefs() {
	prefs := []Pref{}
	err := conn.Select(&prefs, `
		SELECT tablename
		    , columnname
			, hide
			, editable
			, index
			, sorton
			, ascending
			, filter
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
