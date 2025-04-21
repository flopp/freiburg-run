package resources

import (
	"fmt"
	"path/filepath"

	"github.com/flopp/freiburg-run/internal/utils"
)

type ResourceManager struct {
	Out         string
	JsFiles     []string
	CssFiles    []string
	UmamiScript string
}

func NewResourceManager(out string) *ResourceManager {
	return &ResourceManager{
		Out:      out,
		JsFiles:  make([]string, 0),
		CssFiles: make([]string, 0),
	}
}

func (r *ResourceManager) MustRel(path string) string {
	rel, err := filepath.Rel(r.Out, path)
	utils.Check(err)
	return rel
}

func (r *ResourceManager) DownloadHash(url, targetFile string) string {
	target := filepath.Join(r.Out, targetFile)
	res := utils.MustDownloadHash(url, target)
	return r.MustRel(res)
}

func (r *ResourceManager) CopyHash(sourcePath, targetFile string) string {
	res := utils.MustCopyHash(sourcePath, filepath.Join(r.Out, targetFile))
	return r.MustRel(res)
}

func (r *ResourceManager) CopyExternalAssets() {

	// renovate: datasource=npm depName=bulma
	bulmaVersion := "1.0.3"
	// renovate: datasource=npm depName=leaflet
	leafletVersion := "1.9.4"
	// renovate: datasource=npm depName=leaflet-gesture-handling
	leafletGestureHandlingVersion := "1.2.2"

	leafletLegendVersion := "v1.0.0"

	// URLs
	bulmaUrl := utils.Url(fmt.Sprintf("https://cdnjs.cloudflare.com/ajax/libs/bulma/%s", bulmaVersion))
	leafletUrl := utils.Url(fmt.Sprintf("https://cdnjs.cloudflare.com/ajax/libs/leaflet/%s", leafletVersion))
	leafletGestureHandlingUrl := utils.Url(fmt.Sprintf("https://raw.githubusercontent.com/elmarquis/Leaflet.GestureHandling/refs/tags/v%s", leafletGestureHandlingVersion))
	leafletLegendUrl := utils.Url(fmt.Sprintf("https://raw.githubusercontent.com/ptma/Leaflet.Legend/%s", leafletLegendVersion))

	// JS files
	r.JsFiles = append(r.JsFiles, r.DownloadHash(leafletUrl.Join("leaflet.min.js"), "leaflet-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.DownloadHash(leafletLegendUrl.Join("src/leaflet.legend.js"), "leaflet-legend-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.DownloadHash(leafletGestureHandlingUrl.Join("dist/leaflet-gesture-handling.min.js"), "leaflet-gesture-handling-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.CopyHash("static/parkrun-track.js", "parkrun-track-HASH.js"))
	r.JsFiles = append(r.JsFiles, r.CopyHash("static/main.js", "main-HASH.js"))

	r.UmamiScript = r.DownloadHash("https://cloud.umami.is/script.js", "umami-HASH.js")

	// CSS files
	r.CssFiles = append(r.CssFiles, r.DownloadHash(bulmaUrl.Join("css/bulma.min.css"), "bulma-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.DownloadHash(leafletUrl.Join("leaflet.min.css"), "leaflet-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.DownloadHash(leafletLegendUrl.Join("src/leaflet.legend.css"), "leaflet-legend-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.DownloadHash(leafletGestureHandlingUrl.Join("dist/leaflet-gesture-handling.min.css"), "leaflet-gesture-handling-HASH.css"))
	r.CssFiles = append(r.CssFiles, r.CopyHash("static/style.css", "style-HASH.css"))

	// Images
	utils.MustDownload(leafletUrl.Join("images/marker-icon.png"), filepath.Join(r.Out, "images/marker-icon.png"))
	utils.MustDownload(leafletUrl.Join("images/marker-icon-2x.png"), filepath.Join(r.Out, "images/marker-icon-2x.png"))
	utils.MustDownload(leafletUrl.Join("images/marker-shadow.png"), filepath.Join(r.Out, "images/marker-shadow.png"))
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

	for _, file := range staticFiles {
		utils.MustCopy(file.Source, filepath.Join(r.Out, file.Destination))
	}
}
