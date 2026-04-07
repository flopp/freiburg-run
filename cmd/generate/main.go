package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/generator"
	"github.com/flopp/freiburg-run/internal/resources"
	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/flopp/go-googlesheetswrapper"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
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
	backup     string
	basePath   string
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	hashFile := flag.String("hashfile", ".hashes", "file storing file hashes (for sitemap)")
	checkLinks := flag.Bool("checklinks", false, "check links in the generated files")
	backup := flag.String("backup", "", "download and backup sheets data to the specified file")
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
		*backup,
		*basePath,
	}
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func createBackup(config utils.Config, outputFile string) error {
	fmt.Println("-- connecting to Google Drive service...")
	ctx := context.Background()
	service, err := drive.NewService(ctx, option.WithAPIKey(config.Google.ApiKey))
	if err != nil {
		return fmt.Errorf("unable to connect to Google Drive: %w", err)
	}

	fmt.Printf("-- requesting file %s...\n", config.Google.SheetId)
	response, err := service.Files.Export(config.Google.SheetId, "application/vnd.oasis.opendocument.spreadsheet").Download()
	if err != nil {
		return fmt.Errorf("unable to download file: %w", err)
	}
	defer response.Body.Close()

	fmt.Printf("-- saving to %s...\n", outputFile)
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("unable to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return fmt.Errorf("unable to write to output file: %w", err)
	}

	fmt.Println("-- done")
	return nil
}

func main() {
	options := parseCommandLine()

	config_data, err := utils.LoadConfig(options.configFile)
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
		return
	}

	if options.backup != "" {
		if err := createBackup(config_data, options.backup); err != nil {
			log.Fatalf("failed to backup data: %v", err)
		}
		return
	}

	// configuration
	out := utils.NewPath(options.outDir)
	basePath := options.basePath

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// try 3 times to fetch data with increasing timeouts (sometimes the google api is not available)
	eventsData, err := utils.Retry(3, 8*time.Second, func() (events.Data, error) {
		client, err := googlesheetswrapper.New(config_data.Google.ApiKey, config_data.Google.SheetId)
		if err != nil {
			return events.Data{}, fmt.Errorf("creating sheets client: %w", err)
		}
		return events.FetchData(config_data, today, client)
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
		basePath,
		now,
		resourceManager.JsFiles, resourceManager.CssFiles,
		resourceManager.UmamiScript,
		options.hashFile)
	if err := gen.Generate(eventsData); err != nil {
		log.Fatalf("failed to generate: %v", err)
	}
}
