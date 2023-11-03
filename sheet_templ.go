// Code generated by templ@v0.2.334 DO NOT EDIT.

package main

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"acb/db-interface/sheets"
	"fmt"
	"strconv"
)

func sortIcon(ascending bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_1 := templ.GetChildren(ctx)
		if var_1 == nil {
			var_1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		if ascending {
			_, err = templBuffer.WriteString("<img src=\"/static/icons/arrow_upward_FILL0_wght400_GRAD0_opsz24.svg\">")
			if err != nil {
				return err
			}
		} else {
			_, err = templBuffer.WriteString("<img src=\"/static/icons/arrow_downward_FILL0_wght400_GRAD0_opsz24.svg\">")
			if err != nil {
				return err
			}
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func colHeader(tableName string, col sheets.Column, pref sheets.Pref) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_2 := templ.GetChildren(ctx)
		if var_2 == nil {
			var_2 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var var_3 = []any{templ.KV("is-pkey", col.IsPrimaryKey)}
		err = templ.RenderCSSItems(ctx, templBuffer, var_3...)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("<th class=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_3).String()))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-post=\"/set-column-prefs\" hx-vals=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("js:{table_name:\"%s\",col_name:\"%s\",hide:shiftPressed,sorton:\"%t\",ascending:\"%t\"}",
			tableName, col.Name, !pref.SortOn || !pref.Ascending, pref.SortOn && !pref.Ascending)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"><div class=\"col-header\"><span>")
		if err != nil {
			return err
		}
		var var_4 string = col.Name
		_, err = templBuffer.WriteString(templ.EscapeString(var_4))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</span>")
		if err != nil {
			return err
		}
		if pref.SortOn {
			err = sortIcon(pref.Ascending).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</div></th>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func extraColHeader(i int, name string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_5 := templ.GetChildren(ctx)
		if var_5 == nil {
			var_5 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<th hx-post=\"/delete-column\" hx-vals=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("{\"col_index\":%d}", i)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-trigger=\"click[shiftKey]\"><div class=\"flex\"><input name=\"col_name\" hx-vals=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("{\"col_index\":%d}", i)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" value=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(name))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-post=\"/rename-column\" hx-swap=\"none\"></div></th>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func tableCell(tableName string, col sheets.Column, row int, cell sheets.Cell, err error) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_6 := templ.GetChildren(ctx)
		if var_6 == nil {
			var_6 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		if col.IsPrimaryKey {
			_, err = templBuffer.WriteString("<div hx-get=\"/new-row\" hx-trigger=\"click\" hx-vals=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("{\"table_name\":\"%s\"}", tableName)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" hx-include=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("[name=sheet_id],tr[data-row=\"%d\"] [data-table=\"%s\"][name^=pk-]", row, tableName)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" hx-target=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("tr[data-row=\"%d\"]", row)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" hx-swap=\"afterend\">")
			if err != nil {
				return err
			}
			var var_7 string = cell.Value
			_, err = templBuffer.WriteString(templ.EscapeString(var_7))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</div>")
			if err != nil {
				return err
			}
		} else {
			var var_8 = []any{templ.KV("is-danger", err != nil)}
			err = templ.RenderCSSItems(ctx, templBuffer, var_8...)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<input name=\"value\" hx-post=\"/set-cell\" hx-target=\"this\" hx-swap=\"outerHTML\" hx-vals=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("{\"table_name\":\"%s\",\"col_name\":\"%s\",\"row\":%d}", tableName, col.Name, row)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" hx-include=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("[name=sheet_id],tr[data-row=\"%d\"] [data-table=\"%s\"][name^=pk-]", row, tableName)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" value=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(cell.Value))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" size=\"1\" class=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_8).String()))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\">")
			if err != nil {
				return err
			}
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func extraCell(i, j int, cell sheets.SheetCell) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_9 := templ.GetChildren(ctx)
		if var_9 == nil {
			var_9 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var var_10 = []any{templ.KV("is-null", !cell.NotNull)}
		err = templ.RenderCSSItems(ctx, templBuffer, var_10...)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("<td class=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_10).String()))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"><form class=\"flex extra-cell\" onsubmit=\"event.preventDefault()\" hx-trigger=\"click[ctrlKey]\" hx-vals=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("{\"i\":%d,\"j\":%d}", i, j)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-post=\"/fill-column-down\"><input name=\"formula\" class=\"extra-cell-formula hide\" value=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(cell.Formula))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-trigger=\"change\" hx-post=\"/set-extra-cell\" hx-target=\"closest td\" hx-swap=\"outerHTML\" size=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(strconv.Itoa(max(len(cell.Formula), 1))))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"><span class=\"extra-cell-value\">")
		if err != nil {
			return err
		}
		var var_11 string = cell.Value
		_, err = templBuffer.WriteString(templ.EscapeString(var_11))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</span></form></td>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func newRow(tableNames []string, tableName string, cols [][]sheets.Column, numCols int, cells []sheets.Cell, rowIndex int) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_12 := templ.GetChildren(ctx)
		if var_12 == nil {
			var_12 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<tr id=\"new-row\">")
		if err != nil {
			return err
		}
		for i, tcols := range cols {
			for j, col := range tcols {
				if tableNames[i] == tableName && len(cells) > 0 {
					var var_13 = []any{templ.KV("is-null", len(cells) > 0 && !cells[j].NotNull)}
					err = templ.RenderCSSItems(ctx, templBuffer, var_13...)
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("<td style=\"border-bottom: none\" class=\"")
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_13).String()))
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("\"><span>")
					if err != nil {
						return err
					}
					var var_14 string = cells[j].Value
					_, err = templBuffer.WriteString(templ.EscapeString(var_14))
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("</span></td>")
					if err != nil {
						return err
					}
				} else {
					_, err = templBuffer.WriteString("<td style=\"border-bottom: none\"><input name=\"")
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString(templ.EscapeString("column-" + tableNames[i] + " " + col.Name))
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("\"></td>")
					if err != nil {
						return err
					}
				}
			}
		}
		_, err = templBuffer.WriteString("</tr><tr id=\"new-row-err-container\"><td colspan=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("%d", numCols)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" style=\"border-top: none\"><div class=\"flex center\"><button hx-post=\"/add-row\" hx-include=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("[name=sheet_id],#new-row,tr[data-row=\"%d\"] [data-table=\"%s\"][name^=pk-]", rowIndex, tableName)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\" hx-include=\"#new-row\" hx-target-400=\"#new-row-err\" class=\"button is-light\">")
		if err != nil {
			return err
		}
		var_15 := `Add`
		_, err = templBuffer.WriteString(var_15)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button><span id=\"new-row-err\" class=\"has-text-danger\"></span></div></td></tr>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func sheetTable(sheet sheets.Sheet, cols [][]sheets.Column, cells [][][]sheets.Cell, numCols int) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_16 := templ.GetChildren(ctx)
		if var_16 == nil {
			var_16 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<thead><tr>")
		if err != nil {
			return err
		}
		for i, tableName := range sheet.TableNames {
			if len(cols[i]) > 0 {
				_, err = templBuffer.WriteString("<th colspan=\"")
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString(templ.EscapeString(strconv.Itoa(len(cols[i]))))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\">")
				if err != nil {
					return err
				}
				var var_17 string = tableName
				_, err = templBuffer.WriteString(templ.EscapeString(var_17))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</th>")
				if err != nil {
					return err
				}
			}
		}
		_, err = templBuffer.WriteString("</tr><tr id=\"header-row\">")
		if err != nil {
			return err
		}
		for i, tcols := range cols {
			for _, col := range tcols {
				err = colHeader(sheet.TableNames[i], col, sheet.PrefsMap[sheet.TableNames[i]+"."+col.Name]).Render(ctx, templBuffer)
				if err != nil {
					return err
				}
			}
		}
		for i, col := range sheet.ExtraCols {
			err = extraColHeader(i, col.Name).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</tr></thead><tbody>")
		if err != nil {
			return err
		}
		for j := 0; j < sheet.RowCount; j++ {
			_, err = templBuffer.WriteString("<tr class=\"body-row\" data-row=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(strconv.Itoa(j)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\">")
			if err != nil {
				return err
			}
			for i, tableCols := range cells {
				for k, cells := range tableCols {
					var var_18 = []any{templ.KV("is-null", !cells[j].NotNull)}
					err = templ.RenderCSSItems(ctx, templBuffer, var_18...)
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("<td class=\"")
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_18).String()))
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("\"><span class=\"width-control\">")
					if err != nil {
						return err
					}
					var var_19 string = cells[j].Value
					_, err = templBuffer.WriteString(templ.EscapeString(var_19))
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("</span>")
					if err != nil {
						return err
					}
					err = tableCell(sheet.TableNames[i], cols[i][k], j, cells[j], nil).Render(ctx, templBuffer)
					if err != nil {
						return err
					}
					if cols[i][k].IsPrimaryKey && cells[j].NotNull {
						_, err = templBuffer.WriteString("<input name=\"")
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString(templ.EscapeString("pk-" + sheet.TableNames[i] + " " + cols[i][k].Name))
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString("\" data-table=\"")
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString(templ.EscapeString(sheet.TableNames[i]))
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString("\" value=\"")
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString(templ.EscapeString(cells[j].Value))
						if err != nil {
							return err
						}
						_, err = templBuffer.WriteString("\" type=\"hidden\">")
						if err != nil {
							return err
						}
					}
					_, err = templBuffer.WriteString("</td>")
					if err != nil {
						return err
					}
				}
			}
			for i, extraCol := range sheet.ExtraCols {
				err = extraCell(i, j, extraCol.Cells[j]).Render(ctx, templBuffer)
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString("</tr>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</tbody>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
