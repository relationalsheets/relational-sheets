package main

import (
	"net/http"

	"github.com/a-h/templ"
)

func main() {
	conn := Open()
	defer conn.Close()

	tables := GetTables(conn)
	tableMap := make(map[string]Table)
	for _, table := range tables {
		tableMap[table.SchemaName+"."+table.TableName] = table
	}

	InitPrefsTable(conn)

	handleTable := func(w http.ResponseWriter, r *http.Request) {
		tableName := ""
		if r.Method == "POST" {
			tableName = r.FormValue("name")
		} else {
			tableName = r.URL.Query().Get("name")
		}
		t := tableMap[tableName]
		SetCols(conn, &t)
		SetConstraints(conn, &t)
		SetPrefs(conn, t)

		if r.Method == "POST" {
			colName := r.FormValue("column")
			col := t.Cols[colName]
			col.Config.Hide = r.FormValue("hide") == "true"
			t.Cols[colName] = col
			WritePref(conn, t, col.Config)
		}

		cells := GetRows(conn, t, 100, 0)
		handler := templ.Handler(table(t, cells))
		handler.ServeHTTP(w, r)
	}
	http.HandleFunc("/table", handleTable)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.Handle("/", templ.Handler(index(tables)))

	http.ListenAndServe(":8080", nil)
}
