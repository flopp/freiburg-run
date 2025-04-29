package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/flopp/freiburg-run/internal/utils"
)

const (
	usage = `USAGE: %s [OPTIONS...]

	Update external vendor assets.

OPTIONS:
`
)

type CommandLineOptions struct {
	vendorDir string
}

func parseCommandLine() CommandLineOptions {
	vendorDir := flag.String("dir", "", "Vendor dir")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *vendorDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	return CommandLineOptions{
		*vendorDir,
	}
}

func MustDownload(url, targetFile string) {
	fmt.Printf("Downloading %s to %s\n", url, targetFile)
	err := utils.Download(url, targetFile)
	if err != nil {
		panic(fmt.Sprintf("failed to download %s: %v", url, err))
	}
}

func main() {
	options := parseCommandLine()

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

	vendorDir := utils.Path(options.vendorDir)

	// download bulma
	MustDownload(bulmaUrl.Join("css/bulma.min.css"), vendorDir.Join("bulma", "bulma.min.css"))

	// download leaflet
	MustDownload(leafletUrl.Join("leaflet.min.css"), vendorDir.Join("leaflet", "leaflet.css"))
	MustDownload(leafletUrl.Join("leaflet.min.js"), vendorDir.Join("leaflet", "leaflet.js"))
	MustDownload(leafletUrl.Join("images/marker-icon.png"), vendorDir.Join("leaflet", "marker-icon.png"))
	MustDownload(leafletUrl.Join("images/marker-icon-2x.png"), vendorDir.Join("leaflet", "marker-icon-2x.png"))
	MustDownload(leafletUrl.Join("images/marker-shadow.png"), vendorDir.Join("leaflet", "marker-shadow.png"))

	// download leaflet-gesture-handling
	MustDownload(leafletGestureHandlingUrl.Join("dist/leaflet-gesture-handling.min.js"), vendorDir.Join("leaflet-gesture-handling", "leaflet-gesture-handling.js"))
	MustDownload(leafletGestureHandlingUrl.Join("dist/leaflet-gesture-handling.min.css"), vendorDir.Join("leaflet-gesture-handling", "leaflet-gesture-handling.css"))

	// download leaflet-legend
	MustDownload(leafletLegendUrl.Join("src/leaflet.legend.css"), vendorDir.Join("leaflet-legend", "leaflet-legend.css"))
	MustDownload(leafletLegendUrl.Join("src/leaflet.legend.js"), vendorDir.Join("leaflet-legend", "leaflet-legend.js"))

	// download umami
	MustDownload("https://cloud.umami.is/script.js", vendorDir.Join("umami", "umami.js"))
}
