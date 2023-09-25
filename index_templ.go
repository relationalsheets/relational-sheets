// Code generated by templ@v0.2.334 DO NOT EDIT.

package main

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"fmt"
)

func sheetSelect(sheets map[int64]Sheet, sheetId int64) templ.Component {
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
		_, err = templBuffer.WriteString("<button hx-post=\"/sheet\" hx-prompt=\"Sheet name\" hx-target=\"#sheet-select\">")
		if err != nil {
			return err
		}
		var_2 := `New`
		_, err = templBuffer.WriteString(var_2)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button><div class=\"dropdown is-hoverable\"><div class=\"dropdown-trigger\"><button class=\"button\" aria-haspopup=\"true\" aria-controls=\"dropdown-menu\">")
		if err != nil {
			return err
		}
		if sheetId == 0 {
			_, err = templBuffer.WriteString("<span>")
			if err != nil {
				return err
			}
			var_3 := `Select Sheet`
			_, err = templBuffer.WriteString(var_3)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</span>")
			if err != nil {
				return err
			}
		} else {
			_, err = templBuffer.WriteString("<span>")
			if err != nil {
				return err
			}
			var_4 := `Current Sheet: `
			_, err = templBuffer.WriteString(var_4)
			if err != nil {
				return err
			}
			var var_5 string = sheets[sheetId].VisibleName()
			_, err = templBuffer.WriteString(templ.EscapeString(var_5))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</span>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</button></div><div class=\"dropdown-menu\"><div class=\"dropdown-content\">")
		if err != nil {
			return err
		}
		for _, sheet := range sheets {
			var var_6 = []any{"dropdown-item", templ.KV("is-active", sheet.Id == sheetId)}
			err = templ.RenderCSSItems(ctx, templBuffer, var_6...)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<a href=\"")
			if err != nil {
				return err
			}
			var var_7 templ.SafeURL = templ.SafeURL(fmt.Sprintf("/?sheet_id=%d", sheet.Id))
			_, err = templBuffer.WriteString(templ.EscapeString(string(var_7)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" class=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_6).String()))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\">")
			if err != nil {
				return err
			}
			var var_8 string = sheet.VisibleName()
			_, err = templBuffer.WriteString(templ.EscapeString(var_8))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(" ")
			if err != nil {
				return err
			}
			var_9 := `- `
			_, err = templBuffer.WriteString(var_9)
			if err != nil {
				return err
			}
			var var_10 string = fmt.Sprintf("%d", sheet.Id)
			_, err = templBuffer.WriteString(templ.EscapeString(var_10))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</a>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</div></div></div>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func index(sheets map[int64]Sheet, sheetId int64, tables []Table) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_11 := templ.GetChildren(ctx)
		if var_11 == nil {
			var_11 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<!doctype html><html><head><script src=\"https://unpkg.com/htmx.org@1.9.5\">")
		if err != nil {
			return err
		}
		var_12 := ``
		_, err = templBuffer.WriteString(var_12)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script><script src=\"https://unpkg.com/htmx.org/dist/ext/response-targets.js\">")
		if err != nil {
			return err
		}
		var_13 := ``
		_, err = templBuffer.WriteString(var_13)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script><script src=\"/static/index.js\"></script><link rel=\"stylesheet\" href=\"https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css\"><link rel=\"stylesheet\" href=\"/static/index.css\"></head><body><div id=\"toolbar\"><div id=\"sheet-select\">")
		if err != nil {
			return err
		}
		err = sheetSelect(sheets, sheetId).Render(ctx, templBuffer)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</div><div class=\"select table-select\"><select name=\"table_name\" hx-get=\"/table\" hx-target=\"#table\" hx-trigger=\"click,load\"><option value=\"\">")
		if err != nil {
			return err
		}
		var_14 := `Table`
		_, err = templBuffer.WriteString(var_14)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</option>")
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
			if table.TableName == sheets[sheetId].table.TableName {
				_, err = templBuffer.WriteString(" selected")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString(">")
			if err != nil {
				return err
			}
			var var_15 string = table.FullName()
			_, err = templBuffer.WriteString(templ.EscapeString(var_15))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</option>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</select></div></div><div class=\"scrollable\"><table id=\"table\" hx-trigger=\"click\" hx-target=\"#table\" hx-ext=\"response-targets\"></table></div></body></html>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
