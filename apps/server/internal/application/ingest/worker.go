package ingest

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/lelemon/server/internal/domain/entity"
	"github.com/lelemon/server/internal/domain/repository"
	"github.com/lelemon/server/internal/domain/service"
)

// Job represents an ingest job to be processed
type Job struct {
	ProjectID string
	Events    []IngestEvent
}

// Worker processes ingest jobs asynchronously
type Worker struct {
	store    repository.Store
	pricing  *service.PricingCalculator
	jobs     chan Job
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewWorker creates a new ingest worker
// bufferSize: max jobs to queue before blocking
func NewWorker(store repository.Store, pricing *service.PricingCalculator, bufferSize int) *Worker {
	return &Worker{
		store:    store,
		pricing:  pricing,
		jobs:     make(chan Job, bufferSize),
		shutdown: make(chan struct{}),
	}
}

// Start begins processing jobs in background
func (w *Worker) Start(workers int) {
	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go w.run(i)
	}
	slog.Info("ingest worker started", "workers", workers, "buffer_size", cap(w.jobs))
}

// Enqueue adds a job to the queue (non-blocking if buffer available)
func (w *Worker) Enqueue(job Job) bool {
	select {
	case w.jobs <- job:
		return true
	default:
		// Buffer full - log warning but don't block
		slog.Warn("ingest queue full, dropping job", "project_id", job.ProjectID, "events", len(job.Events))
		return false
	}
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop(timeout time.Duration) {
	close(w.shutdown)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("ingest worker stopped gracefully")
	case <-time.After(timeout):
		slog.Warn("ingest worker shutdown timeout", "pending_jobs", len(w.jobs))
	}
}

// QueueSize returns current number of pending jobs
func (w *Worker) QueueSize() int {
	return len(w.jobs)
}

func (w *Worker) run(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.shutdown:
			// Drain remaining jobs before exit
			w.drain(id)
			return
		case job := <-w.jobs:
			w.processJob(job)
		}
	}
}

func (w *Worker) drain(id int) {
	for {
		select {
		case job := <-w.jobs:
			w.processJob(job)
		default:
			return
		}
	}
}

func (w *Worker) processJob(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.processEvents(ctx, job.ProjectID, job.Events); err != nil {
		slog.Error("failed to process ingest job",
			"project_id", job.ProjectID,
			"events", len(job.Events),
			"error", err,
		)
	}
}

func (w *Worker) processEvents(ctx context.Context, projectID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Group events by TraceID (explicit hierarchy) vs no TraceID (legacy)
	traceGroups := make(map[string][]IngestEvent)   // Events with explicit traceId
	sessionGroups := make(map[string][]IngestEvent) // Events without traceId (legacy)

	for _, event := range events {
		if event.TraceID != "" {
			// Has explicit trace ID from SDK - group by trace
			traceGroups[event.TraceID] = append(traceGroups[event.TraceID], event)
		} else {
			// No trace ID - group by session (legacy behavior, creates new trace)
			sessionGroups[event.SessionID] = append(sessionGroups[event.SessionID], event)
		}
	}

	// Process events with explicit traceId
	for traceID, groupEvents := range traceGroups {
		if err := w.processTraceGroup(ctx, projectID, traceID, groupEvents); err != nil {
			slog.Error("failed to process trace group",
				"project_id", projectID,
				"trace_id", traceID,
				"error", err,
			)
			// Continue processing other groups
		}
	}

	// Process events without traceId (legacy)
	for sessionID, groupEvents := range sessionGroups {
		if err := w.processSessionGroup(ctx, projectID, sessionID, groupEvents); err != nil {
			slog.Error("failed to process session group",
				"project_id", projectID,
				"session_id", sessionID,
				"error", err,
			)
			// Continue processing other groups
		}
	}

	return nil
}

// processTraceGroup adds spans to an existing trace or creates it with the specified ID
func (w *Worker) processTraceGroup(ctx context.Context, projectID, traceID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	firstEvent := events[0]

	// Try to get existing trace
	existing, err := w.store.GetTrace(ctx, projectID, traceID)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return err
	}

	// If trace doesn't exist, create it with the specified ID
	if existing == nil {
		trace := &entity.Trace{
			ID:        traceID, // Use the SDK-provided trace ID
			ProjectID: projectID,
			Status:    entity.TraceStatusActive,
			Metadata:  make(map[string]any),
		}

		if firstEvent.SessionID != "" {
			trace.SessionID = &firstEvent.SessionID
		}
		if firstEvent.UserID != "" {
			trace.UserID = &firstEvent.UserID
		}
		if firstEvent.Tags != nil {
			trace.Tags = firstEvent.Tags
		}
		if firstEvent.Input != nil {
			trace.Metadata["input"] = firstEvent.Input
		}
		if firstEvent.Metadata != nil {
			for k, v := range firstEvent.Metadata {
				trace.Metadata[k] = v
			}
		}

		if err := w.store.CreateTrace(ctx, trace); err != nil {
			return err
		}
	}

	// Create spans for all events
	spans := make([]entity.Span, 0, len(events))
	hasErrors := false

	for _, event := range events {
		span := w.eventToSpan(traceID, event)
		spans = append(spans, span)
		if event.Status == "error" {
			hasErrors = true
		}
	}

	if err := w.store.CreateSpans(ctx, spans); err != nil {
		return err
	}

	// Update trace status if there are errors
	if hasErrors {
		return w.store.UpdateTraceStatus(ctx, projectID, traceID, entity.TraceStatusError)
	}

	return nil
}

