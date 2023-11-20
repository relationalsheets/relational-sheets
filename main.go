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
package main

import (
	"acb/db-interface/sheets"
	"net/http"
	"os"
	"slices"
)

func main() {
	conn := sheets.Open()
	defer conn.Close()

	createExampleTable := slices.Contains(os.Args[1:], "--create-example-tables")
	if createExampleTable {
		sheets.SetupTablesDB()
		sheets.LoadExampleData()
	}

	sheets.InitSheetsTables()
	sheets.InitPrefsTable()
	sheets.CreateAggregates()

	sheets.LoadSheets()

	http.HandleFunc("/modal", withSheet(handleModal, false))
	http.HandleFunc("/table", withSheetAndLimit(handleSetTable))
	http.HandleFunc("/new-row", withSheetAndLimit(handleNewRow))
	http.HandleFunc("/add-row", withSheetAndLimit(handleAddRow))
	http.HandleFunc("/add-column", withSheetAndLimit(handleAddCol))
	http.HandleFunc("/rename-column", withSheetAndLimit(handleRenameCol))
	http.HandleFunc("/delete-column", withSheetAndLimit(handleDeleteCol))
	http.HandleFunc("/set-column-prefs", withSheetAndLimit(handleSetColPref))
	http.HandleFunc("/unhide-columns", withSheetAndLimit(handleUnhideCols))
	http.HandleFunc("/clear-filters", withSheetAndLimit(handleClearFilters))
	http.HandleFunc("/set-cell", withSheet(handleSetCell, true))
	http.HandleFunc("/set-extra-cell", withSheet(handleSetExtraCell, true))
	http.HandleFunc("/set-name", withSheet(handleSetName, true))
	http.HandleFunc("/fill-column-down", withSheetAndLimit(handleFillColumnDown))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", withSheet(handleIndex, false))

	port := os.Getenv("RS_PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
