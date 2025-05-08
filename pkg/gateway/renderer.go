package gateway

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		tmplPath := filepath.Join("pkg/admin/templates", name)
		var err error
		tmpl, err = template.ParseFiles(tmplPath)
		if err != nil {
			return err
		}

		t.templates[name] = tmpl
	}

	return tmpl.Execute(w, data)
}
