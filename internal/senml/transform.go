package senml

import (
	"strconv"
	"strings"
	"time"

	"github.com/diwise/integration-havochvatten/internal/models"
	"github.com/diwise/senml"
)

const (
	// LwM2M Temperature Object ID
	TemperatureObjectID = "3303"

	// LwM2M Temperature Resource IDs
	SensorValueResourceID = "5700"

	// Unit for Celsius
	UnitCelsius = "Cel"

	TemperatureURN string = "urn:oma:lwm2m:ext:3303"
)

// TransformForecastToSenML transforms a water temperature forecast to SenML+JSON format
func TransformForecastToSenML(nutsCode string, forecast models.WaterTempForecast, baseTime time.Time) (senml.Pack, error) {
	if forecast.WaterTemp == "" {
		return nil, nil
	}

	// Parse water temperature from string to float64
	temp, err := strconv.ParseFloat(forecast.WaterTemp, 64)
	if err != nil {
		return nil, nil // Skip invalid temperatures
	}

	// Parse measHour and calculate timestamp
	hour, err := strconv.Atoi(forecast.MeasHour)
	if err != nil {
		hour = 0
	}

	// Create timestamp: base date + forecast hour
	forecastTime := time.Date(
		baseTime.Year(), baseTime.Month(), baseTime.Day(),
		hour, 0, 0, 0, time.UTC,
	)

	// Create base name: <nutsCode>/3303/ (lowercase)
	baseName := strings.ToLower(nutsCode) + "/" + TemperatureObjectID + "/"

	// Create timestamp as Unix time (int64)
	bt := float64(forecastTime.Unix())

	// Create SenML pack with temperature value
	pack := senml.Pack{
		{
			BaseName:    baseName,
			BaseTime:    bt,
			Name:        "0",
			StringValue: TemperatureURN,
		},
		{
			Name:  SensorValueResourceID,
			Value: &temp,
			Unit:  UnitCelsius,
		},
	}

	return pack, nil
}

// TransformBathingWaterForecastToSenML transforms all forecasts for a bathing water to SenML packs
func TransformBathingWaterForecastToSenML(nutsCode string, forecast models.BathingWaterForecast) ([]senml.Pack, error) {
	var packs []senml.Pack
	baseTime := time.Now().UTC()

	for _, wf := range forecast.WaterForecasts {
		pack, err := TransformForecastToSenML(nutsCode, wf, baseTime)
		if err != nil {
			return nil, err
		}
		if err := pack.Validate(); err != nil {
			return nil, err
		}
		if pack != nil {
			packs = append(packs, pack)
		}
	}

	return packs, nil
}
