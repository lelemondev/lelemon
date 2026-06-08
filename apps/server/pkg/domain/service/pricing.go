package service

import (
	"math"
	"strings"
)

// ModelPricing contains pricing per 1K tokens.
//
// CacheRead, CacheWrite and Reasoning are optional: when a table entry leaves
// them at 0, findPricing derives them from Input/Output using provider-specific
// multipliers (see deriveRates). Set them explicitly to override the derivation.
type ModelPricing struct {
	Input      float64 // per 1K tokens
	Output     float64 // per 1K tokens
	CacheRead  float64 // per 1K cached-read (cache hit) tokens
	CacheWrite float64 // per 1K cache-creation (cache write) tokens
	Reasoning  float64 // per 1K reasoning/thinking tokens
}

// pricing is the internal pricing table (prices per 1K tokens)
// Last updated: January 2026
var pricing = map[string]ModelPricing{
	// ==================== OpenAI ====================
	// GPT-5 series (2025-2026) - Latest flagship
	"gpt-5":       {Input: 0.00125, Output: 0.01},
	"gpt-5.1":     {Input: 0.00125, Output: 0.01},
	"gpt-5.2":     {Input: 0.00125, Output: 0.01},
	"gpt-5-mini":  {Input: 0.00025, Output: 0.002},
	"gpt-5-nano":  {Input: 0.00005, Output: 0.0004},

	// GPT-4.1 series
	"gpt-4.1":      {Input: 0.002, Output: 0.008},
	"gpt-4.1-mini": {Input: 0.0004, Output: 0.0016},
	"gpt-4.1-nano": {Input: 0.0001, Output: 0.0004},

	// o3 reasoning models (2026)
	"o3":      {Input: 0.002, Output: 0.008},
	"o3-mini": {Input: 0.0011, Output: 0.0044},
	"o3-pro":  {Input: 0.02, Output: 0.08},

	// o1 reasoning models
	"o1":         {Input: 0.015, Output: 0.06},
	"o1-preview": {Input: 0.015, Output: 0.06},
	"o1-mini":    {Input: 0.003, Output: 0.012},
	"o1-pro":     {Input: 0.15, Output: 0.6},

	// GPT-4o series
	"gpt-4o":                 {Input: 0.0025, Output: 0.01},
	"gpt-4o-2024-05-13":      {Input: 0.0025, Output: 0.01},
	"gpt-4o-2024-08-06":      {Input: 0.0025, Output: 0.01},
	"gpt-4o-2024-11-20":      {Input: 0.0025, Output: 0.01},
	"gpt-4o-mini":            {Input: 0.00015, Output: 0.0006},
	"gpt-4o-mini-2024-07-18": {Input: 0.00015, Output: 0.0006},

	// GPT-4 Turbo (legacy)
	"gpt-4-turbo":            {Input: 0.01, Output: 0.03},
	"gpt-4-turbo-preview":    {Input: 0.01, Output: 0.03},
	"gpt-4-turbo-2024-04-09": {Input: 0.01, Output: 0.03},

	// GPT-4 (legacy)
	"gpt-4":     {Input: 0.03, Output: 0.06},
	"gpt-4-32k": {Input: 0.06, Output: 0.12},

	// GPT-3.5 (legacy)
	"gpt-3.5-turbo":     {Input: 0.0005, Output: 0.0015},
	"gpt-3.5-turbo-16k": {Input: 0.003, Output: 0.004},

	// ==================== Anthropic (API directa) ====================
	// Claude 4.5 - $5/$25 (Opus), $3/$15 (Sonnet), $1/$5 (Haiku) per MTok
	"claude-opus-4.5":            {Input: 0.005, Output: 0.025},
	"claude-opus-4-5":            {Input: 0.005, Output: 0.025},
	"claude-opus-4-5-20251101":   {Input: 0.005, Output: 0.025},
	"claude-opus-4.5-latest":     {Input: 0.005, Output: 0.025},
	"claude-sonnet-4.5":          {Input: 0.003, Output: 0.015},
	"claude-sonnet-4-5":          {Input: 0.003, Output: 0.015},
	"claude-sonnet-4-5-20250514": {Input: 0.003, Output: 0.015},
	"claude-sonnet-4.5-latest":   {Input: 0.003, Output: 0.015},
	"claude-haiku-4.5":           {Input: 0.001, Output: 0.005},
	"claude-haiku-4-5":           {Input: 0.001, Output: 0.005},
	"claude-haiku-4-5-20250514":  {Input: 0.001, Output: 0.005},
	"claude-haiku-4.5-latest":    {Input: 0.001, Output: 0.005},

	// Claude 4.1 - $15/$75 per MTok
	"claude-opus-4.1":          {Input: 0.015, Output: 0.075},
	"claude-opus-4-1":          {Input: 0.015, Output: 0.075},
	"claude-opus-4-1-20250410": {Input: 0.015, Output: 0.075},
	"claude-opus-4.1-latest":   {Input: 0.015, Output: 0.075},

	// Claude 4 - $3/$15 (Sonnet), $15/$75 (Opus) per MTok
	"claude-sonnet-4":          {Input: 0.003, Output: 0.015},
	"claude-sonnet-4-20250514": {Input: 0.003, Output: 0.015},
	"claude-sonnet-4-latest":   {Input: 0.003, Output: 0.015},
	"claude-opus-4":            {Input: 0.015, Output: 0.075},
	"claude-opus-4-20250410":   {Input: 0.015, Output: 0.075},
	"claude-opus-4-latest":     {Input: 0.015, Output: 0.075},

	// Claude 3.7 Sonnet - $3/$15 per MTok
	"claude-3.7-sonnet":          {Input: 0.003, Output: 0.015},
	"claude-3-7-sonnet":          {Input: 0.003, Output: 0.015},
	"claude-3-7-sonnet-20250219": {Input: 0.003, Output: 0.015},
	"claude-3.7-sonnet-latest":   {Input: 0.003, Output: 0.015},

	// Claude 3.5 series
	"claude-3.5-sonnet":          {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet":          {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-20241022": {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-20240620": {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-latest":   {Input: 0.003, Output: 0.015},
	"claude-3.5-haiku":           {Input: 0.001, Output: 0.005},
	"claude-3-5-haiku":           {Input: 0.001, Output: 0.005},
	"claude-3-5-haiku-20241022":  {Input: 0.001, Output: 0.005},
	"claude-3-5-haiku-latest":    {Input: 0.001, Output: 0.005},

	// Claude 3 series (legacy) - $0.25/$1.25 (Haiku), $3/$15 (Sonnet), $15/$75 (Opus) per MTok
	"claude-3-opus":            {Input: 0.015, Output: 0.075},
	"claude-3-opus-20240229":   {Input: 0.015, Output: 0.075},
	"claude-3-opus-latest":     {Input: 0.015, Output: 0.075},
	"claude-3-sonnet":          {Input: 0.003, Output: 0.015},
	"claude-3-sonnet-20240229": {Input: 0.003, Output: 0.015},
	"claude-3-haiku":           {Input: 0.00025, Output: 0.00125},
	"claude-3-haiku-20240307":  {Input: 0.00025, Output: 0.00125},

	// ==================== AWS Bedrock (Anthropic) ====================
	// Claude 4.5 on Bedrock
	"anthropic.claude-opus-4-5":   {Input: 0.005, Output: 0.025},
	"anthropic.claude-sonnet-4-5": {Input: 0.003, Output: 0.015},
	"anthropic.claude-haiku-4-5":  {Input: 0.001, Output: 0.005},

	// Claude 4 on Bedrock
	"anthropic.claude-opus-4":   {Input: 0.015, Output: 0.075},
	"anthropic.claude-sonnet-4": {Input: 0.003, Output: 0.015},

	// Claude 3.x on Bedrock
	"anthropic.claude-3-5-sonnet": {Input: 0.003, Output: 0.015},
	"anthropic.claude-3-5-haiku":  {Input: 0.001, Output: 0.005},
	"anthropic.claude-3-opus":     {Input: 0.015, Output: 0.075},
	"anthropic.claude-3-sonnet":   {Input: 0.003, Output: 0.015},
	"anthropic.claude-3-haiku":    {Input: 0.00025, Output: 0.00125},

	// Cross-region inference (us.)
	"us.anthropic.claude-opus-4-5":   {Input: 0.005, Output: 0.025},
	"us.anthropic.claude-sonnet-4-5": {Input: 0.003, Output: 0.015},
	"us.anthropic.claude-haiku-4-5":  {Input: 0.001, Output: 0.005},
	"us.anthropic.claude-opus-4":     {Input: 0.015, Output: 0.075},
	"us.anthropic.claude-sonnet-4":   {Input: 0.003, Output: 0.015},
	"us.anthropic.claude-3-5-sonnet": {Input: 0.003, Output: 0.015},
	"us.anthropic.claude-3-5-haiku":  {Input: 0.001, Output: 0.005},
	"us.anthropic.claude-3-opus":     {Input: 0.015, Output: 0.075},
	"us.anthropic.claude-3-sonnet":   {Input: 0.003, Output: 0.015},
	"us.anthropic.claude-3-haiku":    {Input: 0.00025, Output: 0.00125},

	// ==================== Google Gemini ====================
	// Gemini 3 series (2026) - Latest flagship
	"gemini-3-pro":               {Input: 0.002, Output: 0.012},
	"gemini-3-pro-latest":        {Input: 0.002, Output: 0.012},
	"gemini-3-pro-preview":       {Input: 0.002, Output: 0.012},
	"gemini-3-flash":             {Input: 0.0005, Output: 0.003},
	"gemini-3-flash-latest":      {Input: 0.0005, Output: 0.003},
	"gemini-3-flash-preview":     {Input: 0.0005, Output: 0.003},
	"gemini-3-pro-image":         {Input: 0.002, Output: 0.012},
	"gemini-3-pro-image-preview": {Input: 0.002, Output: 0.012},

	// Gemini 2.5 series
	"gemini-2.5-pro":          {Input: 0.00125, Output: 0.01},
	"gemini-2.5-pro-latest":   {Input: 0.00125, Output: 0.01},
	"gemini-2.5-flash":        {Input: 0.0003, Output: 0.0025},
	"gemini-2.5-flash-latest": {Input: 0.0003, Output: 0.0025},
	"gemini-2.5-flash-lite":   {Input: 0.0001, Output: 0.0004},

	// Gemini 2.0 series
	"gemini-2.0-flash":          {Input: 0.0001, Output: 0.0004},
	"gemini-2.0-flash-exp":      {Input: 0.0001, Output: 0.0004},
	"gemini-2.0-flash-lite":     {Input: 0.000075, Output: 0.0003},
	"gemini-2.0-flash-thinking": {Input: 0.0001, Output: 0.0004},

	// Gemini 1.5 series (legacy)
	"gemini-1.5-pro":          {Input: 0.00125, Output: 0.005},
	"gemini-1.5-pro-latest":   {Input: 0.00125, Output: 0.005},
	"gemini-1.5-flash":        {Input: 0.000075, Output: 0.0003},
	"gemini-1.5-flash-latest": {Input: 0.000075, Output: 0.0003},
	"gemini-1.5-flash-8b":     {Input: 0.0000375, Output: 0.00015},

	// Gemini 1.0 (legacy)
	"gemini-1.0-pro": {Input: 0.0005, Output: 0.0015},
	"gemini-pro":     {Input: 0.0005, Output: 0.0015},
}

// defaultPricing is used for unknown models ($0 = transparent indicator)
var defaultPricing = ModelPricing{Input: 0, Output: 0}

// Cache/reasoning multipliers (relative to Input, except Reasoning which is
// relative to Output) used to derive rates that the table leaves at 0.
//
// Sources (Jan 2026):
//   - Anthropic: cache write 1.25x input, cache read 0.1x input; reasoning billed as output.
//   - OpenAI: cached input ~0.5x input, no separate cache-write surcharge; reasoning billed as output.
//   - Gemini: implicit cache read ~0.25x input, no separate cache-write surcharge; reasoning billed as output.
const (
	anthropicCacheWriteMult = 1.25
	anthropicCacheReadMult  = 0.10
	openaiCacheReadMult     = 0.50
	geminiCacheReadMult     = 0.25
)

// deriveRates fills in CacheRead/CacheWrite/Reasoning when a table entry left
// them at 0, using provider-specific multipliers inferred from the model name.
// Explicitly-set rates are preserved. Unknown providers keep 0 (transparent).
func deriveRates(model string, mp ModelPricing) ModelPricing {
	m := strings.ToLower(model)

	var cacheWrite, cacheRead float64
	switch {
	case strings.Contains(m, "claude"): // incl. anthropic.* and us.anthropic.* on Bedrock
		cacheWrite = mp.Input * anthropicCacheWriteMult
		cacheRead = mp.Input * anthropicCacheReadMult
	case strings.Contains(m, "gemini"):
		cacheWrite = mp.Input // no separate write surcharge
		cacheRead = mp.Input * geminiCacheReadMult
	case strings.HasPrefix(m, "gpt"), strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"), strings.HasPrefix(m, "o4"),
		strings.HasPrefix(m, "chatgpt"):
		cacheWrite = mp.Input // no separate write surcharge
		cacheRead = mp.Input * openaiCacheReadMult
	default:
		// Unknown provider: leave derived rates at 0.
	}

	if mp.CacheWrite == 0 {
		mp.CacheWrite = cacheWrite
	}
	if mp.CacheRead == 0 {
		mp.CacheRead = cacheRead
	}
	if mp.Reasoning == 0 {
		mp.Reasoning = mp.Output // reasoning tokens bill as output by default
	}
	return mp
}

// findPricing looks up pricing by exact match first, then by prefix match
// This handles versioned model names like "anthropic.claude-opus-4-5-20251101-v1:0"
// matching the base "anthropic.claude-opus-4-5". It resolves against the effective
// table (external source overlaid on the local map; see pricing_source.go), and
// the returned pricing has its cache/reasoning rates resolved via deriveRates.
func findPricing(model string) (ModelPricing, bool) {
	table := currentPricingTable()

	// Try exact match first
	if mp, ok := table[model]; ok {
		return deriveRates(model, mp), true
	}

	// Try prefix match (longest match wins)
	var bestMatch string
	var bestPricing ModelPricing
	for key, mp := range table {
		if strings.HasPrefix(model, key) && len(key) > len(bestMatch) {
			bestMatch = key
			bestPricing = mp
		}
	}

	if bestMatch != "" {
		return deriveRates(model, bestPricing), true
	}

	// No pricing found — track it for observability (logs once + on threshold).
	recordUnknownModel(model)
	return defaultPricing, false
}

// PricingCalculator calculates costs for LLM calls
type PricingCalculator struct{}

// NewPricingCalculator creates a new pricing calculator
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// TokenUsage holds the token counts for a single model call, split by billing
// category. The categories are assumed DISJOINT (non-overlapping): when a
// provider reports cached tokens as a subset of input, or reasoning tokens as a
// subset of output, the caller must subtract them first so each token is counted
// exactly once. That per-provider normalization lives in the ingest layer.
type TokenUsage struct {
	Input      int
	Output     int
	CacheRead  int
	CacheWrite int
	Reasoning  int
}

// CostBreakdown is the per-category cost decomposition of a model call, in USD.
type CostBreakdown struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Reasoning  float64 `json:"reasoning"`
	Total      float64 `json:"total"`
}

// round6 rounds a USD amount to 6 decimal places.
func round6(v float64) float64 {
	return math.Round(v*1000000) / 1000000
}

// CalculateCostBreakdown prices each (disjoint) token bucket at its own rate and
// returns the per-category decomposition plus the total. The total is rounded
// from the un-rounded components so it matches the legacy input+output result.
func (p *PricingCalculator) CalculateCostBreakdown(model string, usage TokenUsage) CostBreakdown {
	mp, _ := findPricing(model)

	inputCost := (float64(usage.Input) / 1000) * mp.Input
	outputCost := (float64(usage.Output) / 1000) * mp.Output
	cacheReadCost := (float64(usage.CacheRead) / 1000) * mp.CacheRead
	cacheWriteCost := (float64(usage.CacheWrite) / 1000) * mp.CacheWrite
	reasoningCost := (float64(usage.Reasoning) / 1000) * mp.Reasoning

	return CostBreakdown{
		Input:      round6(inputCost),
		Output:     round6(outputCost),
		CacheRead:  round6(cacheReadCost),
		CacheWrite: round6(cacheWriteCost),
		Reasoning:  round6(reasoningCost),
		Total:      round6(inputCost + outputCost + cacheReadCost + cacheWriteCost + reasoningCost),
	}
}

// CalculateCost calculates the total cost for a model call in USD. It is a
// backward-compatible wrapper over CalculateCostBreakdown (input + output only).
func (p *PricingCalculator) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	return p.CalculateCostBreakdown(model, TokenUsage{Input: inputTokens, Output: outputTokens}).Total
}

