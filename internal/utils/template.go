package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

var templates = make(map[string]*template.Template)

func loadTemplate(conf Config, name string, basePath string) (*template.Template, error) {
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
	t, err := template.New(name + ".html").Funcs(template.FuncMap{
		"BasePath": func(p string) string {
			res := basePath
			if !strings.HasPrefix(p, "/") {
				res += "/"
			}
			res += p
			if strings.HasPrefix(basePath, "/Users/") && strings.HasSuffix(p, "/") {
				res += "index.html"
			}
			return res
		},
		"FullPath": func(p string) string {
			res := conf.Website.Url
			if !strings.HasPrefix(p, "/") {
				res += "/"
			}
			res += p
			if strings.HasPrefix(basePath, "/Users/") && strings.HasSuffix(p, "/") {
				res += "index.html"
			}
			return res
		},
		"ReportFormUrl": func(name string, url string) string {
			formUrl := conf.Contact.ReportFormTemplate
			formUrl = strings.ReplaceAll(formUrl, "NAME", template.URLQueryEscaper(name))
			formUrl = strings.ReplaceAll(formUrl, "URL", template.HTMLEscaper(url))
			return formUrl
		},
		"Config": func() Config {
			return conf
		},
		"NotificationMessagesJSON": func() (string, error) {
			today := time.Now()
			today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

			type message struct {
				Id      int    `json:"id"`
				Start   string `json:"start"`
				End     string `json:"end"`
				Content string `json:"content"`
				Class   string `json:"class"`
			}

			filtered := make([]message, 0, len(conf.Notification.Messages))
			for _, m := range conf.Notification.Messages {
				// filter out messages whose end date is in the past
				if m.End != "" {
					end, err := ParseDate(m.End)
					if err == nil && end.Before(today) {
						continue
					}
				}
				filtered = append(filtered, message{m.Id, m.Start, m.End, m.Content, m.Class})
			}

			// sort by id ascending
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].Id < filtered[j].Id
			})

			data, err := json.Marshal(filtered)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	}).ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	templates[name] = t
	return t, nil
}

func executeTemplateToBuffer(conf Config, templateName string, basePath string, data any) (*bytes.Buffer, error) {
	// load template
	templ, err := loadTemplate(conf, templateName, basePath)
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

func ExecuteTemplate(conf Config, templateName string, fileName string, basePath string, data any) error {
	buffer, err := executeTemplateToBuffer(conf, templateName, basePath, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	out, err := prepareOutputFile(fileName)
	if err != nil {
		return fmt.Errorf("prepare output file: %w", err)
	}
	defer out.Close()

	// minify buffer to output file
	m := minify.New()
	m.AddFunc("text/css", html.Minify)
	m.Add("text/html", &html.Minifier{KeepQuotes: true})
	err = m.Minify("text/html", out, buffer)
	if err != nil {
		return fmt.Errorf("minifying html output: %w", err)
	}

	return nil
}

func ExecuteTemplateNoMinify(conf Config, templateName string, fileName string, basePath string, data any) error {
	buffer, err := executeTemplateToBuffer(conf, templateName, basePath, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	out, err := prepareOutputFile(fileName)
	if err != nil {
		return fmt.Errorf("prepare output file: %w", err)
	}
	defer out.Close()

	// write buffer to output file
	_, err = out.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write buffer to output file: %w", err)
	}

	return nil
}
