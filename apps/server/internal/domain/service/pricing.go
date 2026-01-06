package service

import "math"

// ModelPricing contains pricing per 1K tokens
type ModelPricing struct {
	Input  float64 // per 1K tokens
	Output float64 // per 1K tokens
}

// pricing is the internal pricing table
var pricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4-turbo":            {Input: 0.01, Output: 0.03},
	"gpt-4-turbo-preview":    {Input: 0.01, Output: 0.03},
	"gpt-4o":                 {Input: 0.005, Output: 0.015},
	"gpt-4o-2024-05-13":      {Input: 0.005, Output: 0.015},
	"gpt-4o-mini":            {Input: 0.00015, Output: 0.0006},
	"gpt-4o-mini-2024-07-18": {Input: 0.00015, Output: 0.0006},
	"gpt-4":                  {Input: 0.03, Output: 0.06},
	"gpt-4-32k":              {Input: 0.06, Output: 0.12},
	"gpt-3.5-turbo":          {Input: 0.0005, Output: 0.0015},
	"gpt-3.5-turbo-16k":      {Input: 0.003, Output: 0.004},
	"o1-preview":             {Input: 0.015, Output: 0.06},
	"o1-mini":                {Input: 0.003, Output: 0.012},

	// Anthropic
	"claude-3-5-sonnet-20241022": {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-latest":   {Input: 0.003, Output: 0.015},
	"claude-3-opus-20240229":     {Input: 0.015, Output: 0.075},
	"claude-3-opus-latest":       {Input: 0.015, Output: 0.075},
	"claude-3-sonnet-20240229":   {Input: 0.003, Output: 0.015},
	"claude-3-haiku-20240307":    {Input: 0.00025, Output: 0.00125},

	// AWS Bedrock (Claude)
	"anthropic.claude-3-5-sonnet-20241022-v2:0": {Input: 0.003, Output: 0.015},
	"anthropic.claude-3-opus-20240229-v1:0":     {Input: 0.015, Output: 0.075},
	"anthropic.claude-3-sonnet-20240229-v1:0":   {Input: 0.003, Output: 0.015},
	"anthropic.claude-3-haiku-20240307-v1:0":    {Input: 0.00025, Output: 0.00125},

	// Google
	"gemini-1.5-pro":   {Input: 0.00125, Output: 0.005},
	"gemini-1.5-flash": {Input: 0.000075, Output: 0.0003},
	"gemini-2.0-flash": {Input: 0.0001, Output: 0.0004},
}

// defaultPricing is used for unknown models ($0 = transparent indicator)
var defaultPricing = ModelPricing{Input: 0, Output: 0}

// PricingCalculator calculates costs for LLM calls
type PricingCalculator struct{}

// NewPricingCalculator creates a new pricing calculator
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculateCost calculates the cost for a model call in USD
func (p *PricingCalculator) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	mp, ok := pricing[model]
	if !ok {
		mp = defaultPricing
	}

	inputCost := (float64(inputTokens) / 1000) * mp.Input
	outputCost := (float64(outputTokens) / 1000) * mp.Output

	// Round to 6 decimal places
	return math.Round((inputCost+outputCost)*1000000) / 1000000
}

// GetModelPricing returns the pricing for a model
func (p *PricingCalculator) GetModelPricing(model string) ModelPricing {
	if mp, ok := pricing[model]; ok {
		return mp
	}
	return defaultPricing
}

// CalculateCost is a convenience function
func CalculateCost(model string, inputTokens, outputTokens int) float64 {
	return NewPricingCalculator().CalculateCost(model, inputTokens, outputTokens)
}
