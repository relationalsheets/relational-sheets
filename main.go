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
	http.HandleFunc("/table", withSheet(handleSetTable, true))
	http.HandleFunc("/new-row", withSheet(handleNewRow, true))
	http.HandleFunc("/add-row", withSheet(handleAddRow, true))
	http.HandleFunc("/add-column", withSheet(handleAddCol, true))
	http.HandleFunc("/rename-column", withSheet(handleRenameCol, true))
	http.HandleFunc("/set-column-prefs", withSheet(handleSetColPref, true))
	http.HandleFunc("/unhide-columns", withSheet(handleUnhideCols, true))
	http.HandleFunc("/set-cell", withSheet(handleSetCell, true))
	http.HandleFunc("/set-extra-cell", withSheet(handleSetExtraCell, true))
	http.HandleFunc("/set-name", withSheet(handleSetName, true))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", withSheet(handleIndex, false))

	http.ListenAndServe(":8080", nil)
}
