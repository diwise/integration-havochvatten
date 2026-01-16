package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/diwise/integration-havochvatten/internal/models"
)

// Client is an HTTP client for the Havs- och vattenmyndigheten API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new API client
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetBathingWaters fetches all active bathing waters
func (c *Client) GetBathingWaters(ctx context.Context) ([]models.BathingWaterAndAdvisories, error) {
	url := fmt.Sprintf("%s/bathing-waters", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response models.WatersAndAdvisoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.WatersAndAdvisories, nil
}

// GetForecastByBathingWaterID fetches water temperature forecast for a specific bathing water
func (c *Client) GetForecastByBathingWaterID(ctx context.Context, bathingWaterID string) (*models.BathingWaterForecast, error) {
	url := fmt.Sprintf("%s/forecasts?bathingWaterId=%s", c.baseURL, bathingWaterID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if len(b) == 0 || bytes.Equal(b, []byte("[]")) {
		return nil, nil
	}

	r := io.NopCloser(bytes.NewReader(b))
	defer r.Close()

	var response models.ForecastsResponse
	if err := json.NewDecoder(r).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(response.Forecasts) == 0 {
		return nil, nil
	}

	return &response.Forecasts[0], nil
}