func (w *Worker) processSessionGroup(ctx context.Context, projectID, sessionID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	firstEvent := events[0]

	// Create trace
	trace := &entity.Trace{
		ProjectID: projectID,
		Status:    entity.TraceStatusActive,
		Metadata:  make(map[string]any),
	}

	if sessionID != "" {
		trace.SessionID = &sessionID
	}
	if firstEvent.UserID != "" {
		trace.UserID = &firstEvent.UserID
	}
	if firstEvent.Tags != nil {
		trace.Tags = firstEvent.Tags
	}
	if firstEvent.Input != nil {
		trace.Metadata["input"] = firstEvent.Input
	}
	if firstEvent.Metadata != nil {
		for k, v := range firstEvent.Metadata {
			trace.Metadata[k] = v
		}
	}

	if err := w.store.CreateTrace(ctx, trace); err != nil {
		return err
	}

	// Create spans
	spans := make([]entity.Span, 0, len(events))
	hasErrors := false

	for _, event := range events {
		span := w.eventToSpan(trace.ID, event)
		spans = append(spans, span)
		if event.Status == "error" {
			hasErrors = true
		}
	}

	if err := w.store.CreateSpans(ctx, spans); err != nil {
		return err
	}

	// Update trace status
	status := entity.TraceStatusCompleted
	if hasErrors {
		status = entity.TraceStatusError
	}

	return w.store.UpdateTraceStatus(ctx, projectID, trace.ID, status)
}

func (w *Worker) eventToSpan(traceID string, event IngestEvent) entity.Span {
	now := time.Now()
	startedAt := now
	if event.Timestamp != nil {
		startedAt = *event.Timestamp
	}

	spanType := entity.SpanTypeLLM
	switch event.SpanType {
	case "agent":
		spanType = entity.SpanTypeAgent
	case "tool":
		spanType = entity.SpanTypeTool
	case "retrieval":
		spanType = entity.SpanTypeRetrieval
	case "embedding":
		spanType = entity.SpanTypeEmbedding
	case "guardrail":
		spanType = entity.SpanTypeGuardrail
	case "rerank":
		spanType = entity.SpanTypeRerank
	case "custom":
		spanType = entity.SpanTypeCustom
	}

	spanStatus := entity.SpanStatusSuccess
	if event.Status == "error" {
		spanStatus = entity.SpanStatusError
	}

	var costUSD *float64
	if spanType == entity.SpanTypeLLM && event.Model != "" {
		inputTokens := 0
		outputTokens := 0
		if event.InputTokens != nil {
			inputTokens = *event.InputTokens
		}
		if event.OutputTokens != nil {
			outputTokens = *event.OutputTokens
		}
		cost := w.pricing.CalculateCost(event.Model, inputTokens, outputTokens)
		costUSD = &cost
	}

	name := event.Name
	if name == "" {
		name = event.Model
	}
	if name == "" {
		name = string(spanType)
	}

	metadata := make(map[string]any)
	if event.Streaming {
		metadata["streaming"] = true
	}
	if event.ToolCallID != "" {
		metadata["toolCallId"] = event.ToolCallID
	}
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}

	span := entity.Span{
		TraceID:      traceID,
		Type:         spanType,
		Name:         name,
		Input:        event.Input,
		Output:       event.Output,
		InputTokens:  event.InputTokens,
		OutputTokens: event.OutputTokens,
		CostUSD:      costUSD,
		DurationMs:   event.DurationMs,
		Status:       spanStatus,
		Metadata:     metadata,
		StartedAt:    startedAt,
		EndedAt:      &now,
	}

	if event.ParentSpanID != "" {
		span.ParentSpanID = &event.ParentSpanID
	}
	if event.ErrorMessage != "" {
		span.ErrorMessage = &event.ErrorMessage
	}
	if spanType == entity.SpanTypeLLM && event.Model != "" {
		span.Model = &event.Model
	}
	if event.Provider != "" {
		span.Provider = &event.Provider
	}

	return span
}
