package utils

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

func loadTemplate(name string) *template.Template {
	t, err := template.ParseFiles(fmt.Sprintf("templates/%s.html", name), "templates/header.html", "templates/footer.html", "templates/tail.html", "templates/card.html")
	Check(err)
	return t
}

func ExecuteTemplate(templateName string, fileName string, data any) {
	outDir := filepath.Dir(fileName)
	MustMakeDir(outDir)
	out, err := os.Create(fileName)
	Check(err)
	defer out.Close()
	err = loadTemplate(templateName).Execute(out, data)
	Check(err)
}
