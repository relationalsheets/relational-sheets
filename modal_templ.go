// Code generated by templ@v0.2.334 DO NOT EDIT.

package main

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"acb/db-interface/sheets"
	"strconv"
)

func fkeySelect(table sheets.Table, selected int) templ.Component {
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
		_, err = templBuffer.WriteString("<div class=\"select fkey-select\"><select name=\"fkey\" hx-post=\"/modal\"><option value=\"\"></option>")
		if err != nil {
			return err
		}
		for oid, fkey := range table.FkeysFrom {
			_, err = templBuffer.WriteString("<option value=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(strconv.Itoa(oid)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\"")
			if err != nil {
				return err
			}
			if oid == selected {
				_, err = templBuffer.WriteString(" selected")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString(">")
			if err != nil {
				return err
			}
			var var_2 string = fkey.ToString(true)
			_, err = templBuffer.WriteString(templ.EscapeString(var_2))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</option>")
			if err != nil {
				return err
			}
		}
		for oid, fkey := range table.FkeysTo {
			_, err = templBuffer.WriteString("<option value=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(strconv.Itoa(oid)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\"")
			if err != nil {
				return err
			}
			if oid == selected {
				_, err = templBuffer.WriteString(" selected")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString(">")
			if err != nil {
				return err
			}
			var var_3 string = fkey.ToString(false)
			_, err = templBuffer.WriteString(templ.EscapeString(var_3))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</option>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</select></div>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func modal(sheet sheets.Sheet, tables []sheets.Table, addJoin bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_4 := templ.GetChildren(ctx)
		if var_4 == nil {
			var_4 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<div id=\"modal\" class=\"modal is-active\" hx-target=\"body\" hx-swap=\"beforeend\" hx-include=\"select\"><div class=\"modal-content box\"><div id=\"table-fkey-config\"><label>")
		if err != nil {
			return err
		}
		var_5 := `Tables`
		_, err = templBuffer.WriteString(var_5)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</label><div class=\"dropdown-list\"><div class=\"select table-select\"><select name=\"table_name\" hx-post=\"/modal\"><option value=\"\"></option>")
		if err != nil {
			return err
		}
		for _, table := range tables {
			_, err = templBuffer.WriteString("<option value=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(table.FullName()))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\"")
			if err != nil {
				return err
			}
			if table.FullName() == sheet.TableFullName() {
				_, err = templBuffer.WriteString(" selected")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString(">")
			if err != nil {
				return err
			}
			var var_6 string = table.FullName()
			_, err = templBuffer.WriteString(templ.EscapeString(var_6))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</option>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</select></div>")
		if err != nil {
			return err
		}
		for oid, _ := range sheet.Joins {
			err = fkeySelect(sheet.Table, oid).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
		}
		if addJoin {
			err = fkeySelect(sheet.Table, 0).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("<button hx-post=\"/modal\" name=\"add_join\" class=\"button is-primary\">")
		if err != nil {
			return err
		}
		var_7 := `+ Join`
		_, err = templBuffer.WriteString(var_7)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div></div><div class=\"flex full-width mt center\"><button hx-post=\"/sheet\" class=\"button is-primary\">")
		if err != nil {
			return err
		}
		var_8 := `Ok`
		_, err = templBuffer.WriteString(var_8)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div></div><div class=\"modal-close\"></div></div>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
