package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

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
	addedFile  string
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	hashFile := flag.String("hashfile", ".hashes", "file storing file hashes (for sitemap)")
	addedFile := flag.String("addedfile", ".added", "file storing event addition dates")

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
		*addedFile,
	}
}

func IsNew(s string, now time.Time) bool {
	days := 14

	d, err := utils.ParseDate(s)
	if err == nil {
		return d.AddDate(0, 0, days).After(now)
	}

	return false
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func updateAddedDates(events []*events.Event, added *utils.Added, eventType string, timestamp string, now time.Time) {
	for _, event := range events {
		fromFile := added.GetAdded(eventType, event.Slug())
		if fromFile == "" {
			if event.Added == "" {
				event.Added = timestamp
			}
			added.SetAdded(eventType, event.Slug(), event.Added)
		} else {
			if event.Added == "" {
				event.Added = fromFile
			}
		}
		event.New = IsNew(event.Added, now)
	}
}

func main() {
	options := parseCommandLine()

	config_data, err := events.LoadSheetsConfig(options.configFile)
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
		return
	}

	// configuration
	out := utils.NewPath(options.outDir)
	baseUrl := utils.Url("https://freiburg.run")
	sheetUrl := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", config_data.SheetId)
	umamiId := "6609164f-5e79-4041-b1ed-f37da10a84d2"

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timestamp := now.Format("2006-01-02")

	// try 3 times to fetch data with increasing timeouts (sometimes the google api is not available)
	eventsData, err := utils.Retry(3, 8*time.Second, func() (events.Data, error) {
		return events.FetchData(config_data, today)
	})
	if err != nil {
		log.Fatalf("failed to fetch data: %v", err)
		return
	}

	if options.addedFile != "" {
		added, err := utils.ReadAdded(options.addedFile)
		if err != nil {
			log.Printf("failed to parse added file: '%s' - %v", options.addedFile, err)
		}

		updateAddedDates(eventsData.Events, added, "event", timestamp, now)
		updateAddedDates(eventsData.EventsOld, added, "event", timestamp, now)
		updateAddedDates(eventsData.Groups, added, "group", timestamp, now)
		updateAddedDates(eventsData.Shops, added, "shop", timestamp, now)

		if err = added.Write(options.addedFile); err != nil {
			log.Printf("failed to write added file: '%s' - %v", options.addedFile, err)
		}
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
		out, baseUrl,
		now,
		resourceManager.JsFiles, resourceManager.CssFiles,
		resourceManager.UmamiScript, umamiId,
		sheetUrl,
		options.hashFile)
	if err := gen.Generate(eventsData); err != nil {
		log.Fatalf("failed to generate: %v", err)
	}
}
