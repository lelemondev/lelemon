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

func (s *Service) defaultPeriod(req *PeriodRequest) entity.Period {
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	if req.From != nil {
		from = *req.From
	}
	if req.To != nil {
		to = *req.To
	}
	return entity.Period{From: from, To: to}
}

// GetModelStats returns analytics grouped by model
func (s *Service) GetModelStats(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.ModelStats, error) {
	return s.store.GetModelStats(ctx, projectID, s.defaultPeriod(req))
}

// GetTagStats returns analytics grouped by tag
func (s *Service) GetTagStats(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.TagStats, error) {
	return s.store.GetTagStats(ctx, projectID, s.defaultPeriod(req), req.Prefix)
}

// GetTopUsers returns top users by cost
func (s *Service) GetTopUsers(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.UserStats, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	return s.store.GetTopUsers(ctx, projectID, s.defaultPeriod(req), limit)
}

// GetHourlyHeatmap returns usage by hour and day of week
func (s *Service) GetHourlyHeatmap(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.HourlyHeatmap, error) {
	return s.store.GetHourlyHeatmap(ctx, projectID, s.defaultPeriod(req))
}

// GetLatencyDistribution returns latency histogram buckets
func (s *Service) GetLatencyDistribution(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.LatencyBucket, error) {
	return s.store.GetLatencyDistribution(ctx, projectID, s.defaultPeriod(req))
}

// GetLatencyTimeSeries returns p50/p95/p99 latency over time
func (s *Service) GetLatencyTimeSeries(ctx context.Context, projectID string, req *UsageRequest) ([]entity.LatencyPoint, error) {
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
	return s.store.GetLatencyTimeSeries(ctx, projectID, entity.TimeSeriesOpts{
		Period:      entity.Period{From: from, To: to},
		Granularity: granularity,
	})
}
