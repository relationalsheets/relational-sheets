package main

import (
	"acb/db-interface/sheets"
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

func handleSheet(w http.ResponseWriter, r *http.Request) {
	sheetName := r.Header.Get("HX-Prompt")
	sheet := sheets.Sheet{Name: sheetName}
	sheet.SaveSheet()
	sheets.GlobalSheet = sheet
	templ.Handler(sheetSelect(sheets.SheetMap, sheets.GlobalSheet.Id)).ServeHTTP(w, r)
}

func handleAddCol(w http.ResponseWriter, r *http.Request) {
	sheets.GlobalSheet.AddColumn("")
	cells := sheets.GetRows(sheets.GlobalSheet, 100, 0)
	handler := templ.Handler(RenderSheet(sheets.GlobalSheet, cells))
	handler.ServeHTTP(w, r)
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
	cells := sheets.GetRows(sheets.GlobalSheet, 100, 0)
	handler := templ.Handler(RenderSheet(sheets.GlobalSheet, cells))
	handler.ServeHTTP(w, r)
}

func handleSetTable(w http.ResponseWriter, r *http.Request) {
	tableName := r.URL.Query().Get("table_name")
	if tableName == "" {
		writeError(w, "No table name provided")
		return
	}
	sheets.GlobalSheet.SetTable(tableName)

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

	err := sheets.InsertRow(sheets.GlobalSheet, values)
	if err != nil {
		writeError(w, err.Error())
		return
	}

	reRenderSheet(w, r)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	sheetIdStr := r.URL.Query().Get("sheet_id")
	sheetId := int64(0)
	if sheetIdStr != "" {
		sheetId, err := strconv.ParseInt(sheetIdStr, 10, 64)
		if err != nil {
			writeError(w, "Invalid sheet ID")
			return
		}
		sheets.GlobalSheet = sheets.SheetMap[sheetId]
		sheets.GlobalSheet.LoadSheet()
	}
	templ.Handler(index(sheets.SheetMap, sheetId, sheets.Tables)).ServeHTTP(w, r)
}

func handleSetColPref(w http.ResponseWriter, r *http.Request) {
	colName := r.FormValue("col_name")
	if colName == "" {
		writeError(w, "Missing required key: col_name")
	}
	hide := r.FormValue("hide") == "true"
	sheets.GlobalSheet.SetPref(colName, hide)

	reRenderSheet(w, r)
}
