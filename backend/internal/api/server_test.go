package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rodrigo/home-monitor/internal/api"
	"github.com/rodrigo/home-monitor/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "test-key"

// newTestServer spins up a server backed by a fresh in-memory SQLite DB.
// MaxOpenConns(1) is required: each ":memory:" connection is its own DB,
// so a pool of >1 would give each query a blank database.
func newTestServer(t *testing.T) (http.Handler, *storage.Queries) {
	t.Helper()
	db, err := storage.Open(":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	q := storage.New(db)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewServer(log, q, testKey), q
}

func seedSource(t *testing.T, q *storage.Queries, id, name string) {
	t.Helper()
	_, err := q.InsertSource(context.Background(), storage.InsertSourceParams{
		ID: id, Name: name, Kind: "sensor", Config: "{}",
	})
	require.NoError(t, err)
}

func seedReading(t *testing.T, q *storage.Queries, sourceID, metric string, value float64) {
	t.Helper()
	err := q.InsertReading(context.Background(), storage.InsertReadingParams{
		SourceID:   sourceID,
		MetricType: metric,
		Value:      value,
		Unit:       "°C",
		RecordedAt: time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
		Metadata:   sql.NullString{},
	})
	require.NoError(t, err)
}

func TestHandleIngest(t *testing.T) {
	validBody := `{
		"recorded_at": "2026-04-27T10:00:00Z",
		"samples": [
			{"metric_type": "temperature", "value": 22.4, "unit": "°C"},
			{"metric_type": "humidity",    "value": 58.1, "unit": "%"}
		]
	}`

	tests := []struct {
		name       string
		sourceID   string
		apiKey     string
		body       string
		seed       bool
		wantStatus int
		wantStored int
	}{
		{
			name:       "stores both samples",
			sourceID:   "esp32-sala",
			apiKey:     testKey,
			body:       validBody,
			seed:       true,
			wantStatus: http.StatusAccepted,
			wantStored: 2,
		},
		{
			name:       "wrong api key",
			sourceID:   "esp32-sala",
			apiKey:     "wrong",
			body:       validBody,
			seed:       true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "source not found",
			sourceID:   "nonexistent",
			apiKey:     testKey,
			body:       validBody,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty samples",
			sourceID:   "esp32-sala",
			apiKey:     testKey,
			body:       `{"recorded_at":"2026-04-27T10:00:00Z","samples":[]}`,
			seed:       true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			sourceID:   "esp32-sala",
			apiKey:     testKey,
			body:       `not json`,
			seed:       true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, q := newTestServer(t)
			if tc.seed {
				seedSource(t, q, "esp32-sala", "Sala")
			}

			req := httptest.NewRequest(http.MethodPost, "/api/sources/"+tc.sourceID+"/readings", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Api-Key", tc.apiKey)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStored > 0 {
				var got map[string]int
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tc.wantStored, got["stored"])
			}
		})
	}
}

func TestHandleListSources(t *testing.T) {
	tests := []struct {
		name      string
		seedIDs   []string
		wantCount int
	}{
		{
			name:      "empty DB returns empty array",
			wantCount: 0,
		},
		{
			name:      "returns all sources",
			seedIDs:   []string{"esp32-sala", "esp32-quarto"},
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, q := newTestServer(t)
			for _, id := range tc.seedIDs {
				seedSource(t, q, id, id)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			var got []map[string]any
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Len(t, got, tc.wantCount)
		})
	}
}

func TestHandleGetReadings(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		seed       bool
		wantStatus int
		wantCount  int
	}{
		{
			name:       "missing metric returns 400",
			url:        "/api/sources/esp32-sala/readings",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from date returns 400",
			url:        "/api/sources/esp32-sala/readings?metric=temperature&from=notadate",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "returns readings within range",
			url:        "/api/sources/esp32-sala/readings?metric=temperature&from=2026-04-27T00:00:00Z&to=2026-04-27T23:59:59Z",
			seed:       true,
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "empty when no data in range",
			url:        "/api/sources/esp32-sala/readings?metric=temperature&from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z",
			seed:       true,
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, q := newTestServer(t)
			if tc.seed {
				seedSource(t, q, "esp32-sala", "Sala")
				seedReading(t, q, "esp32-sala", "temperature", 22.4)
			}

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				var got []map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Len(t, got, tc.wantCount)
			}
		})
	}
}

func TestHandleLatestReadings(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		seed       bool
		wantStatus int
		wantCount  int
	}{
		{
			name:       "missing metric returns 400",
			url:        "/api/readings/latest",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no readings returns empty array",
			url:        "/api/readings/latest?metric=temperature",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "returns latest per source with source name",
			url:        "/api/readings/latest?metric=temperature",
			seed:       true,
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, q := newTestServer(t)
			if tc.seed {
				seedSource(t, q, "esp32-sala", "Sala")
				seedReading(t, q, "esp32-sala", "temperature", 22.4)
			}

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantStatus == http.StatusOK {
				var got []map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Len(t, got, tc.wantCount)
				if tc.wantCount > 0 {
					assert.Equal(t, "Sala", got[0]["source_name"])
					assert.Equal(t, "esp32-sala", got[0]["source_id"])
				}
			}
		})
	}
}
