package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	usage = `USAGE: %s [OPTIONS...]

	Download Google Sheet as an ODS file.

OPTIONS:
`
)

type CommandLineOptions struct {
	configFile string
	outputFile string
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "Config file")
	outputFile := flag.String("output", "", "Output file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configFile == "" || *outputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	return CommandLineOptions{
		*configFile,
		*outputFile,
	}
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func UnmarshalConfigData(path string) (ConfigData, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return ConfigData{}, fmt.Errorf("read config file '%s': %w", path, err)
	}
	var data ConfigData
	err = json.Unmarshal(buf, &data)
	if err != nil {
		return ConfigData{}, fmt.Errorf("unmarshall data: %w", err)
	}

	return data, nil
}

func main() {
	options := parseCommandLine()

	config, err := UnmarshalConfigData(options.configFile)
	if err != nil {
		fmt.Printf("Unable to read config file: %v\n", err)
		return
	}

	fmt.Println("-- connecting to Google Drive service...")
	ctx := context.Background()
	service, err := drive.NewService(ctx, option.WithAPIKey(config.ApiKey))
	if err != nil {
		fmt.Printf("Unable to connect to Google Drive: %v\n", err)
		return
	}

	fmt.Printf("-- requesting file %s...\n", config.SheetId)
	response, err := service.Files.Export(config.SheetId, "application/vnd.oasis.opendocument.spreadsheet").Download()
	if err != nil {
		fmt.Printf("Unable to download file: %v\n", err)
		return
	}
	defer response.Body.Close()

	fmt.Printf("-- saving to %s...\n", options.outputFile)
	file, err := os.Create(options.outputFile)
	if err != nil {
		fmt.Printf("Unable to create output file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Printf("Unable to write to output file: %v\n", err)
		return
	}

	fmt.Println("-- done")
}
