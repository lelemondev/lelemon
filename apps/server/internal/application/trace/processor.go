package trace

import (
	"math"
	"sort"

	"github.com/lelemon/server/internal/domain/entity"
)

// ProcessTraceDetail transforms a TraceWithSpans into an optimized TraceDetailResponse
// The heavy extraction (SubType, ToolUses) is done at ingest time for efficiency.
// This function only builds the tree structure and matches tool results.
func ProcessTraceDetail(trace *entity.TraceWithSpans) *TraceDetailResponse {
	if trace == nil {
		return nil
	}

	// Extract tool results from all spans (needed to match tool uses with their results)
	toolResults := extractAllToolResults(trace.Spans)

	// Transform spans to processed spans
	processedSpans := make([]ProcessedSpan, 0, len(trace.Spans))
	var firstLLMSpan *entity.Span

	// Find first LLM span for user input extraction
	for i := range trace.Spans {
		if trace.Spans[i].Type == entity.SpanTypeLLM {
			firstLLMSpan = &trace.Spans[i]
			break
		}
	}

	// Extract user message from first LLM span
	var userMessage *string
	if firstLLMSpan != nil {
		userMessage = extractUserMessage(firstLLMSpan.Input)
	}

	// Process each span (using pre-computed fields from ingest)
	for _, span := range trace.Spans {
		processed := spanToProcessed(span, toolResults)

		// For agent spans, add user input
		if span.Type == entity.SpanTypeAgent {
			processed.UserInput = userMessage
		}

		processedSpans = append(processedSpans, processed)
	}

	// Repair invalid parent relations
	processedSpans = repairParentRelations(processedSpans)

	// Calculate timeline context
	timeline := calculateTimelineContext(processedSpans)

	// Build span tree
	spanTree := buildSpanTree(processedSpans, timeline)

	// Calculate aggregate metrics
	totalTokens := 0
	totalCostUSD := 0.0
	totalDurationMs := 0
	for _, span := range trace.Spans {
		if span.InputTokens != nil {
			totalTokens += *span.InputTokens
		}
		if span.OutputTokens != nil {
			totalTokens += *span.OutputTokens
		}
		if span.CostUSD != nil {
			totalCostUSD += *span.CostUSD
		}
		if span.DurationMs != nil {
			totalDurationMs += *span.DurationMs
		}
	}

	return &TraceDetailResponse{
		ID:              trace.ID,
		ProjectID:       trace.ProjectID,
		SessionID:       trace.SessionID,
		UserID:          trace.UserID,
		Status:          string(trace.Status),
		Tags:            trace.Tags,
		Metadata:        trace.Metadata,
		CreatedAt:       trace.CreatedAt,
		UpdatedAt:       trace.UpdatedAt,
		TotalSpans:      len(trace.Spans),
		TotalTokens:     totalTokens,
		TotalCostUSD:    totalCostUSD,
		TotalDurationMs: totalDurationMs,
		SpanTree:        spanTree,
		Timeline:        timeline,
	}
}

// spanToProcessed converts an entity.Span to ProcessedSpan
// Uses pre-computed SubType and ToolUses from ingest, and matches tool results
func spanToProcessed(span entity.Span, toolResults map[string]toolResultData) ProcessedSpan {
	processed := ProcessedSpan{
		ID:               span.ID,
		TraceID:          span.TraceID,
		ParentSpanID:     span.ParentSpanID,
		Type:             string(span.Type),
		Name:             span.Name,
		Input:            span.Input,
		Output:           span.Output,
		InputTokens:      span.InputTokens,
		OutputTokens:     span.OutputTokens,
		CostUSD:          span.CostUSD,
		DurationMs:       span.DurationMs,
		Status:           string(span.Status),
		ErrorMessage:     span.ErrorMessage,
		Model:            span.Model,
		Provider:         span.Provider,
		Metadata:         span.Metadata,
		StartedAt:        span.StartedAt,
		EndedAt:          span.EndedAt,
		StopReason:       span.StopReason,
		CacheReadTokens:  span.CacheReadTokens,
		CacheWriteTokens: span.CacheWriteTokens,
		ReasoningTokens:  span.ReasoningTokens,
		FirstTokenMs:     span.FirstTokenMs,
		Thinking:         span.Thinking,
		SubType:          span.SubType, // Pre-computed at ingest
	}

	// Match tool uses with their results (results come from subsequent LLM requests)
	if len(span.ToolUses) > 0 {
		toolUses := make([]ToolUse, len(span.ToolUses))
		for i, tu := range span.ToolUses {
			toolUses[i] = ToolUse{
				ID:     tu.ID,
				Name:   tu.Name,
				Input:  tu.Input,
				Output: tu.Output,
				Status: tu.Status,
			}
			// Try to match with results from subsequent requests
			if result, ok := toolResults[tu.ID]; ok {
				toolUses[i].Output = result.content
				if result.status == "error" {
					toolUses[i].Status = "error"
				} else {
					toolUses[i].Status = "success"
				}
			}
		}
		processed.ToolUses = toolUses
	}

	return processed
}

