package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/domain/service"
)

// Service handles event ingestion
type Service struct {
	store   repository.Store
	pricing *service.PricingCalculator
	worker  *Worker
	async   bool
}

// NewService creates a new ingest service (sync mode for tests)
func NewService(store repository.Store, pricing *service.PricingCalculator) *Service {
	return &Service{
		store:   store,
		pricing: pricing,
		async:   false,
	}
}

// NewAsyncService creates a new ingest service with async worker
func NewAsyncService(store repository.Store, pricing *service.PricingCalculator, bufferSize, workers int) *Service {
	worker := NewWorker(store, pricing, bufferSize)
	worker.Start(workers)

	return &Service{
		store:   store,
		pricing: pricing,
		worker:  worker,
		async:   true,
	}
}

// Stop gracefully shuts down the async worker
func (s *Service) Stop(timeout time.Duration) {
	if s.worker != nil {
		s.worker.Stop(timeout)
	}
}

// Ingest processes a batch of events
// In async mode: enqueues and returns immediately (fire-and-forget)
// In sync mode: processes synchronously (for tests)
func (s *Service) Ingest(ctx context.Context, projectID string, req *IngestRequest) (*IngestResponse, error) {
	if len(req.Events) == 0 {
		return &IngestResponse{Success: true, Processed: 0}, nil
	}

	// Async mode: enqueue and return immediately
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

	// Sync mode: process immediately (for tests)
	return s.ingestSync(ctx, projectID, req)
}

// ingestSync processes events synchronously
func (s *Service) ingestSync(ctx context.Context, projectID string, req *IngestRequest) (*IngestResponse, error) {
	type indexedEvent struct {
		event IngestEvent
		index int
	}

	// Group events by TraceID (explicit hierarchy) vs no TraceID (legacy)
	traceGroups := make(map[string][]indexedEvent)    // Events with explicit traceId
	sessionGroups := make(map[string][]indexedEvent)  // Events without traceId (legacy)

	for i, event := range req.Events {
		ie := indexedEvent{event: event, index: i}
		if event.TraceID != "" {
			// Has explicit trace ID from SDK - group by trace
			traceGroups[event.TraceID] = append(traceGroups[event.TraceID], ie)
		} else {
			// No trace ID - group by session (legacy behavior, creates new trace)
			sessionGroups[event.SessionID] = append(sessionGroups[event.SessionID], ie)
		}
	}

	var errors []IngestError
	processed := 0

	// Process events with explicit traceId - add to existing trace or create with that ID
	for traceID, events := range traceGroups {
		evts := make([]IngestEvent, len(events))
		for i, e := range events {
			evts[i] = e.event
		}

		err := s.processTraceEvents(ctx, projectID, traceID, evts)
		if err != nil {
			for _, e := range events {
				errors = append(errors, IngestError{Index: e.index, Message: err.Error()})
			}
		} else {
			processed += len(events)
		}
	}

	// Process events without traceId - legacy behavior (creates new traces)
	for sessionID, events := range sessionGroups {
		evts := make([]IngestEvent, len(events))
		for i, e := range events {
			evts[i] = e.event
		}

		err := s.processSessionEvents(ctx, projectID, sessionID, evts)
		if err != nil {
			for _, e := range events {
				errors = append(errors, IngestError{Index: e.index, Message: err.Error()})
			}
		} else {
			processed += len(events)
		}
	}

	return &IngestResponse{
		Success:   len(errors) == 0,
		Processed: processed,
		Errors:    errors,
	}, nil
}

// processTraceEvents adds spans to an existing trace or creates it with the specified ID
func (s *Service) processTraceEvents(ctx context.Context, projectID, traceID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	firstEvent := events[0]

	// Try to get existing trace
	existing, err := s.store.GetTrace(ctx, projectID, traceID)
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

		if err := s.store.CreateTrace(ctx, trace); err != nil {
			return err
		}
	}

	// Create spans for all events
	spans := make([]entity.Span, 0, len(events))
	hasErrors := false

	for _, event := range events {
		span := s.eventToSpan(traceID, event)
		spans = append(spans, span)
		if event.Status == "error" {
			hasErrors = true
		}
	}

	if err := s.store.CreateSpans(ctx, spans); err != nil {
		return err
	}

	// Update trace status based on span errors
	// Note: We keep it active if there's a mix, only completed/error when all done
	// For now, mark as error if any span has error, otherwise keep active
	if hasErrors {
		return s.store.UpdateTraceStatus(ctx, projectID, traceID, entity.TraceStatusError)
	}

	return nil
}

// processSessionEvents creates a trace and spans for a session group
func (s *Service) processSessionEvents(ctx context.Context, projectID, sessionID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	firstEvent := events[0]

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

	if err := s.store.CreateTrace(ctx, trace); err != nil {
		return err
	}

	spans := make([]entity.Span, 0, len(events))
	hasErrors := false

	for _, event := range events {
		span := s.eventToSpan(trace.ID, event)
		spans = append(spans, span)
		if event.Status == "error" {
			hasErrors = true
		}
	}

	if err := s.store.CreateSpans(ctx, spans); err != nil {
		return err
	}

	status := entity.TraceStatusCompleted
	if hasErrors {
		status = entity.TraceStatusError
	}

	return s.store.UpdateTraceStatus(ctx, projectID, trace.ID, status)
}

