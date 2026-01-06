package trace

import (
	"context"

	"github.com/lelemon/server/internal/domain/entity"
	"github.com/lelemon/server/internal/domain/repository"
	"github.com/lelemon/server/internal/domain/service"
)

// Service handles trace operations
type Service struct {
	store   repository.Store
	pricing *service.PricingCalculator
}

// NewService creates a new trace service
func NewService(store repository.Store, pricing *service.PricingCalculator) *Service {
	return &Service{
		store:   store,
		pricing: pricing,
	}
}

// Create creates a new trace
func (s *Service) Create(ctx context.Context, projectID string, req *CreateTraceRequest) (*entity.Trace, error) {
	trace := &entity.Trace{
		ProjectID: projectID,
		Status:    entity.TraceStatusActive,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
	}

	if req.SessionID != "" {
		trace.SessionID = &req.SessionID
	}
	if req.UserID != "" {
		trace.UserID = &req.UserID
	}
	if trace.Tags == nil {
		trace.Tags = []string{}
	}
	if trace.Metadata == nil {
		trace.Metadata = make(map[string]any)
	}

	if err := s.store.CreateTrace(ctx, trace); err != nil {
		return nil, err
	}

	return trace, nil
}

// Get retrieves a trace with its spans
func (s *Service) Get(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error) {
	return s.store.GetTrace(ctx, projectID, traceID)
}

// GetDetail retrieves a trace with pre-processed span tree for visualization
func (s *Service) GetDetail(ctx context.Context, projectID, traceID string) (*TraceDetailResponse, error) {
	trace, err := s.store.GetTrace(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}
	return ProcessTraceDetail(trace), nil
}

// List retrieves traces with pagination and filtering
func (s *Service) List(ctx context.Context, projectID string, filter entity.TraceFilter) (*entity.Page[entity.TraceWithMetrics], error) {
	return s.store.ListTraces(ctx, projectID, filter)
}

// Update updates a trace
func (s *Service) Update(ctx context.Context, projectID, traceID string, req *UpdateTraceRequest) error {
	updates := entity.TraceUpdate{}

	if req.Status != nil {
		status := entity.TraceStatus(*req.Status)
		updates.Status = &status
	}
	if req.Metadata != nil {
		updates.Metadata = req.Metadata
	}
	if req.Tags != nil {
		updates.Tags = req.Tags
	}

	return s.store.UpdateTrace(ctx, projectID, traceID, updates)
}

// AddSpan adds a span to a trace
func (s *Service) AddSpan(ctx context.Context, projectID, traceID string, req *CreateSpanRequest) (*entity.Span, error) {
	// Verify trace exists and belongs to project
	_, err := s.store.GetTrace(ctx, projectID, traceID)
	if err != nil {
		return nil, err
	}

	// Determine span type
	spanType := entity.SpanTypeLLM
	switch req.Type {
	case "tool":
		spanType = entity.SpanTypeTool
	case "retrieval":
		spanType = entity.SpanTypeRetrieval
	case "custom":
		spanType = entity.SpanTypeCustom
	}

	// Determine status
	spanStatus := entity.SpanStatusSuccess
	if req.Status == "error" {
		spanStatus = entity.SpanStatusError
	} else if req.Status == "pending" {
		spanStatus = entity.SpanStatusPending
	}

	// Calculate cost for LLM spans
	var costUSD *float64
	if spanType == entity.SpanTypeLLM && req.Model != "" {
		inputTokens := 0
		outputTokens := 0
		if req.InputTokens != nil {
			inputTokens = *req.InputTokens
		}
		if req.OutputTokens != nil {
			outputTokens = *req.OutputTokens
		}
		cost := s.pricing.CalculateCost(req.Model, inputTokens, outputTokens)
		costUSD = &cost
	}

	span := &entity.Span{
		TraceID:      traceID,
		Type:         spanType,
		Name:         req.Name,
		Input:        req.Input,
		Output:       req.Output,
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		CostUSD:      costUSD,
		DurationMs:   req.DurationMs,
		Status:       spanStatus,
		Metadata:     req.Metadata,
	}

	if req.ParentSpanID != "" {
		span.ParentSpanID = &req.ParentSpanID
	}
	if req.ErrorMessage != "" {
		span.ErrorMessage = &req.ErrorMessage
	}
	if req.Model != "" {
		span.Model = &req.Model
	}
	if req.Provider != "" {
		span.Provider = &req.Provider
	}
	if span.Metadata == nil {
		span.Metadata = make(map[string]any)
	}

	if err := s.store.CreateSpan(ctx, span); err != nil {
		return nil, err
	}

	return span, nil
}

// ListSessions retrieves sessions with pagination
func (s *Service) ListSessions(ctx context.Context, projectID string, filter entity.SessionFilter) (*entity.Page[entity.Session], error) {
	return s.store.ListSessions(ctx, projectID, filter)
}

// DeleteAll deletes all traces for a project
func (s *Service) DeleteAll(ctx context.Context, projectID string) (int64, error) {
	return s.store.DeleteAllTraces(ctx, projectID)
}
