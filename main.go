package main

import (
	"net/http"
	"os"

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

func main() {
	Open()
	defer conn.Close()

	tables := GetTables()
	for _, table := range tables {
		tableMap[table.FullName()] = table
	}

	InitSheetsTables()
	InitPrefsTable()

	http.HandleFunc("/table", handleTable)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.Handle("/", templ.Handler(index(tables)))

	http.ListenAndServe(":8080", nil)
}
