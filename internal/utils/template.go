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

var templates = make(map[string]*template.Template)

func loadTemplate(name string) (*template.Template, error) {
	if t, ok := templates[name]; ok {
		return t, nil
	}

	// collect all *.html files in templates/parts folder
	parts, err := filepath.Glob("templates/parts/*.html")
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 1+len(parts))
	files = append(files, fmt.Sprintf("templates/%s.html", name))
	files = append(files, parts...)
	t, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	templates[name] = t
	return t, nil
}

func executeTemplateToBuffer(templateName string, data any) (*bytes.Buffer, error) {
	// load template
	templ, err := loadTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// render to buffer
	var buffer bytes.Buffer
	err = templ.Execute(&buffer, data)
	if err != nil {
		return nil, err
	}

	return &buffer, nil
}

func prepareOutputFile(fileName string) (*os.File, error) {
	// create output folder + file
	outDir := filepath.Dir(fileName)
	err := MakeDir(outDir)
	if err != nil {
		return nil, err
	}

	// create output file
	out, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func ExecuteTemplate(templateName string, fileName string, data any) {
	buffer, err := executeTemplateToBuffer(templateName, data)
	Check(err)

	out, err := prepareOutputFile(fileName)
	Check(err)
	defer out.Close()

	// minify buffer to output file
	m := minify.New()
	m.AddFunc("text/css", html.Minify)
	m.Add("text/html", &html.Minifier{KeepQuotes: true})
	err = m.Minify("text/html", out, buffer)
	Check(err)
}

func ExecuteTemplateNoMinify(templateName string, fileName string, data any) {
	buffer, err := executeTemplateToBuffer(templateName, data)
	Check(err)

	out, err := prepareOutputFile(fileName)
	Check(err)
	defer out.Close()

	// write buffer to output file
	_, err = out.Write(buffer.Bytes())
	Check(err)
}
