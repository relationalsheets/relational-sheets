package main

import (
	"acb/db-interface/sheets"
	"errors"
	"github.com/a-h/templ"
	"log"
	"net/http"
	"slices"
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

func getPKs(r *http.Request) map[string]string {
	pkValues := map[string]string{}
	sheets.Check(r.ParseForm())
	for key, v := range r.Form {
		pkColName, found := strings.CutPrefix(key, "pk-")
		if found {
			pkValues[pkColName] = v[0]
		}
	}
	return pkValues
}

func handleAddCol(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	sheet.AddColumn("")
	reRenderSheet(sheet, w, r)
}

func handleRenameCol(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	colIndex, err := strconv.Atoi(r.FormValue("col_index"))
	if err != nil {
		writeError(w, err.Error())
		return
	}
	sheet.RenameCol(colIndex, r.FormValue("col_name"))
	w.WriteHeader(http.StatusNoContent)
}

func handleSetExtraCell(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(r.FormValue("i"))
	sheets.Check(err)
	j, err := strconv.Atoi(r.FormValue("j"))
	sheets.Check(err)
	formula := r.FormValue("formula")

	cell := sheet.SetCell(i, j, formula)

	handler := templ.Handler(extraCell(i, j, cell))
	handler.ServeHTTP(w, r)
}

func reRenderSheet(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	if sheet.TableFullName() == "" {
		writeError(w, "No table name provided")
		return
	}
	tableNames, cols := sheet.OrderedTablesAndCols(nil)
	cells := sheet.LoadRows(100, 0)
	numCols := 0
	for _, tcols := range cols {
		numCols += len(tcols)
	}
	component := sheetTable(sheet, tableNames, cols, cells, numCols)
	handler := templ.Handler(component)
	handler.ServeHTTP(w, r)
}

func handleSetTable(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		tableName := r.FormValue("table_name")
		if tableName == "" {
			writeError(w, "No table name provided")
			return
		}
		sheet.SetTable(tableName)
	}

	reRenderSheet(sheet, w, r)
}

func handleNewRow(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	tableNames, cols := sheet.OrderedTablesAndCols(nil)
	tableName := r.FormValue("table_name")
	tableIndex := slices.Index(tableNames, tableName)
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
	if len(pks) > 0 {
		if tableIndex == -1 {
			writeError(w, "Unrecognized table")
			return
		}
		// TODO: cache
		cells := sheet.LoadRows(100, 0)[tableIndex]
		for rowIndex, _ = range cells[0] {
			match := true
			for colIndex, col := range cols[tableIndex] {
				row[colIndex] = cells[colIndex][rowIndex]
				pkValue, ok := pks[col.Name]
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

	component := newRow(tableNames, tableIndex, cols, numCols, row, rowIndex)
	templ.Handler(component).ServeHTTP(w, r)
}

func handleAddRow(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	values := make(map[string]map[string]string)
	for key, value := range r.Form {
		name, found := strings.CutPrefix(key, "column-")
		if found {
			parts := strings.Split(name, " ")
			if len(parts) != 2 {
				writeError(w, "Unexpected field name: "+name)
			}
			submap, ok := values[parts[0]]
			if !ok {
				submap = make(map[string]string)
				values[parts[0]] = submap
			}
			submap[parts[1]] = value[0]
		}
	}

	err := sheet.InsertMultipleRows(values)
	if err != nil {
		writeError(w, err.Error())
		return
	}

	reRenderSheet(sheet, w, r)
}

func handleIndex(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	if sheet.Id != 0 {
		sheet.LoadSheet()
	}
	templ.Handler(index(sheet, sheets.SheetMap)).ServeHTTP(w, r)
}

func handleSetColPref(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
	tableName := r.FormValue("table_name")
	colName := r.FormValue("col_name")
	if colName == "" {
		writeError(w, "Missing required key: col_name")
	}
	pref := sheet.PrefsMap[tableName+"."+colName]
	pref.TableName = tableName
	pref.ColumnName = colName
	pref.Hide = r.FormValue("hide") == "true"
	pref.SortOn = r.FormValue("sorton") == "true"
	pref.Ascending = r.FormValue("ascending") == "true"
	log.Printf("saving pref: %v", pref)
	sheet.SetPref(pref)
	sheet.LoadRows(100, 0)

	reRenderSheet(sheet, w, r)
}

func handleModal(sheet sheets.Sheet, w http.ResponseWriter, r *http.Request) {
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
	templ.Handler(modal(sheet, sheets.TableMap, addJoin)).ServeHTTP(w, r)
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
		map[string]map[string]string{tableName: getPKs(r)})
	cell := tableCell(tableName, col, row, sheets.Cell{value, value != ""}, err)
	templ.Handler(cell).ServeHTTP(w, r)
}
