package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type rawCurrentWeather struct {
	Temperature float64 `json:"temperature"`
	WeatherCode int     `json:"weathercode"`
	Time        string  `json:"time"`
}

type rawDaily struct {
	Time           []string  `json:"time"`
	TemperatureMin []float64 `json:"temperature_2m_min"`
	TemperatureMax []float64 `json:"temperature_2m_max"`
	Sunrise        []string  `json:"sunrise"`
	Sunset         []string  `json:"sunset"`
}

type rawWeatherResponse struct {
	Timezone       string            `json:"timezone"`
	CurrentWeather rawCurrentWeather `json:"current_weather"`
	Daily          rawDaily          `json:"daily"`
}

type Weather struct {
	Time           time.Time
	Temperature    float64
	TemperatureMin float64
	TemperatureMax float64
	Sunrise        time.Time
	Sunset         time.Time
	WeatherCode    int
}

// WeatherCodeGerman returns a German description of the weather based on the WMO Weather interpretation codes (WW).
func (w *Weather) WeatherCodeGerman() string {
	switch w.WeatherCode {
	case 0:
		return "Klarer Himmel"
	case 1:
		return "Hauptsächlich klar"
	case 2:
		return "Teilweise bewölkt"
	case 3:
		return "Überwiegend bewölkt"
	case 45:
		return "Nebel"
	case 48:
		return "Gefrierender Nebel"
	case 51:
		return "Leichter Nieselregen"
	case 53:
		return "Mäßiger Nieselregen"
	case 55:
		return "Starker Nieselregen"
	case 56:
		return "Leichter gefrierender Nieselregen"
	case 57:
		return "Starker gefrierender Nieselregen"
	case 61:
		return "Leichter Regen"
	case 63:
		return "Mäßiger Regen"
	case 65:
		return "Starker Regen"
	case 66:
		return "Leichter gefrierender Regen"
	case 67:
		return "Starker gefrierender Regen"
	case 71:
		return "Leichter Schneefall"
	case 73:
		return "Mäßiger Schneefall"
	case 75:
		return "Starker Schneefall"
	case 77:
		return "Schneegriesel"
	case 80:
		return "Leichter Regenschauer"
	case 81:
		return "Mäßiger Regenschauer"
	case 82:
		return "Gewaltiger Regenschauer"
	case 85:
		return "Leichter Schneeschauer"
	case 86:
		return "Starker Schneeschauer"
	case 95:
		return "Leichtes oder mäßiges Gewitter"
	case 96:
		return "Gewitter mit leichtem Hagel"
	case 99:
		return "Gewitter mit starkem Hagel"
	default:
		return "Unbekannt"
	}
}

func GetCurrentWeather(lat, lon float64) (*Weather, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.3f&longitude=%.3f&current_weather=true&daily=sunrise,sunset,temperature_2m_min,temperature_2m_max&timezone=auto", lat, lon)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var weatherResp rawWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %w", err)
	}

	// Transform rawWeatherResponse to WeatherResponse
	loc, err := time.LoadLocation(weatherResp.Timezone)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone: %w", err)
	}

	weatherTime, err := time.ParseInLocation("2006-01-02T15:04", weatherResp.CurrentWeather.Time, loc)
	if err != nil {
		return nil, fmt.Errorf("error parsing current weather time: %w", err)
	}

	sunriseTime, err := time.ParseInLocation("2006-01-02T15:04", weatherResp.Daily.Sunrise[0], loc)
	if err != nil {
		return nil, fmt.Errorf("error parsing sunrise time: %w", err)
	}

	sunsetTime, err := time.ParseInLocation("2006-01-02T15:04", weatherResp.Daily.Sunset[0], loc)
	if err != nil {
		return nil, fmt.Errorf("error parsing sunset time: %w", err)
	}

	weather := Weather{
		Time:           weatherTime,
		Temperature:    weatherResp.CurrentWeather.Temperature,
		TemperatureMin: weatherResp.Daily.TemperatureMin[0],
		TemperatureMax: weatherResp.Daily.TemperatureMax[0],
		Sunrise:        sunriseTime,
		Sunset:         sunsetTime,
		WeatherCode:    weatherResp.CurrentWeather.WeatherCode,
	}

	return &weather, nil
}
