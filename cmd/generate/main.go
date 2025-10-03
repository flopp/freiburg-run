package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/flopp/freiburg-run/internal/config"
	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/generator"
	"github.com/flopp/freiburg-run/internal/resources"
	"github.com/flopp/freiburg-run/internal/utils"
)

const (
	usage = `USAGE: %s [OPTIONS...]

OPTIONS:
`
)

type CommandLineOptions struct {
	configFile string
	outDir     string
	hashFile   string
	checkLinks bool
	basePath   string
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	hashFile := flag.String("hashfile", ".hashes", "file storing file hashes (for sitemap)")
	checkLinks := flag.Bool("checklinks", false, "check links in the generated files")
	basePath := flag.String("basepath", "", "base path")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configFile == "" {
		panic("You have to specify a config file, e.g. -config myconfig.json")
	}

	return CommandLineOptions{
		*configFile,
		*outDir,
		*hashFile,
		*checkLinks,
		*basePath,
	}
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func main() {
	options := parseCommandLine()

	config_data, err := config.LoadConfig(options.configFile)
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
		return
	}

	// configuration
	out := utils.NewPath(options.outDir)
	baseUrl := utils.Url("https://freiburg.run")
	basePath := options.basePath
	umamiId := "6609164f-5e79-4041-b1ed-f37da10a84d2"
	feedbackFormUrl := "https://docs.google.com/forms/d/e/1FAIpQLScJlPFYKgT5WxDaH9FDJzha5hQ2cBsqALLrVjGp7bgB-ssubA/viewform?usp=sf_link"

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// try 3 times to fetch data with increasing timeouts (sometimes the google api is not available)
	eventsData, err := utils.Retry(3, 8*time.Second, func() (events.Data, error) {
		return events.FetchData(config_data, today)
	})
	if err != nil {
		log.Fatalf("failed to fetch data: %v", err)
		return
	}

	if options.checkLinks {
		eventsData.CheckLinks()
		return
	}

	resourceManager := resources.NewResourceManager(".", string(out))
	resourceManager.CopyExternalAssets()
	if resourceManager.Error != nil {
		log.Fatalf("failed to copy external assets: %v", resourceManager.Error)
	}
	resourceManager.CopyStaticAssets()
	if resourceManager.Error != nil {
		log.Fatalf("failed to copy static assets: %v", resourceManager.Error)
	}

	gen := generator.NewGenerator(
		config_data,
		out,
		baseUrl, basePath,
		now,
		resourceManager.JsFiles, resourceManager.CssFiles,
		resourceManager.UmamiScript, umamiId,
		feedbackFormUrl,
		options.hashFile)
	if err := gen.Generate(eventsData); err != nil {
		log.Fatalf("failed to generate: %v", err)
	}
}
