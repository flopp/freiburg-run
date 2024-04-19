package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

func loadTemplate(name string) *template.Template {
	t, err := template.ParseFiles(fmt.Sprintf("templates/%s.html", name), "templates/header.html", "templates/footer.html", "templates/tail.html", "templates/card.html", "templates/support-modal.html")
	Check(err)
	return t
}

func ExecuteTemplate(templateName string, fileName string, data any) {
	// render to buffer
	var buffer bytes.Buffer
	err := loadTemplate(templateName).Execute(&buffer, data)
	Check(err)

	// create output folder + file
	outDir := filepath.Dir(fileName)
	MustMakeDir(outDir)
	out, err := os.Create(fileName)
	Check(err)
	defer out.Close()

	// minify buffer to output file
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	err = m.Minify("text/html", out, &buffer)
	Check(err)
}
