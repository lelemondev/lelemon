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

// EventProcessor handles the core logic of converting events to spans and storing them.
// This is the single source of truth for event processing, used by both sync and async paths.
type EventProcessor struct {
	store   repository.Store
	pricing *service.PricingCalculator
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(store repository.Store, pricing *service.PricingCalculator) *EventProcessor {
	return &EventProcessor{
		store:   store,
		pricing: pricing,
	}
}

// ProcessEvents processes a batch of events for a project
func (p *EventProcessor) ProcessEvents(ctx context.Context, projectID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Group events by TraceID (explicit) vs SessionID (legacy)
	traceGroups := make(map[string][]IngestEvent)
	sessionGroups := make(map[string][]IngestEvent)

	for _, event := range events {
		if event.TraceID != "" {
			traceGroups[event.TraceID] = append(traceGroups[event.TraceID], event)
		} else {
			sessionGroups[event.SessionID] = append(sessionGroups[event.SessionID], event)
		}
	}

	// Process trace groups
	for traceID, groupEvents := range traceGroups {
		if err := p.processTraceGroup(ctx, projectID, traceID, groupEvents); err != nil {
			slog.Error("failed to process trace group", "trace_id", traceID, "error", err)
		}
	}

	// Process session groups (legacy)
	for sessionID, groupEvents := range sessionGroups {
		if err := p.processSessionGroup(ctx, projectID, sessionID, groupEvents); err != nil {
			slog.Error("failed to process session group", "session_id", sessionID, "error", err)
		}
	}

	return nil
}

// processTraceGroup adds spans to an existing trace or creates it with the specified ID
func (p *EventProcessor) processTraceGroup(ctx context.Context, projectID, traceID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Get or create trace
	existing, err := p.store.GetTrace(ctx, projectID, traceID)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return fmt.Errorf("get trace: %w", err)
	}

	if existing == nil {
		trace := p.buildTrace(projectID, traceID, events)
		if err := p.store.CreateTrace(ctx, trace); err != nil {
			return fmt.Errorf("create trace: %w", err)
		}
	}

	// Create spans
	spans, hasErrors := p.buildSpans(traceID, events)
	if err := p.store.CreateSpans(ctx, spans); err != nil {
		return fmt.Errorf("create spans: %w", err)
	}

	// Update status if errors
	if hasErrors {
		return p.store.UpdateTraceStatus(ctx, projectID, traceID, entity.TraceStatusError)
	}

	return nil
}

// processSessionGroup creates a new trace for a session (legacy behavior)
func (p *EventProcessor) processSessionGroup(ctx context.Context, projectID, sessionID string, events []IngestEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Create trace (generates new ID)
	trace := p.buildTrace(projectID, "", events)
	if sessionID != "" {
		trace.SessionID = &sessionID
	}

	if err := p.store.CreateTrace(ctx, trace); err != nil {
		return fmt.Errorf("create trace: %w", err)
	}

	// Create spans
	spans, hasErrors := p.buildSpans(trace.ID, events)
	if err := p.store.CreateSpans(ctx, spans); err != nil {
		return fmt.Errorf("create spans: %w", err)
	}

	// Update status
	status := entity.TraceStatusCompleted
	if hasErrors {
		status = entity.TraceStatusError
	}

	return p.store.UpdateTraceStatus(ctx, projectID, trace.ID, status)
}

