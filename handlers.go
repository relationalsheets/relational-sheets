package main

import (
	"acb/db-interface/sheets"
	"errors"
	"github.com/a-h/templ"
	"net/http"
	"strconv"
	"strings"
)

func writeError(w http.ResponseWriter, text string) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("<span class=\"error\">" + text + "</span>"))
}

func getSheet(r *http.Request, required bool) (sheets.Sheet, error) {
	sheetIdStr := r.FormValue("sheet_id")
	if sheetIdStr != "" {
		sheetId, err := strconv.Atoi(sheetIdStr)
		return sheets.SheetMap[sheetId], err
	}
	if required {
		return sheets.Sheet{}, errors.New("missing sheet_id")
	}
	return sheets.Sheet{}, nil
}

func handleAddCol(w http.ResponseWriter, r *http.Request) {
	sheets.GlobalSheet.AddColumn("")
	reRenderSheet(w, r)
}

func handleRenameCol(w http.ResponseWriter, r *http.Request) {
	colIndex, err := strconv.Atoi(r.FormValue("col_index"))
	sheets.Check(err)
	sheets.GlobalSheet.RenameCol(colIndex, r.FormValue("col_name"))
	w.WriteHeader(http.StatusNoContent)
}

func handleSetCell(w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(r.FormValue("i"))
	sheets.Check(err)
	j, err := strconv.Atoi(r.FormValue("j"))
	sheets.Check(err)
	formula := r.FormValue("formula")

	cell := sheets.GlobalSheet.SetCell(i, j, formula)

	handler := templ.Handler(extraCell(i, j, cell))
	handler.ServeHTTP(w, r)
}

func reRenderSheet(w http.ResponseWriter, r *http.Request) {
	if sheets.GlobalSheet.TableFullName() == "" {
		writeError(w, "No table name provided")
		return
	}
	tableNames, cols := sheets.GlobalSheet.OrderedTablesAndCols(nil)
	cells := sheets.GlobalSheet.LoadRows(100, 0)
	numCols := 0
	for _, tcols := range cols {
		numCols += len(tcols)
	}
	component := sheetTable(sheets.GlobalSheet, tableNames, cols, cells, numCols)
	handler := templ.Handler(component)
	handler.ServeHTTP(w, r)
}

func handleSetTable(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		tableName := r.FormValue("table_name")
		if tableName == "" {
			writeError(w, "No table name provided")
			return
		}
		sheets.GlobalSheet.SetTable(tableName)
	}

	reRenderSheet(w, r)
}

func handleAddRow(w http.ResponseWriter, r *http.Request) {
	values := make(map[string]string)
	for key, value := range r.Form {
		colName, found := strings.CutPrefix(key, "column-")
		if found {
			values[colName] = value[0]
		}
	}

	tx := sheets.Begin()
	_, err := sheets.GlobalSheet.InsertRow(tx, r.FormValue("table_name"), values, []string{})
	if err != nil {
		writeError(w, err.Error())
		return
	}
	err = tx.Commit()
	if err != nil {
		writeError(w, err.Error())
		return
	}

	reRenderSheet(w, r)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	sheet, err := getSheet(r, false)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	if sheet.Id != 0 {
		sheet.LoadSheet()
	}
	sheets.GlobalSheet = sheet
	templ.Handler(index(sheet, sheets.SheetMap)).ServeHTTP(w, r)
}

func handleSetColPref(w http.ResponseWriter, r *http.Request) {
	colName := r.FormValue("col_name")
	if colName == "" {
		writeError(w, "Missing required key: col_name")
	}
	hide := r.FormValue("hide") == "true"
	sheets.GlobalSheet.SetPref(colName, hide)
	sheets.GlobalSheet.LoadRows(100, 0)

	reRenderSheet(w, r)
}

func handleModal(w http.ResponseWriter, r *http.Request) {
	sheet, err := getSheet(r, false)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	if sheet.Id != 0 {
		sheets.GlobalSheet = sheet
	}

	tableName := r.FormValue("table_name")
	if tableName != "" {
		sheets.GlobalSheet.SetTable(tableName)
	}

	fkeyOidStrs, ok := r.Form["fkey"]
	if ok {
		sheets.GlobalSheet.JoinOids = make([]int64, len(fkeyOidStrs))
		for i, fkeyOidStr := range fkeyOidStrs {
			oid, err := strconv.ParseInt(fkeyOidStr, 10, 64)
			_, ok := sheets.GlobalSheet.Table.Fkeys[oid]
			if err != nil || !ok {
				writeError(w, "Invalid fkey Oid")
				return
			}
			sheets.GlobalSheet.JoinOids[i] = oid
		}
		sheets.GlobalSheet.SaveSheet()
	}

	_, addJoin := r.Form["add_join"]
	templ.Handler(modal(sheets.GlobalSheet, sheets.TableMap, addJoin)).ServeHTTP(w, r)
}

func handleSetName(w http.ResponseWriter, r *http.Request) {
	sheets.GlobalSheet.Name = r.FormValue("name")
	sheets.GlobalSheet.SaveSheet()
	w.WriteHeader(204)
	w.Write([]byte{})
}
