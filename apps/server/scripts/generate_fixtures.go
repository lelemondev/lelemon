//go:build ignore

// Script to generate REAL API response fixtures for testing.
// These are COMPLEX, production-like scenarios, not simple calls.
//
// Run with: go run scripts/generate_fixtures.go
//
// Required environment variables:
//   ANTHROPIC_API_KEY - Anthropic API key
//   OPENAI_API_KEY    - OpenAI API key
//   GOOGLE_API_KEY    - Google Gemini API key
//
// Output: pkg/domain/service/testdata/fixtures/real_*.json

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const fixturesDir = "pkg/domain/service/testdata/fixtures"

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       LELEMON FIXTURE GENERATOR - Production Scenarios     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Anthropic
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		fmt.Println("🔮 ANTHROPIC")
		fmt.Println("─────────────────────────────────────────────")
		generateAnthropicFixtures(key)
	} else {
		fmt.Println("⏭️  Skipping Anthropic (ANTHROPIC_API_KEY not set)")
	}

	// OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		fmt.Println("\n🤖 OPENAI")
		fmt.Println("─────────────────────────────────────────────")
		generateOpenAIFixtures(key)
	} else {
		fmt.Println("\n⏭️  Skipping OpenAI (OPENAI_API_KEY not set)")
	}

	// Gemini
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		fmt.Println("\n💎 GEMINI")
		fmt.Println("─────────────────────────────────────────────")
		generateGeminiFixtures(key)
	} else {
		fmt.Println("\n⏭️  Skipping Gemini (GOOGLE_API_KEY not set)")
	}

	fmt.Println("\n════════════════════════════════════════════════════════════")
	fmt.Println("✅ Done! Fixtures saved to:", fixturesDir)
}

// ============================================================================
// ANTHROPIC - Complex production scenarios
// ============================================================================

func generateAnthropicFixtures(apiKey string) {
	baseURL := "https://api.anthropic.com/v1/messages"

	// Scenario 1: Multi-turn conversation with long system prompt (simulates agent)
	fmt.Println("  📝 Scenario: Multi-turn sales conversation...")
	generateAnthropicMultiTurn(apiKey, baseURL)

	// Scenario 2: Tool use with multiple tools (simulates agentic workflow)
	fmt.Println("  🔧 Scenario: Multi-tool agentic workflow...")
	generateAnthropicMultiTool(apiKey, baseURL)

	// Scenario 3: Complex analysis task (generates thinking-like behavior)
	fmt.Println("  🧠 Scenario: Complex analysis task...")
	generateAnthropicComplexAnalysis(apiKey, baseURL)
}

