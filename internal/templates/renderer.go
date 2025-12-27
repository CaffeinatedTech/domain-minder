package templates

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(dir string) *Renderer {
	funcs := template.FuncMap{
		"formatDate":  formatDate,
		"daysLeft":    daysLeft,
		"colorClass":  colorClass,
		"percent":     percentRemaining,
		"boolChecked": boolChecked,
		"nullString":  nullString,
		"eq":          eq,
		"notEq":       notEq,
	}

	tmpl := template.New("templates").Funcs(funcs)

	var templateFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".html" {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	for _, file := range templateFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		_, err = tmpl.Parse(string(content))
		if err != nil {
			panic("Error parsing " + file + ": " + err.Error())
		}
	}

	return &Renderer{templates: tmpl}
}

func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}
	return t.templates.ExecuteTemplate(w, name, data)
}

func formatDate(t interface{}) string {
	if t == nil {
		return ""
	}
	if tt, ok := t.(time.Time); ok {
		return tt.Format("2006-01-02")
	}
	return ""
}

func daysLeft(t interface{}) int {
	return 0
}

func colorClass(daysLeft int) string {
	switch {
	case daysLeft < 0:
		return "progress-red"
	case daysLeft < 7:
		return "progress-critical"
	case daysLeft < 14:
		return "progress-red"
	case daysLeft < 30:
		return "progress-orange"
	case daysLeft < 60:
		return "progress-yellow"
	default:
		return "progress-green"
	}
}

func percentRemaining(daysLeft, maxDays int) float64 {
	if daysLeft <= 0 {
		return 100
	}
	if daysLeft > maxDays {
		return 5
	}
	return float64(daysLeft) / float64(maxDays) * 100
}

func boolChecked(b bool) string {
	if b {
		return "checked"
	}
	return ""
}

func nullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func eq(a, b interface{}) bool {
	return a == b
}

func notEq(a, b interface{}) bool {
	return a != b
}