// eventToSpan converts an IngestEvent to a Span
func (s *Service) eventToSpan(traceID string, event IngestEvent) entity.Span {
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
		TraceID:   traceID,
		Type:      spanType,
		Name:      name,
		Input:     event.Input,
		DurationMs: event.DurationMs,
		Status:    spanStatus,
		Metadata:  metadata,
		StartedAt: startedAt,
		EndedAt:   &now,
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

	// SDK-provided timing (can't be extracted from response)
	if event.FirstTokenMs != nil {
		span.FirstTokenMs = event.FirstTokenMs
	}

	// Parse RawResponse if available (SDK thin, server smart)
	slog.Info("eventToSpan", "provider", event.Provider, "hasRawResponse", event.RawResponse != nil, "model", event.Model)
	if event.RawResponse != nil {
		// Debug: log the raw response structure to see what we're receiving
		if rawMap, ok := event.RawResponse.(map[string]any); ok {
			slog.Info("rawResponse keys", "keys", getMapKeys(rawMap))
			if usage, ok := rawMap["usage"]; ok {
				slog.Info("rawResponse.usage", "usage", usage, "type", fmt.Sprintf("%T", usage))
			} else {
				slog.Warn("rawResponse missing 'usage' field")
			}
		}
		slog.Info("parsing rawResponse", "provider", event.Provider)
		parsed := service.ParseProviderResponse(event.Provider, event.RawResponse)
		if parsed != nil {
			slog.Debug("parsed rawResponse", "output", parsed.Output != nil, "inputTokens", parsed.InputTokens, "outputTokens", parsed.OutputTokens, "toolUses", len(parsed.ToolUses))
			span.Output = parsed.Output
			span.InputTokens = intPtr(parsed.InputTokens)
			span.OutputTokens = intPtr(parsed.OutputTokens)
			span.CacheReadTokens = parsed.CacheReadTokens
			span.CacheWriteTokens = parsed.CacheWriteTokens
			span.ReasoningTokens = parsed.ReasoningTokens
			span.StopReason = parsed.StopReason
			span.Thinking = parsed.Thinking
			span.SubType = parsed.SubType
			span.ToolUses = parsed.ToolUses

			// Calculate cost with extracted tokens
			if spanType == entity.SpanTypeLLM && event.Model != "" {
				cost := s.pricing.CalculateCost(event.Model, parsed.InputTokens, parsed.OutputTokens)
				span.CostUSD = &cost
			}
		} else {
			slog.Warn("rawResponse parsing returned nil", "provider", event.Provider)
		}
	} else {
		slog.Debug("no rawResponse, using legacy mode", "provider", event.Provider, "hasOutput", event.Output != nil)
		// Legacy mode: use fields from event directly
		span.Output = event.Output
		span.InputTokens = event.InputTokens
		span.OutputTokens = event.OutputTokens
		span.CacheReadTokens = event.CacheReadTokens
		span.CacheWriteTokens = event.CacheWriteTokens
		span.ReasoningTokens = event.ReasoningTokens
		if event.StopReason != "" {
			span.StopReason = &event.StopReason
		}
		if event.Thinking != "" {
			span.Thinking = &event.Thinking
		}

		// Calculate cost with legacy tokens
		if spanType == entity.SpanTypeLLM && event.Model != "" {
			inputTokens := 0
			outputTokens := 0
			if event.InputTokens != nil {
				inputTokens = *event.InputTokens
			}
			if event.OutputTokens != nil {
				outputTokens = *event.OutputTokens
			}
			cost := s.pricing.CalculateCost(event.Model, inputTokens, outputTokens)
			span.CostUSD = &cost
		}

		// Extract subtype and tool uses from legacy output
		if spanType == entity.SpanTypeLLM {
			subType := determineLLMSubType(event.Output)
			span.SubType = &subType

			toolUses := extractToolUsesFromOutput(event.Output, span.ID)
			if len(toolUses) > 0 {
				span.ToolUses = toolUses
			}
		}
	}

	return span
}

// intPtr creates a pointer to an int value
func intPtr(v int) *int {
	return &v
}

// getMapKeys returns all keys from a map (for debugging)
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// determineLLMSubType determines if an LLM span is "planning" or "response"
func determineLLMSubType(output any) string {
	if output == nil {
		return "response"
	}

	arr, ok := output.([]any)
	if !ok {
		return "response"
	}

	for _, block := range arr {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}

		// Bedrock format: { toolUse: { ... } }
		if _, ok := blockMap["toolUse"]; ok {
			return "planning"
		}

		// Anthropic format: { type: "tool_use", ... }
		if blockMap["type"] == "tool_use" {
			return "planning"
		}
	}

	return "response"
}

// extractToolUsesFromOutput extracts tool calls from LLM output
func extractToolUsesFromOutput(output any, spanID string) []entity.ToolUse {
	if output == nil {
		return nil
	}

	arr, ok := output.([]any)
	if !ok {
		return nil
	}

	var toolUses []entity.ToolUse
	idx := 0

	for _, block := range arr {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}

		var originalID, name string
		var input any

		// Bedrock format: { toolUse: { name, input, toolUseId } }
		if toolUseMap, ok := blockMap["toolUse"].(map[string]any); ok {
			name, _ = toolUseMap["name"].(string)
			input = toolUseMap["input"]
			originalID, _ = toolUseMap["toolUseId"].(string)
		}

		// Anthropic format: { type: "tool_use", id, name, input }
		if blockMap["type"] == "tool_use" {
			name, _ = blockMap["name"].(string)
			input = blockMap["input"]
			originalID, _ = blockMap["id"].(string)
		}

		if name == "" {
			continue
		}

		// Use original ID for result matching, fallback to synthetic ID
		id := originalID
		if id == "" {
			id = fmt.Sprintf("%s-tool-%d", spanID, idx)
		}

		toolUses = append(toolUses, entity.ToolUse{
			ID:     id, // Preserve original ID for result matching
			Name:   name,
			Input:  input,
			Status: "pending", // Will be matched with results at query time
		})
		idx++
	}

	return toolUses
}
