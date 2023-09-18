package main

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

func WriteError(w http.ResponseWriter, text string) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("<span class=\"error\">" + text + "</span>"))
}

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

		if tableName == "" {
			WriteError(w, "No table name provided")
			return
		}

		t := tableMap[tableName]
		SetCols(conn, &t)
		SetConstraints(conn, &t)
		SetPrefs(conn, t)

		if r.Method == "POST" {
			colName := r.FormValue("column")
			if colName != "" {
				col := t.Cols[colName]
				col.Config.Hide = r.FormValue("hide") == "true"
				t.Cols[colName] = col
				WritePref(conn, t, col.Config)
			} else {
				values := make(map[string]string)
				for key, value := range r.Form {
					colName, found := strings.CutPrefix(key, "column-")
					if found {
						values[colName] = value[0]
					}
				}

				err := InsertRow(conn, t, values)
				if err != nil {
					WriteError(w, err.Error())
					return
				}
			}
		}

		cells := GetRows(conn, t, 100, 0)
		handler := templ.Handler(RenderTable(t, cells))
		handler.ServeHTTP(w, r)
	}
	http.HandleFunc("/table", handleTable)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.Handle("/", templ.Handler(index(tables)))

	http.ListenAndServe(":8080", nil)
}
