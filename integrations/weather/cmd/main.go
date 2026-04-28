package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// --- config ---

type config struct {
	apiURL    string
	apiKey    string
	sourceID  string
	latitude  float64
	longitude float64
	interval  time.Duration
}

func loadConfig() (config, error) {
	require := func(key string) (string, error) {
		v := os.Getenv(key)
		if v == "" {
			return "", fmt.Errorf("required env var %s is not set", key)
		}
		return v, nil
	}

	apiURL, err := require("HOME_MONITOR_API_URL")
	if err != nil {
		return config{}, err
	}
	apiKey, err := require("HOME_MONITOR_API_KEY")
	if err != nil {
		return config{}, err
	}
	sourceID, err := require("SOURCE_ID")
	if err != nil {
		return config{}, err
	}

	latStr, err := require("LATITUDE")
	if err != nil {
		return config{}, err
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return config{}, fmt.Errorf("LATITUDE: %w", err)
	}

	lonStr, err := require("LONGITUDE")
	if err != nil {
		return config{}, err
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return config{}, fmt.Errorf("LONGITUDE: %w", err)
	}

	interval := 10 * time.Minute
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		interval, err = time.ParseDuration(v)
		if err != nil {
			return config{}, fmt.Errorf("POLL_INTERVAL: %w", err)
		}
	}

	return config{
		apiURL:    apiURL,
		apiKey:    apiKey,
		sourceID:  sourceID,
		latitude:  lat,
		longitude: lon,
		interval:  interval,
	}, nil
}

// --- Open-Meteo types ---

type openMeteoResponse struct {
	Current struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		Humidity    float64 `json:"relative_humidity_2m"`
	} `json:"current"`
}

// --- home-monitor ingest types ---

type ingestRequest struct {
	RecordedAt time.Time        `json:"recorded_at"`
	Samples    []sample         `json:"samples"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
}

type sample struct {
	MetricType string  `json:"metric_type"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
}

// --- fetcher ---

func fetchWeather(ctx context.Context, client *http.Client, lat, lon float64) (openMeteoResponse, error) {
	endpoint := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%g&longitude=%g&current=temperature_2m,relative_humidity_2m&timezone=UTC",
		lat, lon,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return openMeteoResponse{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return openMeteoResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return openMeteoResponse{}, fmt.Errorf("open-meteo: unexpected status %d", resp.StatusCode)
	}
	var w openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return openMeteoResponse{}, fmt.Errorf("open-meteo: decode: %w", err)
	}
	return w, nil
}

// --- poster ---

func postReadings(ctx context.Context, client *http.Client, cfg config, req ingestRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/api/sources/%s/readings", cfg.apiURL, url.PathEscape(cfg.sourceID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", cfg.apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("home-monitor: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// --- poll ---

func poll(ctx context.Context, log *slog.Logger, client *http.Client, cfg config) {
	weather, err := fetchWeather(ctx, client, cfg.latitude, cfg.longitude)
	if err != nil {
		log.Error("fetch weather", "err", err)
		return
	}

	// Open-Meteo returns "2006-01-02T15:04" in UTC when timezone=UTC
	recordedAt, err := time.Parse("2006-01-02T15:04", weather.Current.Time)
	if err != nil {
		log.Warn("could not parse recorded_at, using now", "raw", weather.Current.Time)
		recordedAt = time.Now()
	}

	req := ingestRequest{
		RecordedAt: recordedAt.UTC(),
		Samples: []sample{
			{MetricType: "temperature", Value: weather.Current.Temperature, Unit: "°C"},
			{MetricType: "humidity", Value: weather.Current.Humidity, Unit: "%"},
		},
		Metadata: map[string]any{
			"provider":  "open-meteo",
			"latitude":  cfg.latitude,
			"longitude": cfg.longitude,
		},
	}

	if err := postReadings(ctx, client, cfg, req); err != nil {
		log.Error("post readings", "err", err)
		return
	}

	log.Info("readings posted",
		"temperature_c", weather.Current.Temperature,
		"humidity_pct", weather.Current.Humidity,
		"recorded_at", recordedAt.UTC().Format(time.RFC3339),
	)
}

// --- main ---

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := loadConfig()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	log.Info("weather provider starting",
		"source_id", cfg.sourceID,
		"latitude", cfg.latitude,
		"longitude", cfg.longitude,
		"interval", cfg.interval,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Poll once immediately on startup, then on each tick.
	poll(ctx, log, client, cfg)

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			poll(ctx, log, client, cfg)
		case <-ctx.Done():
			log.Info("shutting down")
			return
		}
	}
}
