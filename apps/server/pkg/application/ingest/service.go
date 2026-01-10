package ingest

import (
	"context"
	"time"

	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/domain/service"
)

// Service handles event ingestion with sync/async support
type Service struct {
	processor *EventProcessor
	worker    *Worker
	async     bool
}

// NewService creates a new ingest service (sync mode for tests)
func NewService(store repository.Store, pricing *service.PricingCalculator) *Service {
	return &Service{
		processor: NewEventProcessor(store, pricing),
		async:     false,
	}
}

// NewAsyncService creates a new ingest service with async worker
func NewAsyncService(store repository.Store, pricing *service.PricingCalculator, bufferSize, workers int) *Service {
	processor := NewEventProcessor(store, pricing)
	worker := NewWorker(processor, bufferSize)
	worker.Start(workers)

	return &Service{
		processor: processor,
		worker:    worker,
		async:     true,
	}
}

// Stop gracefully shuts down the async worker
func (s *Service) Stop(timeout time.Duration) {
	if s.worker != nil {
		s.worker.Stop(timeout)
	}
}

// Ingest processes a batch of events
// In async mode: enqueues and returns immediately
// In sync mode: processes synchronously
func (s *Service) Ingest(ctx context.Context, projectID string, req *IngestRequest) (*IngestResponse, error) {
	if len(req.Events) == 0 {
		return &IngestResponse{Success: true, Processed: 0}, nil
	}

	// Async mode: enqueue and return
	if s.async && s.worker != nil {
		queued := s.worker.Enqueue(Job{
			ProjectID: projectID,
			Events:    req.Events,
		})
		return &IngestResponse{
			Success:   queued,
			Processed: len(req.Events),
		}, nil
	}

	// Sync mode: process directly
	err := s.processor.ProcessEvents(ctx, projectID, req.Events)
	if err != nil {
		return &IngestResponse{
			Success:   false,
			Processed: 0,
			Errors:    []IngestError{{Index: 0, Message: err.Error()}},
		}, nil
	}

	return &IngestResponse{
		Success:   true,
		Processed: len(req.Events),
	}, nil
}
