package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

var Config Configuration

type Configuration struct {
	LogLevel int `json:"logLevel"`
}

func LoadConfig(path string) {
	var c = Configuration{}

	var cf []byte
	var err error
	if path != "" {
		cf, err = os.ReadFile(path)
	} else {
		cf, err = os.ReadFile("config.json")
	}
	if err != nil {
		slog.Info("failed to open config at path provided, using default config instead")
	}

	err = json.Unmarshal(cf, &c)
	if err != nil {
		slog.Info("failed to read configuration, using default config instead...", err)
	}

	Config = c
	return
}