// buildTrace creates a trace entity from events
func (p *EventProcessor) buildTrace(projectID, traceID string, events []IngestEvent) *entity.Trace {
	firstEvent := events[0]

	trace := &entity.Trace{
		ProjectID: projectID,
		Status:    entity.TraceStatusActive,
		Metadata:  make(map[string]any),
	}

	if traceID != "" {
		trace.ID = traceID
	}

	// Extract trace name from agent span or _traceName metadata
	for _, event := range events {
		if event.SpanType == "agent" && event.Name != "" {
			trace.Name = &event.Name
			break
		}
	}
	if trace.Name == nil && firstEvent.Metadata != nil {
		if name, ok := firstEvent.Metadata["_traceName"].(string); ok && name != "" {
			trace.Name = &name
		}
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

	return trace
}

// buildSpans converts events to spans
func (p *EventProcessor) buildSpans(traceID string, events []IngestEvent) ([]entity.Span, bool) {
	spans := make([]entity.Span, 0, len(events))
	hasErrors := false

	for _, event := range events {
		span := p.EventToSpan(traceID, event)
		spans = append(spans, span)
		if event.Status == "error" {
			hasErrors = true
		}
	}

	return spans, hasErrors
}

// EventToSpan converts an IngestEvent to a Span entity.
// This is the SINGLE implementation used by both sync and async paths.
func (p *EventProcessor) EventToSpan(traceID string, event IngestEvent) entity.Span {
	now := time.Now()
	startedAt := now
	if event.Timestamp != nil {
		startedAt = *event.Timestamp
	}

	spanType := parseSpanType(event.SpanType)
	spanStatus := entity.SpanStatusSuccess
	if event.Status == "error" {
		spanStatus = entity.SpanStatusError
	}

	name := coalesce(event.Name, event.Model, string(spanType))
	metadata := p.buildMetadata(event)

	span := entity.Span{
		TraceID:    traceID,
		Type:       spanType,
		Name:       name,
		Input:      event.Input,
		DurationMs: event.DurationMs,
		Status:     spanStatus,
		Metadata:   metadata,
		StartedAt:  startedAt,
		EndedAt:    &now,
	}

	// Set optional fields
	if event.SpanID != "" {
		span.ID = event.SpanID
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
	if event.FirstTokenMs != nil {
		span.FirstTokenMs = event.FirstTokenMs
	}

	// Process response data (rawResponse or legacy fields)
	p.processResponseData(&span, event, spanType)

	return span
}

// buildMetadata constructs the metadata map for a span
func (p *EventProcessor) buildMetadata(event IngestEvent) map[string]any {
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

	// Debug info for SDK troubleshooting
	metadata["_debug"] = map[string]any{
		"hasRawResponse":  event.RawResponse != nil,
		"hasOutput":       event.Output != nil,
		"hasInputTokens":  event.InputTokens != nil,
		"hasOutputTokens": event.OutputTokens != nil,
		"provider":        event.Provider,
		"spanType":        event.SpanType,
	}

	return metadata
}

// processResponseData extracts output, tokens, and other data from the event
func (p *EventProcessor) processResponseData(span *entity.Span, event IngestEvent, spanType entity.SpanType) {
	if event.RawResponse != nil {
		p.processRawResponse(span, event, spanType)
	} else {
		p.processLegacyFields(span, event, spanType)
	}
}

// processRawResponse parses rawResponse and populates span fields
func (p *EventProcessor) processRawResponse(span *entity.Span, event IngestEvent, spanType entity.SpanType) {
	parsed := service.ParseProviderResponse(event.Provider, event.RawResponse)
	if parsed == nil {
		slog.Warn("rawResponse parsing returned nil", "provider", event.Provider)
		return
	}

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

	// Calculate cost
	if spanType == entity.SpanTypeLLM && event.Model != "" {
		cost := p.pricing.CalculateCost(event.Model, parsed.InputTokens, parsed.OutputTokens)
		span.CostUSD = &cost
	}
}

// processLegacyFields uses event fields directly (for backward compatibility)
func (p *EventProcessor) processLegacyFields(span *entity.Span, event IngestEvent, spanType entity.SpanType) {
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

	// Calculate cost
	if spanType == entity.SpanTypeLLM && event.Model != "" {
		inputTokens := derefInt(event.InputTokens)
		outputTokens := derefInt(event.OutputTokens)
		cost := p.pricing.CalculateCost(event.Model, inputTokens, outputTokens)
		span.CostUSD = &cost
	}

	// Extract subtype and tool uses from output
	if spanType == entity.SpanTypeLLM {
		subType := determineLLMSubType(event.Output)
		span.SubType = &subType

		if toolUses := extractToolUsesFromOutput(event.Output, span.ID); len(toolUses) > 0 {
			span.ToolUses = toolUses
		}
	}
}

// --- Helper functions ---

func parseSpanType(s string) entity.SpanType {
	switch s {
	case "agent":
		return entity.SpanTypeAgent
	case "tool":
		return entity.SpanTypeTool
	case "retrieval":
		return entity.SpanTypeRetrieval
	case "embedding":
		return entity.SpanTypeEmbedding
	case "guardrail":
		return entity.SpanTypeGuardrail
	case "rerank":
		return entity.SpanTypeRerank
	case "custom":
		return entity.SpanTypeCustom
	default:
		return entity.SpanTypeLLM
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func intPtr(v int) *int {
	return &v
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// determineLLMSubType determines if an LLM span is "planning" or "response"
func determineLLMSubType(output any) string {
	arr, ok := output.([]any)
	if !ok {
		return "response"
	}

	for _, block := range arr {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		// Bedrock format
		if _, ok := blockMap["toolUse"]; ok {
			return "planning"
		}
		// Anthropic format
		if blockMap["type"] == "tool_use" {
			return "planning"
		}
	}

	return "response"
}

// extractToolUsesFromOutput extracts tool calls from LLM output
func extractToolUsesFromOutput(output any, spanID string) []entity.ToolUse {
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

		var id, name string
		var input any

		// Bedrock format
		if toolUse, ok := blockMap["toolUse"].(map[string]any); ok {
			name, _ = toolUse["name"].(string)
			input = toolUse["input"]
			id, _ = toolUse["toolUseId"].(string)
		}

		// Anthropic format
		if blockMap["type"] == "tool_use" {
			name, _ = blockMap["name"].(string)
			input = blockMap["input"]
			id, _ = blockMap["id"].(string)
		}

		if name == "" {
			continue
		}

		if id == "" {
			id = fmt.Sprintf("%s-tool-%d", spanID, idx)
		}

		toolUses = append(toolUses, entity.ToolUse{
			ID:     id,
			Name:   name,
			Input:  input,
			Status: "pending",
		})
		idx++
	}

	return toolUses
}
