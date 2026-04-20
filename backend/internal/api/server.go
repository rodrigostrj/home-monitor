package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	log *slog.Logger
	mux *http.ServeMux
}

func NewServer(log *slog.Logger) http.Handler {
	s := &Server{log: log, mux: http.NewServeMux()}
	s.routes()
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/sources/{sourceId}/readings", s.handleIngest)
}

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

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("sourceId")

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	s.log.Info("readings received",
		"source_id", sourceID,
		"recorded_at", req.RecordedAt,
		"sample_count", len(req.Samples),
	)
	for _, sm := range req.Samples {
		s.log.Info("sample",
			"source_id", sourceID,
			"metric_type", sm.MetricType,
			"value", sm.Value,
			"unit", sm.Unit,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]int{"stored": len(req.Samples)})
}
