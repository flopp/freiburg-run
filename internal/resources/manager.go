package resources

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/flopp/freiburg-run/internal/utils"
)

type ResourceManager struct {
	SourceDir   string
	TargetDir   string
	JsFiles     []string
	CssFiles    []string
	UmamiScript string
	Error       error
}

func NewResourceManager(sourceDir string, out string) *ResourceManager {
	return &ResourceManager{
		SourceDir: sourceDir,
		TargetDir: out,
		JsFiles:   make([]string, 0),
		CssFiles:  make([]string, 0),
	}
}

func (r *ResourceManager) DownloadErr(url, targetFile string) {
	target := filepath.Join(r.TargetDir, targetFile)
	err := utils.Download(url, target)
	if err != nil {
		r.Error = err
	}
}

func (r *ResourceManager) CopyHashErr(sourcePath, targetFile string) string {
	res, err := utils.CopyHash(sourcePath, filepath.Join(r.TargetDir, targetFile))
	if err != nil {
		r.Error = err
		return ""
	}

	rel, err := filepath.Rel(r.TargetDir, res)
	if err != nil {
		r.Error = err
		return ""
	}

	return rel
}

func patchLeafletGestureHandling(TargetDir, fileName string) string {
	targetFile := utils.Path(TargetDir).Join(fileName)
	contents, err := os.ReadFile(targetFile)
	if err != nil {
		log.Printf("while trying to patch file '%s': %v", targetFile, err)
		return fileName
	}

	// Patch German language translation, see https://github.com/elmarquis/Leaflet.GestureHandling/pull/97/files
	//
	// scrollMac:"⌘"
	// ->
	// scrollMac:"Verwende \u2318 + Scrollen zum Zoomen der Karte"
	contents = []byte(strings.ReplaceAll(string(contents), "scrollMac:\"⌘\"", "scrollMac:\"Verwende \\u2318 + Scrollen zum Zoomen der Karte\""))

	err = os.WriteFile(targetFile, contents, 0644)
	if err != nil {
		log.Printf("while trying to patch file '%s': %v", targetFile, err)
		return fileName
	}

	return fileName
}

func (r *ResourceManager) CopyExternalAssets() {
	source := utils.Path(r.SourceDir)
	static := utils.Path(source.Join("static"))
	vendor := utils.Path(source.Join("external-files"))

	// JS files
	r.JsFiles = append(r.JsFiles, r.CopyHashErr(vendor.Join("leaflet", "leaflet.js"), "leaflet-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.CopyHashErr(vendor.Join("leaflet-legend", "leaflet-legend.js"), "leaflet-legend-HASH.js"))
	r.JsFiles = append(r.JsFiles, patchLeafletGestureHandling(r.TargetDir, r.CopyHashErr(vendor.Join("leaflet-gesture-handling", "leaflet-gesture-handling.js"), "leaflet-gesture-handling-HASH.js")))
	r.JsFiles = append(r.JsFiles, r.CopyHashErr(static.Join("parkrun-track.js"), "parkrun-track-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.CopyHashErr(static.Join("main.js"), "main-HASH.js"))

	r.UmamiScript = r.CopyHashErr(vendor.Join("umami", "umami.js"), "umami-HASH.js")

	// CSS files
	r.CssFiles = append(r.CssFiles, r.CopyHashErr(vendor.Join("bulma", "bulma.css"), "bulma-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.CopyHashErr(vendor.Join("leaflet", "leaflet.css"), "leaflet-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.CopyHashErr(vendor.Join("leaflet-legend", "leaflet-legend.css"), "leaflet-legend-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.CopyHashErr(vendor.Join("leaflet-gesture-handling", "leaflet-gesture-handling.css"), "leaflet-gesture-handling-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.CopyHashErr(static.Join("style.css"), "style-HASH.css"))

	// Images
	r.CopyHashErr(vendor.Join("leaflet", "marker-icon.png"), filepath.Join(r.TargetDir, "images/marker-icon.png"))
	r.CopyHashErr(vendor.Join("leaflet", "marker-icon-2x.png"), filepath.Join(r.TargetDir, "images/marker-icon-2x.png"))
	r.CopyHashErr(vendor.Join("leaflet", "marker-shadow.png"), filepath.Join(r.TargetDir, "images/marker-shadow.png"))
}

func (r *ResourceManager) CopyStaticAssets() {
	// Copy static files using a slice of pairs to handle duplicate source files
	staticFiles := []struct {
		Source      string
		Destination string
	}{
		{"static/robots.txt", "robots.txt"},
		{"static/manifest.json", "manifest.json"},
		{"static/5vkf9hdnfkay895vyx33zdvesnyaphgv.txt", "5vkf9hdnfkay895vyx33zdvesnyaphgv.txt"},
		{"static/512.png", "favicon.png"},
		{"static/favicon.ico", "favicon.ico"},
		{"static/180.png", "apple-touch-icon.png"},
		{"static/192.png", "android-chrome-192x192.png"},
		{"static/512.png", "android-chrome-512x512.png"},
		{"static/freiburg-run.svg", "images/freiburg-run.svg"},
		{"static/freiburg-run-new.svg", "images/freiburg-run-new.svg"},
		{"static/freiburg-run-new-blue.svg", "images/freiburg-run-new-blue.svg"},
		{"static/512.png", "images/512.png"},
		{"static/parkrun.png", "images/parkrun.png"},
		{"static/marker-grey-icon.png", "images/marker-grey-icon.png"},
		{"static/marker-grey-icon-2x.png", "images/marker-grey-icon-2x.png"},
		{"static/marker-green-icon.png", "images/marker-green-icon.png"},
		{"static/marker-green-icon-2x.png", "images/marker-green-icon-2x.png"},
		{"static/marker-red-icon.png", "images/marker-red-icon.png"},
		{"static/marker-red-icon-2x.png", "images/marker-red-icon-2x.png"},
		{"static/circle-small.png", "images/circle-small.png"},
		{"static/circle-big.png", "images/circle-big.png"},
		{"static/freiburg-run-flyer.pdf", "freiburg-run-flyer.pdf"},
	}

	source := utils.Path(r.SourceDir)
	target := utils.Path(r.TargetDir)

	for _, file := range staticFiles {
		if err := utils.Copy(source.Join(file.Source), target.Join(file.Destination)); err != nil {
			r.Error = err
			return
		}
	}
}
