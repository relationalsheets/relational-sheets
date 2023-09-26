package main

import (
	"acb/db-interface/sheets"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/http"
)

func main() {
	conn := sheets.Open()
	defer conn.Close()

	sheets.InitSheetsTables()
	sheets.InitPrefsTable()

	sheets.GetTables()
	sheets.LoadSheets()

	http.HandleFunc("/sheet", handleSheet)
	http.HandleFunc("/table", handleSetTable)
	http.HandleFunc("/add-row", handleAddRow)
	http.HandleFunc("/add-column", handleAddCol)
	http.HandleFunc("/rename-column", handleRenameCol)
	http.HandleFunc("/set-column-prefs", handleSetColPref)
	http.HandleFunc("/set-cell", handleSetCell)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handleIndex)

	http.ListenAndServe(":8080", nil)
}