func generateAnthropicMultiTurn(apiKey, baseURL string) {
	// Simulates a real sales agent conversation with multiple turns
	request := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1024,
		"system": `You are an AI sales assistant for TechCorp, a B2B SaaS company.
Your role is to:
1. Understand customer needs through discovery questions
2. Present relevant product features
3. Handle objections professionally
4. Guide towards next steps

Product: TechCorp Analytics Platform
- Price: $499/month (Starter), $999/month (Professional), $2499/month (Enterprise)
- Features: Real-time dashboards, API access, team collaboration, custom reports
- Integration: Salesforce, HubSpot, Slack, Zapier

Always be helpful, professional, and focused on customer value.`,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": "Hi, I'm looking for an analytics solution for my team of 25 people. We currently use spreadsheets but it's becoming unmanageable.",
			},
			{
				"role":    "assistant",
				"content": "Hello! Thanks for reaching out. I'd be happy to help you find the right analytics solution.\n\nManaging analytics in spreadsheets for a team of 25 can definitely become challenging - I hear this often from growing teams.\n\nTo recommend the best fit, could you tell me:\n1. What type of data are you primarily analyzing? (sales, marketing, operations, etc.)\n2. How often does your team need to access these reports?\n3. Do you currently use any other business tools like Salesforce or HubSpot?",
			},
			{
				"role":    "user",
				"content": "We're mainly tracking sales pipeline and marketing campaign performance. The team needs daily access. We use HubSpot for CRM.",
			},
			{
				"role":    "assistant",
				"content": "Perfect, that's really helpful context!\n\nGiven your needs - sales pipeline tracking, marketing analytics, and HubSpot integration - our Professional plan would be an excellent fit. Here's why:\n\n**Real-time Dashboards**: Your team gets live views of pipeline health and campaign performance, updated automatically.\n\n**HubSpot Integration**: Native two-way sync means your CRM data flows directly into analytics without manual exports.\n\n**Team Collaboration**: With 25 users, you'll benefit from shared dashboards, commenting, and scheduled report delivery.\n\nThe Professional plan at $999/month includes:\n- Up to 50 users\n- All integrations (including HubSpot)\n- Custom report builder\n- Priority support\n\nWould you like to see a demo tailored to your sales and marketing use cases?",
			},
			{
				"role":    "user",
				"content": "That sounds interesting but $999/month is above our budget. We were thinking more around $500. Also, we'd need to get buy-in from our CFO.",
			},
		},
	}

	response, err := callAnthropic(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_anthropic_multi_turn.json", map[string]any{
		"_description": "Anthropic - Multi-turn sales conversation (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "claude-3-5-sonnet-20241022",
		"_scenario":    "Sales agent handling objection and budget concerns",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_anthropic_multi_turn.json")
}

func generateAnthropicMultiTool(apiKey, baseURL string) {
	// Simulates an agentic workflow with multiple tools
	request := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1024,
		"system":     "You are a helpful AI assistant with access to various tools. Use them when needed to help users.",
		"tools": []map[string]any{
			{
				"name":        "search_database",
				"description": "Search the company database for customer information, orders, or products",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Search query",
						},
						"table": map[string]any{
							"type":        "string",
							"enum":        []string{"customers", "orders", "products"},
							"description": "Which table to search",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Max results to return",
						},
					},
					"required": []string{"query", "table"},
				},
			},
			{
				"name":        "send_email",
				"description": "Send an email to a customer or team member",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"to": map[string]any{
							"type":        "string",
							"description": "Recipient email address",
						},
						"subject": map[string]any{
							"type":        "string",
							"description": "Email subject",
						},
						"body": map[string]any{
							"type":        "string",
							"description": "Email body content",
						},
						"priority": map[string]any{
							"type":        "string",
							"enum":        []string{"low", "normal", "high"},
							"description": "Email priority",
						},
					},
					"required": []string{"to", "subject", "body"},
				},
			},
			{
				"name":        "create_task",
				"description": "Create a task in the project management system",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title": map[string]any{
							"type":        "string",
							"description": "Task title",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "Task description",
						},
						"assignee": map[string]any{
							"type":        "string",
							"description": "Person to assign the task to",
						},
						"due_date": map[string]any{
							"type":        "string",
							"description": "Due date in YYYY-MM-DD format",
						},
						"priority": map[string]any{
							"type":        "string",
							"enum":        []string{"low", "medium", "high", "urgent"},
							"description": "Task priority",
						},
					},
					"required": []string{"title", "assignee"},
				},
			},
			{
				"name":        "get_calendar",
				"description": "Get calendar availability for scheduling",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"user": map[string]any{
							"type":        "string",
							"description": "User to check calendar for",
						},
						"date_range": map[string]any{
							"type":        "string",
							"description": "Date range to check (e.g., 'next week', '2024-01-15 to 2024-01-20')",
						},
					},
					"required": []string{"user", "date_range"},
				},
			},
		},
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": "I need to follow up with our top customer Acme Corp about their recent order. Find their info, check when Sarah (their account manager) is free next week, and draft a follow-up email. Also create a task for Sarah to call them.",
			},
		},
	}

	response, err := callAnthropic(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_anthropic_multi_tool.json", map[string]any{
		"_description": "Anthropic - Multi-tool agentic workflow (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "claude-3-5-sonnet-20241022",
		"_scenario":    "Agent using multiple tools to complete complex task",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_anthropic_multi_tool.json")
}

