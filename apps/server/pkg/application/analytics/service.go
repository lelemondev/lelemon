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
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	if req.From != nil {
		from = *req.From
	}
	if req.To != nil {
		to = *req.To
	}

	return s.store.GetStats(ctx, projectID, entity.AnalyticsQuery{
		Period: entity.Period{From: from, To: to},
	})
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

	return s.store.GetUsageTimeSeries(ctx, projectID, entity.TimeSeriesOpts{
		Period:      entity.Period{From: from, To: to},
		Granularity: granularity,
	})
}

// buildQuery constructs an AnalyticsQuery from a PeriodRequest
func buildQuery(req *PeriodRequest) entity.AnalyticsQuery {
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	if req.From != nil {
		from = *req.From
	}
	if req.To != nil {
		to = *req.To
	}
	return entity.AnalyticsQuery{
		Period: entity.Period{From: from, To: to},
		Filter: entity.AnalyticsFilter{
			Tag:       req.Tag,
			SessionID: req.SessionID,
			UserID:    req.UserID,
			Name:      req.Name,
		},
	}
}

// GetModelStats returns analytics grouped by model
func (s *Service) GetModelStats(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.ModelStats, error) {
	return s.store.GetModelStats(ctx, projectID, buildQuery(req))
}

// GetTagStats returns analytics grouped by tag
func (s *Service) GetTagStats(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.TagStats, error) {
	return s.store.GetTagStats(ctx, projectID, buildQuery(req), req.Prefix)
}

// GetTopUsers returns top users by cost
func (s *Service) GetTopUsers(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.UserStats, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	return s.store.GetTopUsers(ctx, projectID, buildQuery(req), limit)
}

// GetHourlyHeatmap returns usage by hour and day of week
func (s *Service) GetHourlyHeatmap(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.HourlyHeatmap, error) {
	return s.store.GetHourlyHeatmap(ctx, projectID, buildQuery(req))
}

// GetLatencyDistribution returns latency histogram buckets
func (s *Service) GetLatencyDistribution(ctx context.Context, projectID string, req *PeriodRequest) ([]entity.LatencyBucket, error) {
	return s.store.GetLatencyDistribution(ctx, projectID, buildQuery(req))
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
