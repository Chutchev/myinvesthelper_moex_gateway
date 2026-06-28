package cbr

import "time"

type Direction string

const (
	DirectionUp      Direction = "up"
	DirectionFlat    Direction = "flat"
	DirectionDown    Direction = "down"
	DirectionUnknown Direction = "unknown"
)

type RatePoint struct {
	EffectiveDate string  `json:"effective_date"`
	Rate          float64 `json:"rate"`
}

type RateForecast struct {
	Year        int     `json:"year"`
	Low         float64 `json:"low"`
	Midpoint    float64 `json:"midpoint"`
	High        float64 `json:"high"`
	PublishedAt string  `json:"published_at"`
	SourceURL   string  `json:"source_url"`
}

type RateSnapshot struct {
	CurrentRate *float64      `json:"current_rate"`
	History     []RatePoint   `json:"history"`
	Direction   Direction     `json:"direction"`
	Forecast    *RateForecast `json:"forecast"`
	FetchedAt   time.Time     `json:"fetched_at"`
	Warnings    []string      `json:"warnings"`
}
