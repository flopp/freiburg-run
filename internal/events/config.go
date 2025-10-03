package events

import (
	"encoding/json"
	"os"
)

type Config struct {
	BaseUrl string
	Google  struct {
		ApiKey  string
		SheetId string
	}
	UmamiId         string
	FeedbackFormUrl string
}

// example of a config.json file:
// {
// 	"base_url": "https://example.com",
// 	"google": {
// 		"api_key": "your_api_key",
// 		"sheet_id": "your_sheet_id"
// 	},
// 	"umami_id": "your_umami_id"
// }

func LoadConfigJson(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
