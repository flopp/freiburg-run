package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Website struct {
		Name string `json:"name"`
	} `json:"website"`
	City struct {
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	} `json:"city"`
	Contact struct {
		FeedbackForm string `json:"feedback_form"`
		Instagram    string `json:"instagram"`
		Mastodon     string `json:"mastodon"`
		Whatsapp     string `json:"whatsapp"`
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
	return config, err
}

func (c Config) DataSheetUrl() string {
	return "https://docs.google.com/spreadsheets/d/" + c.Google.SheetId
}
