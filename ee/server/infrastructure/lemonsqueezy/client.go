package lemonsqueezy

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.lemonsqueezy.com/v1"

// Client is the Lemon Squeezy API client
type Client struct {
	apiKey        string
	webhookSecret string
	httpClient    *http.Client
	storeID       string
}

// NewClient creates a new Lemon Squeezy client
func NewClient(apiKey, webhookSecret, storeID string) *Client {
	return &Client{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		storeID:       storeID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckoutRequest holds data for creating a checkout
type CheckoutRequest struct {
	Email      string
	VariantID  string
	CustomData map[string]string
}

// WebhookEvent represents an incoming webhook from Lemon Squeezy
type WebhookEvent struct {
	EventName          string            `json:"event_name"`
	SubscriptionID     string            `json:"subscription_id"`
	CustomerID         string            `json:"customer_id"`
	VariantID          string            `json:"variant_id"`
	Status             string            `json:"status"`
	CurrentPeriodStart time.Time         `json:"current_period_start"`
	CurrentPeriodEnd   time.Time         `json:"current_period_end"`
	CustomData         map[string]string `json:"custom_data"`
}

// CreateCheckout creates a checkout session and returns the URL
func (c *Client) CreateCheckout(ctx context.Context, req CheckoutRequest) (string, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"checkout_data": map[string]interface{}{
					"email":       req.Email,
					"custom":      req.CustomData,
				},
			},
			"relationships": map[string]interface{}{
				"store": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "stores",
						"id":   c.storeID,
					},
				},
				"variant": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "variants",
						"id":   req.VariantID,
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal checkout request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/checkouts", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/vnd.api+json")
	httpReq.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("checkout request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("checkout failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode checkout response: %w", err)
	}

	return result.Data.Attributes.URL, nil
}

// GetCustomerPortalURL returns the customer portal URL
func (c *Client) GetCustomerPortalURL(ctx context.Context, customerID string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/customers/%s", baseURL, customerID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("customer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("customer request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Attributes struct {
				URLs struct {
					CustomerPortal string `json:"customer_portal"`
				} `json:"urls"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode customer response: %w", err)
	}

	return result.Data.Attributes.URLs.CustomerPortal, nil
}

// CancelSubscription cancels a subscription
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/subscriptions/%s", baseURL, subscriptionID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("cancel request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cancel failed with status %d", resp.StatusCode)
	}

	return nil
}

// VerifyWebhookSignature verifies that a webhook came from Lemon Squeezy
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// ParseWebhookEvent parses a webhook payload into a WebhookEvent
func (c *Client) ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	var rawEvent struct {
		Meta struct {
			EventName  string            `json:"event_name"`
			CustomData map[string]string `json:"custom_data"`
		} `json:"meta"`
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Status             string    `json:"status"`
				CustomerID         int       `json:"customer_id"`
				VariantID          int       `json:"variant_id"`
				CurrentPeriodStart time.Time `json:"renews_at"`
				CurrentPeriodEnd   time.Time `json:"ends_at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	return &WebhookEvent{
		EventName:          rawEvent.Meta.EventName,
		SubscriptionID:     rawEvent.Data.ID,
		CustomerID:         fmt.Sprintf("%d", rawEvent.Data.Attributes.CustomerID),
		VariantID:          fmt.Sprintf("%d", rawEvent.Data.Attributes.VariantID),
		Status:             rawEvent.Data.Attributes.Status,
		CurrentPeriodStart: rawEvent.Data.Attributes.CurrentPeriodStart,
		CurrentPeriodEnd:   rawEvent.Data.Attributes.CurrentPeriodEnd,
		CustomData:         rawEvent.Meta.CustomData,
	}, nil
}
