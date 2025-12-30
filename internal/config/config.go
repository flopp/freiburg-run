package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Website struct {
		Url    string `json:"url"`
		Domain string `json:"domain"`
		Name   string `json:"name"`
	} `json:"website"`
	City struct {
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	} `json:"city"`
	Contact struct {
		FeedbackForm       string `json:"feedback_form"`
		ReportFormTemplate string `json:"report_form_template"`
		Instagram          string `json:"instagram"`
		Mastodon           string `json:"mastodon"`
		Whatsapp           string `json:"whatsapp"`
	} `json:"contact"`
	FooterLinks []struct {
		Name string `json:"name"`
		Url  string `json:"url"`
	} `json:"footer_links"`
	Google struct {
		ApiKey  string `json:"api_key"`
		SheetId string `json:"sheet_id"`
	} `json:"google"`
	Umami struct {
		WebsiteId string `json:"website_id"`
	} `json:"umami"`
}

func LoadConfig(filename string) (Config, error) {
	var config Config
	data, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.Website.Url == "" {
		return config, fmt.Errorf("website/url is empty in config file %s", filename)
	}
	if config.Website.Domain == "" {
		return config, fmt.Errorf("website/domain is empty in config file %s", filename)
	}

	return config, nil
}

func (c Config) DataSheetUrl() string {
	return "https://docs.google.com/spreadsheets/d/" + c.Google.SheetId
}
