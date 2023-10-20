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
)

func toolbar(sheet sheets.Sheet, sheets map[int]sheets.Sheet) templ.Component {
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
		_, err = templBuffer.WriteString("<div class=\"toolbar-group\"><button hx-get=\"/table\" hx-target=\"#table\" hx-trigger=\"click,load\"")
		if err != nil {
			return err
		}
		if sheet.TableFullName() != "" {
			_, err = templBuffer.WriteString(" disabled")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("><img src=\"/static/icons/cached_FILL0_wght400_GRAD0_opsz24.svg\"></button><button hx-get=\"/modal\" hx-target=\"#modal\" hx-swap=\"outerHTML\" name=\"sheet_id\" value=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(fmt.Sprintf("%d", sheet.Id)))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\">")
		if err != nil {
			return err
		}
		var_2 := `Edit`
		_, err = templBuffer.WriteString(var_2)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button><div class=\"dropdown is-hoverable\"><div class=\"dropdown-trigger\"><button aria-haspopup=\"true\" aria-controls=\"dropdown-menu\">")
		if err != nil {
			return err
		}
		var_3 := `Open`
		_, err = templBuffer.WriteString(var_3)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div><div class=\"dropdown-menu\"><div class=\"dropdown-content\"><a hx-get=\"/modal\" hx-target=\"#modal\" hx-swap=\"outerHTML\" class=\"dropdown-item\">")
		if err != nil {
			return err
		}
		var_4 := `+ New`
		_, err = templBuffer.WriteString(var_4)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a>")
		if err != nil {
			return err
		}
		for _, s := range sheets {
			var var_5 = []any{"dropdown-item", templ.KV("is-active", s.Id == sheet.Id)}
			err = templ.RenderCSSItems(ctx, templBuffer, var_5...)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<a href=\"")
			if err != nil {
				return err
			}
			var var_6 templ.SafeURL = templ.SafeURL(fmt.Sprintf("/?sheet_id=%d", s.Id))
			_, err = templBuffer.WriteString(templ.EscapeString(string(var_6)))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\" class=\"")
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_5).String()))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("\">")
			if err != nil {
				return err
			}
			var var_7 string = s.VisibleName()
			_, err = templBuffer.WriteString(templ.EscapeString(var_7))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(" ")
			if err != nil {
				return err
			}
			var_8 := `- `
			_, err = templBuffer.WriteString(var_8)
			if err != nil {
				return err
			}
			var var_9 string = fmt.Sprintf("%d", s.Id)
			_, err = templBuffer.WriteString(templ.EscapeString(var_9))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</a>")
			if err != nil {
				return err
			}
		}
		_, err = templBuffer.WriteString("</div></div></div><button disabled>")
		if err != nil {
			return err
		}
		var_10 := `Export`
		_, err = templBuffer.WriteString(var_10)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button><div class=\"dropdown is-hoverable\"><div class=\"dropdown-trigger\"><button aria-haspopup=\"true\" aria-controls=\"dropdown-menu\">")
		if err != nil {
			return err
		}
		var_11 := `Insert`
		_, err = templBuffer.WriteString(var_11)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div><div class=\"dropdown-menu\"><div class=\"dropdown-content\"><a class=\"dropdown-item\" onclick=\"htmx.removeClass(htmx.find(&#39;#new-row&#39;), &#39;hide&#39;)\n                            htmx.removeClass(htmx.find(&#39;#new-row-err-container&#39;), &#39;hide&#39;)\">")
		if err != nil {
			return err
		}
		var_12 := `Row`
		_, err = templBuffer.WriteString(var_12)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a><a hx-post=\"/add-column\" hx-target=\"#table\" hx-trigger=\"click\" class=\"dropdown-item\">")
		if err != nil {
			return err
		}
		var_13 := `Column`
		_, err = templBuffer.WriteString(var_13)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></div></div></div></div><div class=\"toolbar-group\"><input hx-post=\"/set-name\" name=\"name\" value=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(sheet.VisibleName()))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"></div><div class=\"toolbar-group\"><button disabled>")
		if err != nil {
			return err
		}
		var_14 := `Share`
		_, err = templBuffer.WriteString(var_14)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func index(sheet sheets.Sheet, sheets map[int]sheets.Sheet) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_15 := templ.GetChildren(ctx)
		if var_15 == nil {
			var_15 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<!doctype html><html><head><script src=\"https://unpkg.com/htmx.org@1.9.5\">")
		if err != nil {
			return err
		}
		var_16 := ``
		_, err = templBuffer.WriteString(var_16)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script><script src=\"https://unpkg.com/htmx.org/dist/ext/response-targets.js\">")
		if err != nil {
			return err
		}
		var_17 := ``
		_, err = templBuffer.WriteString(var_17)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script><script src=\"/static/index.js\"></script><link rel=\"stylesheet\" href=\"https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css\"><link rel=\"stylesheet\" href=\"/static/index.css\"></head><body><div id=\"toolbar\">")
		if err != nil {
			return err
		}
		err = toolbar(sheet, sheets).Render(ctx, templBuffer)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</div><div class=\"scrollable\"><table id=\"table\" hx-trigger=\"click\" hx-target=\"#table\" hx-ext=\"response-targets\"></table></div><div id=\"modal\"></div></body></html>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
