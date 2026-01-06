package service

import (
	"math"
	"strings"
)

// ModelPricing contains pricing per 1K tokens
type ModelPricing struct {
	Input  float64 // per 1K tokens
	Output float64 // per 1K tokens
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

// findPricing looks up pricing by exact match first, then by prefix match
// This handles versioned model names like "anthropic.claude-opus-4-5-20251101-v1:0"
// matching the base "anthropic.claude-opus-4-5"
func findPricing(model string) (ModelPricing, bool) {
	// Try exact match first
	if mp, ok := pricing[model]; ok {
		return mp, true
	}

	// Try prefix match (longest match wins)
	var bestMatch string
	var bestPricing ModelPricing
	for key, mp := range pricing {
		if strings.HasPrefix(model, key) && len(key) > len(bestMatch) {
			bestMatch = key
			bestPricing = mp
		}
	}

	if bestMatch != "" {
		return bestPricing, true
	}

	return defaultPricing, false
}

// PricingCalculator calculates costs for LLM calls
type PricingCalculator struct{}

// NewPricingCalculator creates a new pricing calculator
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculateCost calculates the cost for a model call in USD
func (p *PricingCalculator) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	mp, _ := findPricing(model)

	inputCost := (float64(inputTokens) / 1000) * mp.Input
	outputCost := (float64(outputTokens) / 1000) * mp.Output

	// Round to 6 decimal places
	return math.Round((inputCost+outputCost)*1000000) / 1000000
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
