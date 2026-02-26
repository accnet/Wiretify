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
	// Kiểm tra xem template có chứa định nghĩa "layout.html" không.
	// Nếu có thì dùng ExecuteTemplate, nếu không thì Execute trực tiếp (ví dụ login.html)
	if tmpl.Lookup("layout.html") != nil {
		return tmpl.ExecuteTemplate(w, "layout.html", data)
	}
	return tmpl.Execute(w, data)
}
