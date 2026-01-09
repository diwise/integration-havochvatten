package models

// WatersAndAdvisoriesResponse is the response from the /bathing-waters endpoint
type WatersAndAdvisoriesResponse struct {
	WatersAndAdvisories []BathingWaterAndAdvisories `json:"watersAndAdvisories"`
}

// BathingWaterAndAdvisories represents a bathing water with its advisories
type BathingWaterAndAdvisories struct {
	BathingWater         BathingWater           `json:"bathingWater"`
	AdviceAgainstBathing []AdviceAgainstBathing `json:"adviceAgainstBathing"`
}

// BathingWater represents a bathing water location
type BathingWater struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	EuType       bool          `json:"euType"`
	Description  string        `json:"description,omitempty"`
	Municipality *Municipality `json:"municipality,omitempty"`
}

// Municipality represents municipality information
type Municipality struct {
	Name string `json:"name"`
}

// AdviceAgainstBathing represents advice against bathing
type AdviceAgainstBathing struct {
	StartsAt    string `json:"startsAt"`
	Description string `json:"description"`
	TypeID      int    `json:"typeId"`
	TypeIDText  string `json:"typeIdText"`
}

// ForecastsResponse is the response from the /forecasts endpoint
type ForecastsResponse struct {
	Forecasts []BathingWaterForecast `json:"forecasts"`
}

// BathingWaterForecast represents forecast data for a bathing water
type BathingWaterForecast struct {
	BathingWaterID string              `json:"bathingWaterId"`
	WaterForecasts []WaterTempForecast `json:"waterForecasts"`
}

// WaterTempForecast represents a water temperature forecast from Copernicus
type WaterTempForecast struct {
	WaterTemp string `json:"waterTemp"`
	MeasHour  string `json:"measHour"`
}
