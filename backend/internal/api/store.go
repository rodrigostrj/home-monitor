package api

import (
	"context"

	"github.com/rodrigo/home-monitor/internal/storage"
)

type Store interface {
	GetSourceByID(ctx context.Context, id string) (storage.Source, error)
	ListSources(ctx context.Context) ([]storage.Source, error)
	InsertReading(ctx context.Context, arg storage.InsertReadingParams) error
	GetReadingsInRange(ctx context.Context, arg storage.GetReadingsInRangeParams) ([]storage.Reading, error)
	GetLatestReadingsByMetric(ctx context.Context, metricType string) ([]storage.Reading, error)
}