// extractUserMessage extracts the first user message text from LLM input
func extractUserMessage(input any) *string {
	if input == nil {
		return nil
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil
	}

	messages, ok := inputMap["messages"].([]any)
	if !ok {
		return nil
	}

	for _, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			continue
		}

		role, _ := msgMap["role"].(string)
		if role != "user" {
			continue
		}

		content := msgMap["content"]
		// String content
		if str, ok := content.(string); ok {
			return &str
		}

		// Array content (check for tool results to skip)
		if arr, ok := content.([]any); ok {
			hasToolResult := false
			for _, block := range arr {
				blockMap, ok := block.(map[string]any)
				if !ok {
					continue
				}
				// Bedrock format
				if _, ok := blockMap["toolResult"]; ok {
					hasToolResult = true
					break
				}
				// Anthropic format
				if blockMap["type"] == "tool_result" {
					hasToolResult = true
					break
				}
			}
			if hasToolResult {
				continue
			}

			// Extract text from content blocks
			for _, block := range arr {
				blockMap, ok := block.(map[string]any)
				if !ok {
					continue
				}
				if text, ok := blockMap["text"].(string); ok {
					return &text
				}
			}
		}
	}

	return nil
}

// toolResultData holds extracted tool result information
type toolResultData struct {
	status  string
	content any
}

// extractAllToolResults extracts tool results from all span inputs
func extractAllToolResults(spans []entity.Span) map[string]toolResultData {
	results := make(map[string]toolResultData)

	for _, span := range spans {
		if span.Input == nil {
			continue
		}

		inputMap, ok := span.Input.(map[string]any)
		if !ok {
			continue
		}

		messages, ok := inputMap["messages"].([]any)
		if !ok {
			continue
		}

		for _, msg := range messages {
			msgMap, ok := msg.(map[string]any)
			if !ok {
				continue
			}

			role, _ := msgMap["role"].(string)
			if role != "user" {
				continue
			}

			content, ok := msgMap["content"].([]any)
			if !ok {
				continue
			}

			for _, block := range content {
				blockMap, ok := block.(map[string]any)
				if !ok {
					continue
				}

				// Bedrock format: { toolResult: { toolUseId, status, content } }
				if toolResult, ok := blockMap["toolResult"].(map[string]any); ok {
					toolUseID, _ := toolResult["toolUseId"].(string)
					status, _ := toolResult["status"].(string)
					content := toolResult["content"]
					if toolUseID != "" {
						results[toolUseID] = toolResultData{status: status, content: content}
					}
				}

				// Anthropic format: { type: "tool_result", tool_use_id, content }
				if blockMap["type"] == "tool_result" {
					toolUseID, _ := blockMap["tool_use_id"].(string)
					content := blockMap["content"]
					if toolUseID != "" {
						results[toolUseID] = toolResultData{status: "success", content: content}
					}
				}
			}
		}
	}

	return results
}

// repairParentRelations fixes invalid parentSpanId references
func repairParentRelations(spans []ProcessedSpan) []ProcessedSpan {
	spanIDs := make(map[string]bool)
	var agentSpanID *string

	for _, span := range spans {
		spanIDs[span.ID] = true
		if span.Type == "agent" && span.ParentSpanID == nil {
			id := span.ID
			agentSpanID = &id
		}
	}

	result := make([]ProcessedSpan, len(spans))
	for i, span := range spans {
		result[i] = span
		if span.ParentSpanID != nil && !spanIDs[*span.ParentSpanID] {
			// Invalid parent reference
			if agentSpanID != nil && span.ID != *agentSpanID {
				result[i].ParentSpanID = agentSpanID
			} else {
				result[i].ParentSpanID = nil
			}
		}
	}

	return result
}

// calculateTimelineContext computes the timeline bounds
func calculateTimelineContext(spans []ProcessedSpan) TimelineContext {
	if len(spans) == 0 {
		return TimelineContext{}
	}

	minTime := int64(math.MaxInt64)
	maxTime := int64(0)

	for _, span := range spans {
		start := span.StartedAt.UnixMilli()
		var end int64
		if span.EndedAt != nil {
			end = span.EndedAt.UnixMilli()
		} else if span.DurationMs != nil {
			end = start + int64(*span.DurationMs)
		} else {
			end = start
		}

		if start < minTime {
			minTime = start
		}
		if end > maxTime {
			maxTime = end
		}
	}

	return TimelineContext{
		MinTime:       minTime,
		MaxTime:       maxTime,
		TotalDuration: int(maxTime - minTime),
	}
}

