package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/diwise/integration-havochvatten/internal/client"
	"github.com/diwise/integration-havochvatten/internal/config"
	"github.com/diwise/integration-havochvatten/internal/models"
	"github.com/diwise/integration-havochvatten/internal/senml"
	senmlpkg "github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

const (
	serviceName string = "integration-havochvatten"
)

func main() {
	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion, "json")
	defer cleanup()

	testMode := flag.Bool("test", false, "Run in test mode with a local mock IoT Agent server")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err.Error())
		os.Exit(1)
	}

	if len(cfg.NutsCodes) == 0 {
		logger.Error("no NUTS codes configured")
		time.Sleep(200 * time.Millisecond)
		os.Exit(1)
	}

	// If test mode, start a local mock server
	if *testMode {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error("test server: failed to read body", "err", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			var data any
			if err := json.Unmarshal(body, &data); err != nil {
				logger.Error("test server: failed to unmarshal JSON", "err", err.Error())
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			indented, _ := json.MarshalIndent(data, "", "  ")
			logger.Info("test server received SenML pack",
				"path", r.URL.Path,
				"contentType", r.Header.Get("Content-Type"),
				"body", string(indented),
			)

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		cfg.IotAgentURL = server.URL + "/api/v0/messages/lwm2m"
		logger.Info("test mode enabled", "mockServerUrl", cfg.IotAgentURL)
	}

	logger.Info("starting integration-havochvatten",
		"bathingWaterIds", cfg.NutsCodes,
		"includeFutureForecasts", cfg.IncludeFutureForecasts,
		"iotAgentUrl", cfg.IotAgentURL,
	)

	httpClient := client.New(cfg.BaseURL)
	postClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	currentHour := time.Now().Hour()
	baseTime := time.Now().UTC()

	var sentCount int

	for _, bathingWaterID := range cfg.NutsCodes {
		// Fetch forecast for this specific bathing water
		forecast, err := httpClient.GetForecastByBathingWaterID(ctx, bathingWaterID)
		if err != nil {
			logger.Error("failed to fetch forecast",
				"bathingWaterId", bathingWaterID,
				"err", err.Error(),
			)
			continue
		}

		if forecast == nil {
			logger.Warn("no forecast found", "bathingWaterId", bathingWaterID)
			continue
		}

		// Filter forecasts based on time
		validForecasts := filterForecastsByTime(forecast.WaterForecasts, currentHour, cfg.IncludeFutureForecasts)

		if len(validForecasts) == 0 {
			logger.Warn("no valid forecasts after filtering", "bathingWaterId", bathingWaterID)
			continue
		}

		logger.Info("found forecast",
			"bathingWaterId", forecast.BathingWaterID,
			"forecastCount", len(validForecasts),
			"firstTemp", validForecasts[0].WaterTemp,
			"firstHour", validForecasts[0].MeasHour,
		)

		// Transform to SenML and POST each pack individually
		for _, wf := range validForecasts {
			pack, err := senml.TransformForecastToSenML(bathingWaterID, wf, baseTime)
			if err != nil {
				logger.Error("failed to transform forecast",
					"bathingWaterId", bathingWaterID,
					"err", err.Error(),
				)
				continue
			}
			if pack == nil {
				continue
			}

			// POST this single pack to IoT Agent
			if err := postSenMLPack(ctx, postClient, cfg.IotAgentURL, pack); err != nil {
				logger.Error("failed to post SenML pack",
					"bathingWaterId", bathingWaterID,
					"measHour", wf.MeasHour,
					"err", err.Error(),
				)
				continue
			}
			sentCount++
		}

		time.Sleep(200 * time.Millisecond)
	}

	logger.Info("integration completed successfully", "sentCount", sentCount)
	time.Sleep(200 * time.Millisecond)
}

// filterForecastsByTime filters forecasts based on current hour
// If includeFuture is false, only forecasts for current or past hours are included
func filterForecastsByTime(forecasts []models.WaterTempForecast, currentHour int, includeFuture bool) []models.WaterTempForecast {
	if includeFuture {
		return forecasts
	}

	return slices.Collect(filterByHour(forecasts, currentHour))
}

// filterByHour returns an iterator that yields forecasts for hours <= currentHour
func filterByHour(forecasts []models.WaterTempForecast, currentHour int) iter.Seq[models.WaterTempForecast] {
	return func(yield func(models.WaterTempForecast) bool) {
		for _, f := range forecasts {
			hour, err := strconv.Atoi(f.MeasHour)
			if err != nil {
				continue
			}
			if hour <= currentHour {
				if !yield(f) {
					return
				}
			}
		}
	}
}

var tracer = otel.Tracer("integration-havochvatten")

// postSenMLPack sends a SenML pack to the IoT Agent via HTTP POST
func postSenMLPack(ctx context.Context, client *http.Client, url string, pack senmlpkg.Pack) error {
	var err error

	ctx, span := tracer.Start(ctx, "post-lwm2m-senml-pack")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	data, err := json.Marshal(pack)
	if err != nil {
		err = fmt.Errorf("marshaling SenML pack: %w", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		err = fmt.Errorf("creating request: %w", err)
		return err
	}

	req.Header.Set("Content-Type", senmlpkg.MediaTypeSenmlJSON)

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("executing request: %w", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return err
	}

	return nil
}
