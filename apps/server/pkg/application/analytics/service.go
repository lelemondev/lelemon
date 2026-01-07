package analytics

import (
	"context"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
)

// Service handles analytics operations
type Service struct {
	store repository.Store
}

// NewService creates a new analytics service
func NewService(store repository.Store) *Service {
	return &Service{store: store}
}

// GetSummary returns aggregate statistics for a project
func (s *Service) GetSummary(ctx context.Context, projectID string, req *SummaryRequest) (*entity.Stats, error) {
	// Default to last 7 days
	to := time.Now()
	from := to.AddDate(0, 0, -7)

	if req.From != nil {
		from = *req.From
	}
	if req.To != nil {
		to = *req.To
	}

	period := entity.Period{
		From: from,
		To:   to,
	}

	return s.store.GetStats(ctx, projectID, period)
}

// GetUsage returns usage time series data
func (s *Service) GetUsage(ctx context.Context, projectID string, req *UsageRequest) ([]entity.DataPoint, error) {
	// Default to last 7 days with daily granularity
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	granularity := "day"

	if req.From != nil {
		from = *req.From
	}
	if req.To != nil {
		to = *req.To
	}
	if req.Granularity != "" {
		granularity = req.Granularity
	}

	opts := entity.TimeSeriesOpts{
		Period:      entity.Period{From: from, To: to},
		Granularity: granularity,
	}

	return s.store.GetUsageTimeSeries(ctx, projectID, opts)
}