func generateAnthropicComplexAnalysis(apiKey, baseURL string) {
	// Complex analysis that requires step-by-step reasoning
	request := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 2048,
		"system":     "You are a senior data analyst. Provide thorough, structured analysis with clear reasoning.",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": `Analyze this quarterly sales data and provide insights:

Q1 2024:
- North Region: $2.4M (up 15% YoY), 340 deals, avg deal size $7,058
- South Region: $1.8M (down 5% YoY), 290 deals, avg deal size $6,207
- East Region: $3.1M (up 28% YoY), 380 deals, avg deal size $8,158
- West Region: $2.7M (up 8% YoY), 310 deals, avg deal size $8,710

Additional context:
- New product launch in East region in January
- South region lost 2 senior sales reps in December
- West region implemented new CRM system in February
- Overall market grew 12% YoY

Please provide:
1. Key insights from the data
2. Root cause analysis for underperforming regions
3. Recommendations for Q2
4. Risk factors to monitor`,
			},
		},
	}

	response, err := callAnthropic(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_anthropic_analysis.json", map[string]any{
		"_description": "Anthropic - Complex data analysis (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "claude-3-5-sonnet-20241022",
		"_scenario":    "Complex business analysis requiring structured reasoning",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_anthropic_analysis.json")
}

func callAnthropic(apiKey, url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

// ============================================================================
// OPENAI - Complex production scenarios
// ============================================================================

func generateOpenAIFixtures(apiKey string) {
	baseURL := "https://api.openai.com/v1/chat/completions"

	// Scenario 1: Multi-turn with system prompt
	fmt.Println("  📝 Scenario: Multi-turn customer support...")
	generateOpenAIMultiTurn(apiKey, baseURL)

	// Scenario 2: Function calling with parallel tool use
	fmt.Println("  🔧 Scenario: Parallel function calling...")
	generateOpenAIParallelTools(apiKey, baseURL)

	// Scenario 3: JSON mode structured output
	fmt.Println("  📊 Scenario: Structured JSON output...")
	generateOpenAIStructuredOutput(apiKey, baseURL)
}

func generateOpenAIMultiTurn(apiKey, baseURL string) {
	request := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{
				"role": "system",
				"content": `You are a senior customer support agent for CloudStore, an e-commerce platform.
Guidelines:
- Be empathetic and professional
- Offer concrete solutions
- Escalate complex issues appropriately
- Always confirm customer satisfaction before closing`,
			},
			{
				"role":    "user",
				"content": "I ordered a laptop 2 weeks ago and it still hasn't arrived. Order #CS-2024-78234. This is unacceptable!",
			},
			{
				"role":    "assistant",
				"content": "I completely understand your frustration, and I sincerely apologize for this delay. Two weeks is definitely longer than expected.\n\nLet me look up your order #CS-2024-78234 right away.\n\nI can see your order was for a ProBook X15 laptop, shipped on January 5th via Express Shipping. According to the tracking, it appears the package was held at a regional distribution center due to an address verification issue.\n\nHere's what I can do immediately:\n1. Contact the carrier directly to expedite delivery\n2. Provide you with a direct tracking link with real-time updates\n3. Apply a 15% discount to your order as compensation for this inconvenience\n\nWould you like me to proceed with all three options?",
			},
			{
				"role":    "user",
				"content": "Yes please. But what's this address verification issue? I've ordered here before with the same address.",
			},
		},
		"max_tokens": 800,
		"temperature": 0.7,
	}

	response, err := callOpenAI(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_openai_multi_turn.json", map[string]any{
		"_description": "OpenAI - Multi-turn customer support (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "gpt-4o",
		"_scenario":    "Customer support agent handling escalated complaint",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_openai_multi_turn.json")
}

func generateOpenAIParallelTools(apiKey, baseURL string) {
	request := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{
				"role": "system",
				"content": "You are an executive assistant AI. Use tools efficiently, making parallel calls when possible.",
			},
			{
				"role": "user",
				"content": "Prepare for my meeting with Acme Corp tomorrow. I need: 1) Their company profile, 2) Our last 3 interactions with them, 3) Book a conference room, and 4) Send a calendar invite to the attendees: john@acme.com and sarah@ourcompany.com",
			},
		},
		"tools": []map[string]any{
			{
				"type": "function",
				"function": map[string]any{
					"name":        "lookup_company",
					"description": "Get company profile and information",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"company_name": map[string]any{"type": "string"},
							"include":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						},
						"required": []string{"company_name"},
					},
				},
			},
			{
				"type": "function",
				"function": map[string]any{
					"name":        "get_interactions",
					"description": "Get interaction history with a company",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"company_name": map[string]any{"type": "string"},
							"limit":        map[string]any{"type": "integer"},
							"type":         map[string]any{"type": "string", "enum": []string{"all", "meetings", "emails", "calls"}},
						},
						"required": []string{"company_name"},
					},
				},
			},
			{
				"type": "function",
				"function": map[string]any{
					"name":        "book_room",
					"description": "Book a conference room",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"date":       map[string]any{"type": "string"},
							"time":       map[string]any{"type": "string"},
							"duration":   map[string]any{"type": "integer", "description": "Duration in minutes"},
							"room_size":  map[string]any{"type": "string", "enum": []string{"small", "medium", "large"}},
							"equipment":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						},
						"required": []string{"date", "time", "duration"},
					},
				},
			},
			{
				"type": "function",
				"function": map[string]any{
					"name":        "send_calendar_invite",
					"description": "Send a calendar invitation",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"attendees":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							"title":       map[string]any{"type": "string"},
							"date":        map[string]any{"type": "string"},
							"time":        map[string]any{"type": "string"},
							"duration":    map[string]any{"type": "integer"},
							"description": map[string]any{"type": "string"},
							"location":    map[string]any{"type": "string"},
						},
						"required": []string{"attendees", "title", "date", "time"},
					},
				},
			},
		},
		"tool_choice": "auto",
		"max_tokens": 1000,
	}

	response, err := callOpenAI(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_openai_parallel_tools.json", map[string]any{
		"_description": "OpenAI - Parallel function calling (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "gpt-4o",
		"_scenario":    "Executive assistant using multiple tools in parallel",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_openai_parallel_tools.json")
}

func generateOpenAIStructuredOutput(apiKey, baseURL string) {
	request := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{
				"role": "system",
				"content": "You extract structured data from text. Output valid JSON only.",
			},
			{
				"role": "user",
				"content": `Extract entities from this email:

Subject: Re: Partnership Proposal - Q2 2024

Hi Michael,

Thanks for the detailed proposal. I've reviewed it with our team at Innovate Solutions (our CEO, Jennifer Walsh, was particularly impressed).

We'd like to move forward with Option B - the $150,000 annual contract with quarterly deliverables. Our legal team (contact: david.chen@innovatesolutions.com) will send over the MSA by Friday.

Can we schedule a kickoff call for the week of January 22nd? I'm available Monday or Wednesday afternoon.

Best regards,
Amanda Foster
VP of Partnerships
Innovate Solutions
amanda.foster@innovatesolutions.com
+1 (555) 234-5678`,
			},
		},
		"response_format": map[string]any{"type": "json_object"},
		"max_tokens": 800,
	}

	response, err := callOpenAI(apiKey, baseURL, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_openai_structured.json", map[string]any{
		"_description": "OpenAI - Structured JSON output (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "gpt-4o",
		"_scenario":    "Entity extraction with JSON response format",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_openai_structured.json")
}

func callOpenAI(apiKey, url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

// ============================================================================
// GEMINI - Complex production scenarios
// ============================================================================

func generateGeminiFixtures(apiKey string) {
	// Scenario 1: Complex multi-modal analysis (text only for now)
	fmt.Println("  📊 Scenario: Complex data analysis...")
	generateGeminiAnalysis(apiKey)

	// Scenario 2: Function calling
	fmt.Println("  🔧 Scenario: Function calling...")
	generateGeminiFunctionCalling(apiKey)
}

func generateGeminiAnalysis(apiKey string) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro:generateContent?key=%s", apiKey)

	request := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"text": `Analyze this startup pitch and provide structured feedback:

Company: DataFlow AI
Stage: Series A ($5M raised)
Sector: B2B SaaS - Data Integration

Problem: Enterprises spend 40% of data engineering time on ETL pipelines
Solution: AI-powered no-code data integration platform
Market Size: $15B TAM, growing 25% annually

Traction:
- $1.2M ARR (300% YoY growth)
- 45 enterprise customers
- 92% gross retention
- NPS: 72

Team:
- CEO: Ex-Google engineer, 10 years ML experience
- CTO: PhD Stanford, published 20 papers
- VP Sales: Former Snowflake, built $50M book

Ask: $15M Series A at $60M pre-money

Provide:
1. Strengths (bulleted)
2. Concerns/Red flags (bulleted)
3. Questions for due diligence
4. Comparable companies
5. Investment recommendation (1-10 scale with reasoning)`,
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 2048,
			"temperature":     0.7,
		},
	}

	response, err := callGemini(url, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_gemini_analysis.json", map[string]any{
		"_description": "Gemini - Complex startup analysis (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "gemini-1.5-pro",
		"_scenario":    "VC-style startup pitch analysis",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_gemini_analysis.json")
}

func generateGeminiFunctionCalling(apiKey string) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro:generateContent?key=%s", apiKey)

	request := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{
						"text": "I need to book a flight from San Francisco to New York for next Monday, find a hotel near Times Square for 3 nights, and rent a car for the same period.",
					},
				},
			},
		},
		"tools": []map[string]any{
			{
				"functionDeclarations": []map[string]any{
					{
						"name":        "search_flights",
						"description": "Search for available flights",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"origin":         map[string]any{"type": "string"},
								"destination":    map[string]any{"type": "string"},
								"departure_date": map[string]any{"type": "string"},
								"return_date":    map[string]any{"type": "string"},
								"passengers":     map[string]any{"type": "integer"},
								"class":          map[string]any{"type": "string"},
							},
							"required": []string{"origin", "destination", "departure_date"},
						},
					},
					{
						"name":        "search_hotels",
						"description": "Search for hotels in a location",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location":   map[string]any{"type": "string"},
								"check_in":   map[string]any{"type": "string"},
								"check_out":  map[string]any{"type": "string"},
								"guests":     map[string]any{"type": "integer"},
								"amenities":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							},
							"required": []string{"location", "check_in", "check_out"},
						},
					},
					{
						"name":        "search_car_rental",
						"description": "Search for car rentals",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"pickup_location": map[string]any{"type": "string"},
								"pickup_date":     map[string]any{"type": "string"},
								"return_date":     map[string]any{"type": "string"},
								"car_type":        map[string]any{"type": "string"},
							},
							"required": []string{"pickup_location", "pickup_date", "return_date"},
						},
					},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 1024,
		},
	}

	response, err := callGemini(url, request)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
		return
	}

	saveFixture("real_gemini_function_calling.json", map[string]any{
		"_description": "Gemini - Function calling (production scenario)",
		"_source":      "Live API call",
		"_captured":    time.Now().Format("2006-01-02T15:04:05Z"),
		"_model":       "gemini-1.5-pro",
		"_scenario":    "Travel booking with multiple function calls",
		"request":      request,
		"response":     response,
	})
	fmt.Println("    ✅ real_gemini_function_calling.json")
}

func callGemini(url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

// ============================================================================
// HELPERS
// ============================================================================

func saveFixture(filename string, data map[string]any) {
	path := filepath.Join(fixturesDir, filename)
	content, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, content, 0644)
}