// NormalizeTokenUsage converts provider-reported token counts into the disjoint
// buckets that CalculateCostBreakdown expects. Providers differ in whether their
// cache/reasoning counts overlap with input/output (see parser.go):
//
//   - Anthropic/Bedrock: input_tokens already EXCLUDES cache tokens, and there is
//     no separate reasoning count (thinking is billed within output). Disjoint.
//   - OpenAI/OpenRouter: completion_tokens INCLUDES reasoning_tokens, and
//     prompt_tokens INCLUDES cached tokens. Subtract both so nothing is double-counted.
//   - Gemini: promptTokenCount INCLUDES cachedContentTokenCount (subtract);
//     candidatesTokenCount is disjoint from thoughtsTokenCount.
//   - Unknown: assume the common (OpenAI-style) overlap and subtract subsets.
func NormalizeTokenUsage(provider string, input, output, cacheRead, cacheWrite, reasoning int) TokenUsage {
	u := TokenUsage{
		Input:      input,
		Output:     output,
		CacheRead:  cacheRead,
		CacheWrite: cacheWrite,
		Reasoning:  reasoning,
	}

	switch strings.ToLower(provider) {
	case "anthropic", "bedrock":
		// Already disjoint; nothing to subtract.
	case "gemini":
		u.Input = max(0, input-cacheRead)
	default: // openai, openrouter, unknown
		u.Input = max(0, input-cacheRead)
		u.Output = max(0, output-reasoning)
	}

	return u
}

// GetModelPricing returns the pricing for a model
func (p *PricingCalculator) GetModelPricing(model string) ModelPricing {
	mp, _ := findPricing(model)
	return mp
}

// CalculateCost is a convenience function
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	return NewPricingCalculator().CalculateCost(model, inputTokens, outputTokens)
}
