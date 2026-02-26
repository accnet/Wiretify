package web

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	Templates map[string]*template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.Templates[name]
	if !ok {
		return nil // Avoid crashing if missing template
	}
	// Luôn sử dụng layout.html làm khung bao bọc
	return tmpl.ExecuteTemplate(w, "layout.html", data)
}
