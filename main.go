package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/a-h/templ"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var conn *sqlx.DB

func Open() {
	var err error
	conn, err = sqlx.Open("pgx", os.Getenv("DATABASE_URL"))
	check(err)
}

func WriteError(w http.ResponseWriter, text string) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("<span class=\"error\">" + text + "</span>"))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	sheetIdStr := r.URL.Query().Get("sheet_id")
	sheetId := int64(0)
	var err error
	if sheetIdStr != "" {
		sheetId, err = strconv.ParseInt(sheetIdStr, 10, 64)
		check(err)
	}
	globalSheet = sheetMap[sheetId]
	templ.Handler(index(sheetMap, sheetId, tables)).ServeHTTP(w, r)
}

func main() {
	Open()
	defer conn.Close()

	InitSheetsTables()
	InitPrefsTable()

	GetTables()
	LoadSheets()

	http.HandleFunc("/table", handleTable)
	http.HandleFunc("/sheet", handleSheet)
	http.HandleFunc("/add-column", handleAddColumn)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handleIndex)

	check(http.ListenAndServe(":8080", nil))
}
