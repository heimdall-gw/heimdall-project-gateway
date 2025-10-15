package main

import (
	"fmt"
	"os"
	"log"

	"gopkg.in/yaml.v3"
)

// Defines the structure for a single provider in config.yaml
type ProviderConfig struct {
	Name         string `yaml:"name"`
	HTTPURL     string `yaml:"http_url"`
	WsURL        string `yaml:"ws_url"`
	APIKeyEnv    string `yaml:"api_key_env"`
	HostnameEnv  string `yaml:"hostname_env"`
	AuthTokenEnv string `yaml:"auth_token_env"`
}

// Defines the top-level structure of config.yaml
type Config struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// Reads the YAML file and injects API keys from the environment
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	// Important: This loop reads the actual secret keys from the environment
	// (loaded by docker-compose from your .env file) and places them in the config
	for i := range config.Providers {
		switch config.Providers[i].Name {
		case "QuickNode":
			hostname := os.Getenv(config.Providers[i].HostnameEnv)
			authToken := os.Getenv(config.Providers[i].AuthTokenEnv)
			if hostname == "" || authToken == "" {
				log.Printf("WARNING: QuickNode environment variables (%s, %s) not fully set.", config.Providers[i].HostnameEnv, config.Providers[i].AuthTokenEnv)
			}
			config.Providers[i].HTTPURL = fmt.Sprintf("https://%s/%s", hostname, authToken)
			config.Providers[i].WsURL = fmt.Sprintf("wss://%s/%s", hostname, authToken)

		default:
			apiKey := os.Getenv(config.Providers[i].APIKeyEnv)
			if apiKey == "" {
				log.Printf("WARNING: Environment variable %s not set for provider %s", config.Providers[i].APIKeyEnv, config.Providers[i].Name)
			}
			config.Providers[i].HTTPURL += apiKey
			config.Providers[i].WsURL += apiKey
		}
	}

	return &config, nil
}
