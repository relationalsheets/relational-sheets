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

func handleSheet(w http.ResponseWriter, r *http.Request) {
	sheet, err := getSheet(r, true)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	sheet.LoadSheet()
	sheets.GlobalSheet = sheet
	reRenderSheet(w, r)
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
	handler := templ.Handler(renderSheet(sheets.GlobalSheet))
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

	err := sheets.GlobalSheet.InsertRow(values)
	sheets.GlobalSheet.LoadCells(100, 0)
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
	sheets.GlobalSheet = sheet
	if sheets.GlobalSheet.Id != 0 {
		sheets.GlobalSheet.LoadSheet()
	}
	templ.Handler(index(sheets.GlobalSheet, sheets.SheetMap)).ServeHTTP(w, r)
}

func handleSetColPref(w http.ResponseWriter, r *http.Request) {
	colName := r.FormValue("col_name")
	if colName == "" {
		writeError(w, "Missing required key: col_name")
	}
	hide := r.FormValue("hide") == "true"
	sheets.GlobalSheet.SetPref(colName, hide)
	sheets.GlobalSheet.LoadCells(100, 0)

	reRenderSheet(w, r)
}

func handleModal(w http.ResponseWriter, r *http.Request) {
	sheet, err := getSheet(r, false)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	tableName := r.FormValue("table_name")
	if tableName != "" {
		sheet.SetTable(tableName)
	}

	fkeyOidStrs, ok := r.Form["fkey"]
	if ok {
		sheet.JoinOids = make([]int64, len(fkeyOidStrs))
		for i, fkeyOidStr := range fkeyOidStrs {
			oid, err := strconv.ParseInt(fkeyOidStr, 10, 64)
			_, ok := sheet.Table.Fkeys[oid]
			if err != nil || !ok {
				writeError(w, "Invalid fkey Oid")
				return
			}
			sheet.JoinOids[i] = oid
		}
		sheet.SaveSheet()
	}

	_, addJoin := r.Form["add_join"]
	templ.Handler(modal(sheet, sheets.Tables, addJoin)).ServeHTTP(w, r)
}
