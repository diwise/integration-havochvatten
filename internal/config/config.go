package config

import (
	"os"
	"strings"
)

const (
	DefaultBaseURL = "https://gw.havochvatten.se/external-public/bathing-waters/v2"
)

const (
	DefaultIotAgentURL = "http://iot-agent/api/v0/messages/lwm2m"
)

// Config holds the application configuration
type Config struct {
	// NutsCodes is a list of NUTS codes to fetch bathing waters for
	NutsCodes []string

	// BaseURL is the base URL for the Havs- och vattenmyndigheten API
	BaseURL string

	// IotAgentURL is the URL to POST SenML messages to
	IotAgentURL string

	// IncludeFutureForecasts controls whether to include forecasts for future hours
	// Default is false (only include current/past forecasts)
	IncludeFutureForecasts bool
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		BaseURL:                DefaultBaseURL,
		IotAgentURL:            DefaultIotAgentURL,
		IncludeFutureForecasts: false,
	}

	// Load NUTS codes from environment variable
	nutsCodesEnv := os.Getenv("NUTS_CODES")
	if nutsCodesEnv != "" {
		codes := strings.Split(nutsCodesEnv, ",")
		for _, code := range codes {
			trimmed := strings.TrimSpace(code)
			if trimmed != "" {
				cfg.NutsCodes = append(cfg.NutsCodes, trimmed)
			}
		}
	}

	// Allow overriding the base URL
	if baseURL := os.Getenv("HOV_BADPLATSEN_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	// Allow overriding the LwM2M endpoint URL
	if lwm2mURL := os.Getenv("LWM2M_ENDPOINT_URL"); lwm2mURL != "" {
		cfg.IotAgentURL = lwm2mURL
	}

	// Include future forecasts (default: false)
	if includeFuture := os.Getenv("INCLUDE_FUTURE_FORECASTS"); strings.ToLower(includeFuture) == "true" {
		cfg.IncludeFutureForecasts = true
	}

	return cfg, nil
}
