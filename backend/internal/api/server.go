package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/rodrigo/home-monitor/internal/storage"
)

type Server struct {
	log    *slog.Logger
	store  Store
	apiKey string
	mux    *http.ServeMux
}

func NewServer(log *slog.Logger, store Store, apiKey string) http.Handler {
	s := &Server{log: log, store: store, apiKey: apiKey, mux: http.NewServeMux()}
	s.routes()
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/sources/{sourceId}/readings", s.handleIngest)
	s.mux.HandleFunc("GET /api/sources", s.handleListSources)
	s.mux.HandleFunc("GET /api/sources/{sourceId}/readings", s.handleGetReadings)
	s.mux.HandleFunc("GET /api/readings/latest", s.handleLatestReadings)
}

// --- request / response types ---

type ingestRequest struct {
	RecordedAt time.Time       `json:"recorded_at"`
	Samples    []sample        `json:"samples"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type sample struct {
	MetricType string  `json:"metric_type"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
}

type sourceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

type readingResponse struct {
	ID         int64     `json:"id"`
	SourceID   string    `json:"source_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
	ReceivedAt time.Time `json:"received_at"`
}

type latestReadingResponse struct {
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// --- handlers ---

// POST /api/sources/{sourceId}/readings
// Requires X-Api-Key header matching the source's key.
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("sourceId")

	if r.Header.Get("X-Api-Key") != s.apiKey {
		http.Error(w, "invalid API key", http.StatusUnauthorized)
		return
	}

	_, err := s.store.GetSourceByID(r.Context(), sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error("get source by id", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Samples) == 0 {
		http.Error(w, "samples must not be empty", http.StatusBadRequest)
		return
	}

	var metadata sql.NullString
	if len(req.Metadata) > 0 && string(req.Metadata) != "null" {
		metadata = sql.NullString{String: string(req.Metadata), Valid: true}
	}

	for _, sm := range req.Samples {
		if err := s.store.InsertReading(r.Context(), storage.InsertReadingParams{
			SourceID:   sourceID,
			MetricType: sm.MetricType,
			Value:      sm.Value,
			Unit:       sm.Unit,
			RecordedAt: req.RecordedAt,
			Metadata:   metadata,
		}); err != nil {
			s.log.Error("insert reading", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	s.log.Info("readings stored", "source_id", sourceID, "count", len(req.Samples))
	writeJSON(w, http.StatusAccepted, map[string]int{"stored": len(req.Samples)})
}

// GET /api/sources
func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.store.ListSources(r.Context())
	if err != nil {
		s.log.Error("list sources", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]sourceResponse, len(sources))
	for i, src := range sources {
		resp[i] = sourceResponse{
			ID:        src.ID,
			Name:      src.Name,
			Kind:      src.Kind,
			CreatedAt: src.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /api/sources/{sourceId}/readings?metric=…&from=…&to=…
// from and to are RFC3339; both default to the last 24 hours if omitted.
func (s *Server) handleGetReadings(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("sourceId")

	metric := r.URL.Query().Get("metric")
	if metric == "" {
		http.Error(w, "metric query param is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	from, to := now.Add(-24*time.Hour), now

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			http.Error(w, "from must be RFC3339 (e.g. 2026-04-21T00:00:00Z)", http.StatusBadRequest)
			return
		}
		from = t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			http.Error(w, "to must be RFC3339 (e.g. 2026-04-21T23:59:59Z)", http.StatusBadRequest)
			return
		}
		to = t
	}

	readings, err := s.store.GetReadingsInRange(r.Context(), storage.GetReadingsInRangeParams{
		SourceID:   sourceID,
		MetricType: metric,
		FromTime:   from,
		ToTime:     to,
	})
	if err != nil {
		s.log.Error("get readings in range", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]readingResponse, len(readings))
	for i, rd := range readings {
		resp[i] = readingResponse{
			ID:         rd.ID,
			SourceID:   rd.SourceID,
			MetricType: rd.MetricType,
			Value:      rd.Value,
			Unit:       rd.Unit,
			RecordedAt: rd.RecordedAt,
			ReceivedAt: rd.ReceivedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /api/readings/latest?metric=…
// Returns the most recent reading per source for the given metric.
func (s *Server) handleLatestReadings(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		http.Error(w, "metric query param is required", http.StatusBadRequest)
		return
	}

	readings, err := s.store.GetLatestReadingsByMetric(r.Context(), metric)
	if err != nil {
		s.log.Error("get latest readings", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	sources, err := s.store.ListSources(r.Context())
	if err != nil {
		s.log.Error("list sources for latest readings", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	nameByID := make(map[string]string, len(sources))
	for _, src := range sources {
		nameByID[src.ID] = src.Name
	}

	resp := make([]latestReadingResponse, len(readings))
	for i, rd := range readings {
		resp[i] = latestReadingResponse{
			SourceID:   rd.SourceID,
			SourceName: nameByID[rd.SourceID],
			MetricType: rd.MetricType,
			Value:      rd.Value,
			Unit:       rd.Unit,
			RecordedAt: rd.RecordedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
