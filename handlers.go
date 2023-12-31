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
	"acb/db-interface/fkeys"
	"acb/db-interface/sheets"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"golang.org/x/exp/maps"
)

func writeError(w http.ResponseWriter, text string) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("<span class=\"has-text-danger\">" + text + "</span>"))
}

func mustGetInt(r *http.Request, key string) int {
	str := r.FormValue(key)
	i, err := strconv.Atoi(str)
	sheets.Check(err)
	return i
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

func withSheet(f func(sheets.Sheet, http.ResponseWriter, *http.Request), required bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sheet, err := getSheet(r, required)
		if err != nil {
			writeError(w, err.Error())
			return
		}
		f(sheet, w, r)
	}
}

func withSheetAndLimit(f func(sheets.Sheet, int, http.ResponseWriter, *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	g := func (sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
		str := r.FormValue("limit")
		limit, err := strconv.Atoi(str)
		if str == "" {
			limit = 100
		} else if err != nil {
			writeError(w, err.Error())
			return
		}
		f(sheet, limit, w, r)
	}
	return withSheet(g, true)
}

func parseColFields(r *http.Request, prefix string) (map[string]map[string]string, error) {
	sheets.Check(r.ParseForm())
	values := make(map[string]map[string]string)
	for key, value := range r.Form {
		name, found := strings.CutPrefix(key, prefix)
		if found {
			parts := strings.Split(name, " ")
			if len(parts) != 2 {
				return nil, errors.New("Unexpected field name: " + name)
			}
			submap, ok := values[parts[0]]
			if !ok {
				submap = make(map[string]string)
				values[parts[0]] = submap
			}
			submap[parts[1]] = value[0]
		}
	}
	return values, nil
}

func getPKs(r *http.Request) map[string]map[string]string {
	pks, err := parseColFields(r, "pk-")
	sheets.Check(err)
	return pks
}

func handleAddCol(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	sheet.AddColumn("")
	reRenderSheet(sheet, limit, w, r)
}

func handleRenameCol(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	colIndex, err := strconv.Atoi(r.FormValue("col_index"))
	if err != nil {
		writeError(w, err.Error())
		return
	}
	sheet.RenameCol(colIndex, r.FormValue("col_name"))
	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteCol(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	colIndex, err := strconv.Atoi(r.FormValue("col_index"))
	if err != nil {
		writeError(w, err.Error())
		return
	}
	sheet.DeleteColumn(colIndex)
	reRenderSheet(sheet, limit, w, r)
}

func handleSetExtraCell(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	i := mustGetInt(r, "i")
	j := mustGetInt(r, "j")
	formula := r.FormValue("formula")

	cell, err := sheet.SetCell(i, j, formula)
	if err != nil {
		writeError(w, err.Error())
		return
	}

	handler := templ.Handler(extraCell(i, j, cell))
	handler.ServeHTTP(w, r)
}

func reRenderSheet(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	if sheet.TableFullName() == "" {
		writeError(w, "No table name provided")
		return
	}
	err := sheet.LoadRows(limit, 0)
	cols := sheet.OrderedCols(nil)
	numCols := 0
	for _, tcols := range cols {
		numCols += len(tcols)
	}
	component := sheetTable(sheet, cols, numCols, err)
	handler := templ.Handler(component)
	handler.ServeHTTP(w, r)
}

func handleSetTable(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		tableName := r.FormValue("table_name")
		if tableName == "" {
			writeError(w, "No table name provided")
			return
		}
		sheet.SetTable(tableName)
	}

	reRenderSheet(sheet, limit, w, r)
}

func handleNewRow(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	cols := sheet.OrderedCols(nil)
	tableName := r.FormValue("table_name")
	tableIndex := slices.Index(sheet.TableNames, tableName)
	numCols := 0
	for _, tcols := range cols {
		numCols += len(tcols)
	}

	rowIndex := 0
	row := []sheets.Cell{}
	if tableIndex > -1 {
		row = make([]sheets.Cell, len(cols[tableIndex]))
	}
	pks := getPKs(r)
	if len(pks[tableName]) > 0 {
		if tableIndex == -1 {
			writeError(w, "Unrecognized table")
			return
		}
		// TODO: cache
		err := sheet.LoadRows(limit, 0)
		if err != nil {
			writeError(w, err.Error())
			return
		}
		cells := sheet.Cells[tableIndex]
		for rowIndex, _ = range cells[0] {
			match := true
			for colIndex, col := range cols[tableIndex] {
				row[colIndex] = cells[colIndex][rowIndex]
				pkValue, ok := pks[tableName][col.Name]
				if ok && pkValue != row[colIndex].Value {
					match = false
					break
				}
			}
			if match {
				break
			}
		}
	}

	component := newRow(sheet.TableNames, tableName, cols, numCols, row, rowIndex)
	templ.Handler(component).ServeHTTP(w, r)
}

func handleAddRow(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	pks := getPKs(r)
	values, err := parseColFields(r, "column-")
	if err != nil {
		writeError(w, err.Error())
		return
	}

	err = sheet.InsertMultipleRows(values, pks)
	if err != nil {
		writeError(w, err.Error())
		return
	}

	reRenderSheet(sheet, limit, w, r)
}

func handleIndex(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	templ.Handler(index(sheet, sheets.SheetMap)).ServeHTTP(w, r)
}

func handleSetColPref(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	tableName := r.FormValue("table_name")
	colName := r.FormValue("col_name")
	if colName == "" {
		writeError(w, "Missing required key: col_name")
	}
	sheet.LoadPrefs()
	pref := sheet.PrefsMap[tableName+"."+colName]
	pref.TableName = tableName
	pref.ColumnName = colName
	// Either update filtering or sorting, never both
	filters, setFilter := r.Form["filter"]
	if setFilter {
		pref.Filter = filters[0]
	} else {
		pref.Hide = r.FormValue("hide") == "true"
		// reset sorting and filtering if the column is hidden
		// since these will no longer be SELECTed
		if pref.Hide {
			pref.SortOn = false
			pref.Ascending = false
			pref.Filter = ""
		} else {
			pref.SortOn = r.FormValue("sorton") == "true"
			pref.Ascending = r.FormValue("ascending") == "true"
		}
	}
	log.Printf("saving pref: %v", pref)
	sheet.SavePref(pref)

	reRenderSheet(sheet, limit, w, r)
}

func handleUnhideCols(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	for _, pref := range sheet.PrefsMap {
		pref.Hide = false
		sheet.SavePref(pref)
	}

	reRenderSheet(sheet, limit, w, r)
}

func handleClearFilters(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	for _, pref := range sheet.PrefsMap {
		pref.Filter = ""
		sheet.SavePref(pref)
	}

	reRenderSheet(sheet, limit, w, r)
}

func handleModal(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	tableName := r.FormValue("table_name")
	if tableName != "" {
		sheet.SetTable(tableName)
		sheet.LoadJoins()
	}

	tableNames := maps.Keys(sheets.TableMap)
	slices.Sort(tableNames)

	fkeyOidsSeen := make(map[int64]bool)
	options := make(map[string]map[int64]fkeys.ForeignKey)
	for _, name := range sheet.TableNames {
		options[name] = make(map[int64]fkeys.ForeignKey)
		for oid, fkey := range sheets.TableMap[name].Fkeys {
			if !fkeyOidsSeen[oid] {
				fkeyOidsSeen[oid] = true
				options[name][oid] = fkey
			}
		}
	}

	_, addJoin := r.Form["add_join"]
	if addJoin || r.Method != "POST" {
		log.Printf("Available fkeys: %v", options)
		templ.Handler(modal(sheet, tableNames, options, addJoin)).ServeHTTP(w, r)
		return
	}

	for name, value := range r.Form {
		fkeyIndexStr, ok := strings.CutPrefix(name, "fkey-")
		if !ok {
			continue
		}

		fkeyIndex, err := strconv.Atoi(fkeyIndexStr)
		if err != nil {
			writeError(w, "Invalid key "+fkeyIndexStr)
			return
		}
		oid, err := strconv.ParseInt(value[0], 10, 64)
		if err != nil {
			writeError(w, "Invalid integer "+value[0])
			return
		}

		err = sheet.SetJoin(fkeyIndex, oid)
		if err != nil {
			writeError(w, "No such fkey "+value[0])
			return
		}
	}

	templ.Handler(modal(sheet, tableNames, options, addJoin)).ServeHTTP(w, r)
}

func handleSetName(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	sheet.Name = r.FormValue("name")
	sheet.SaveSheet()
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte{})
}

func handleSetCell(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	tableName := r.FormValue("table_name")
	name := r.FormValue("col_name")
	value := r.FormValue("value")
	col := sheets.TableMap[tableName].Cols[name]
	rowStr := r.FormValue("row")
	row, err := strconv.Atoi(rowStr)
	sheets.Check(err)

	err = sheet.UpdateRows(
		map[string]map[string]string{tableName: {name: value}},
		map[string]map[string]string{tableName: getPKs(r)[tableName]})
	cell := tableCell(tableName, col, row, sheets.Cell{value, value != ""}, err)
	templ.Handler(cell).ServeHTTP(w, r)
}

func handleFillColumnDown(sheet sheets.Sheet, limit int, w http.ResponseWriter, r *http.Request) {
	i := mustGetInt(r, "i")
	j := mustGetInt(r, "j")
	formula := r.FormValue("formula")
	sheet.FillColumnDown(i, j, formula)

	reRenderSheet(sheet, limit, w, r)
}