// calculateTimelinePosition computes the position and width of a span in the timeline
func calculateTimelinePosition(span ProcessedSpan, timeline TimelineContext) (start, width float64) {
	if timeline.TotalDuration == 0 {
		return 0, 100
	}

	spanStart := span.StartedAt.UnixMilli()
	duration := 0
	if span.DurationMs != nil {
		duration = *span.DurationMs
	}

	start = float64(spanStart-timeline.MinTime) / float64(timeline.TotalDuration) * 100
	width = math.Max(float64(duration)/float64(timeline.TotalDuration)*100, 1)

	return start, width
}

// buildSpanTree constructs the hierarchical span tree
func buildSpanTree(spans []ProcessedSpan, timeline TimelineContext) []SpanNode {
	if len(spans) == 0 {
		return []SpanNode{}
	}

	// Create node map with all spans
	nodeMap := make(map[string]*SpanNode)
	for _, span := range spans {
		start, width := calculateTimelinePosition(span, timeline)
		nodeMap[span.ID] = &SpanNode{
			Span:          span,
			Children:      []SpanNode{},
			Depth:         0,
			TimelineStart: start,
			TimelineWidth: width,
		}
	}

	// FIRST: Add tool uses as children of LLM spans (before building tree)
	for _, span := range spans {
		if span.Type != "llm" || len(span.ToolUses) == 0 {
			continue
		}

		parentNode := nodeMap[span.ID]
		for _, toolUse := range span.ToolUses {
			toolSpan := ProcessedSpan{
				ID:           toolUse.ID,
				TraceID:      span.TraceID,
				ParentSpanID: &span.ID,
				Type:         "tool",
				Name:         toolUse.Name,
				Input:        toolUse.Input,
				Output:       toolUse.Output,
				Status:       toolUse.Status,
				DurationMs:   toolUse.DurationMs,
				StartedAt:    span.StartedAt,
				EndedAt:      span.EndedAt,
				IsToolUse:    true,
				ToolUseData:  &toolUse,
			}

			start, width := calculateTimelinePosition(toolSpan, timeline)
			toolNode := SpanNode{
				Span:          toolSpan,
				Children:      []SpanNode{},
				Depth:         0,
				TimelineStart: start,
				TimelineWidth: width,
			}
			parentNode.Children = append(parentNode.Children, toolNode)
		}
	}

	// THEN: Build parent-child relationships for regular spans
	var rootNodes []*SpanNode
	for _, span := range spans {
		node := nodeMap[span.ID]
		if span.ParentSpanID != nil {
			if parent, ok := nodeMap[*span.ParentSpanID]; ok {
				// Add as child (node already has its tool use children)
				parent.Children = append(parent.Children, *node)
			} else {
				rootNodes = append(rootNodes, node)
			}
		} else {
			rootNodes = append(rootNodes, node)
		}
	}

	// Set depths recursively
	setDepths(rootNodes, 0)

	// Sort children: agent first, then by time, tool uses at end
	sortNodes(rootNodes)

	// Convert to value slice
	result := make([]SpanNode, len(rootNodes))
	for i, node := range rootNodes {
		result[i] = *node
	}

	return result
}

// setDepths recursively sets the depth of nodes
func setDepths(nodes []*SpanNode, depth int) {
	for _, node := range nodes {
		node.Depth = depth
		childPtrs := make([]*SpanNode, len(node.Children))
		for i := range node.Children {
			childPtrs[i] = &node.Children[i]
		}
		setDepths(childPtrs, depth+1)
	}
}

// sortNodes recursively sorts nodes
func sortNodes(nodes []*SpanNode) {
	sort.Slice(nodes, func(i, j int) bool {
		// Agent spans first
		if nodes[i].Span.Type == "agent" && nodes[j].Span.Type != "agent" {
			return true
		}
		if nodes[i].Span.Type != "agent" && nodes[j].Span.Type == "agent" {
			return false
		}
		// Tool uses after regular spans
		if nodes[i].Span.IsToolUse && !nodes[j].Span.IsToolUse {
			return false
		}
		if !nodes[i].Span.IsToolUse && nodes[j].Span.IsToolUse {
			return true
		}
		// Then by start time
		return nodes[i].Span.StartedAt.Before(nodes[j].Span.StartedAt)
	})

	// Sort children recursively
	for _, node := range nodes {
		childPtrs := make([]*SpanNode, len(node.Children))
		for i := range node.Children {
			childPtrs[i] = &node.Children[i]
		}
		sortNodes(childPtrs)
	}
}
