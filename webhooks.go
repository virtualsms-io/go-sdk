package virtualsms

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Webhooks: base path /customer/webhooks. Auth is the same X-API-Key
// header as every other customer route (customer-webhooks.js's
// `authenticate` middleware falls through to the same api_keys sha256-hash
// lookup) — no special-case handling needed.

// ListWebhooks lists the account's webhook subscriptions.
func (c *VirtualSMS) ListWebhooks(ctx context.Context) (*ListWebhooksResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result ListWebhooksResult
	if err := c.do(ctx, "GET", "/customer/webhooks", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateWebhook creates a new webhook subscription. The response's
// Webhook.Secret is populated exactly once, on create — store it
// immediately, it is never returned again by GetWebhook/ListWebhooks.
func (c *VirtualSMS) CreateWebhook(ctx context.Context, params CreateWebhookParams) (*WebhookEndpoint, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(params.URL, "https://") {
		return nil, fmt.Errorf("virtualsms: webhook url must be https://")
	}
	if len(params.Events) == 0 {
		return nil, fmt.Errorf("virtualsms: events must be non-empty")
	}
	for _, e := range params.Events {
		if e == "balance.low" && params.Threshold <= 0 {
			return nil, fmt.Errorf(`virtualsms: threshold is required when events includes "balance.low"`)
		}
	}
	body := map[string]any{
		"url":    params.URL,
		"events": params.Events,
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if params.Threshold > 0 {
		body["threshold"] = params.Threshold
	}
	var raw struct {
		Success bool            `json:"success"`
		Webhook WebhookEndpoint `json:"webhook"`
	}
	if err := c.do(ctx, "POST", "/customer/webhooks", nil, body, &raw); err != nil {
		return nil, err
	}
	return &raw.Webhook, nil
}

// GetWebhook returns one webhook (no secret — the secret is only ever
// present in the CreateWebhook response).
func (c *VirtualSMS) GetWebhook(ctx context.Context, id string) (*WebhookEndpoint, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var raw GetWebhookResult
	if err := c.do(ctx, "GET", "/customer/webhooks/"+url.PathEscape(id), nil, nil, &raw); err != nil {
		return nil, err
	}
	return &raw.Webhook, nil
}

// UpdateWebhook partially updates a webhook (url/description/events/
// threshold/active/paused). At least one field must be set. Un-pausing
// (Paused: false when it was previously true) resets the server-side
// consecutive-failure counter to 0.
func (c *VirtualSMS) UpdateWebhook(ctx context.Context, id string, params UpdateWebhookParams) (*WebhookEndpoint, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if params.URL != nil {
		body["url"] = *params.URL
	}
	if params.Description != nil {
		body["description"] = *params.Description
	}
	if params.Events != nil {
		body["events"] = params.Events
	}
	if params.Threshold != nil {
		body["threshold"] = *params.Threshold
	}
	if params.Active != nil {
		body["active"] = *params.Active
	}
	if params.Paused != nil {
		body["paused"] = *params.Paused
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("virtualsms: at least one field is required to update a webhook")
	}
	var raw struct {
		Success bool            `json:"success"`
		Webhook WebhookEndpoint `json:"webhook"`
	}
	if err := c.do(ctx, "PATCH", "/customer/webhooks/"+url.PathEscape(id), nil, body, &raw); err != nil {
		return nil, err
	}
	return &raw.Webhook, nil
}

// DeleteWebhook deletes a webhook subscription.
func (c *VirtualSMS) DeleteWebhook(ctx context.Context, id string) (*DeleteWebhookResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result DeleteWebhookResult
	if err := c.do(ctx, "DELETE", "/customer/webhooks/"+url.PathEscape(id), nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TestWebhook fires a synthetic test event through the real dispatcher.
// Requires the webhook to be Active and not Paused, else the backend
// returns a 400.
func (c *VirtualSMS) TestWebhook(ctx context.Context, id string) (*TestWebhookResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result TestWebhookResult
	if err := c.do(ctx, "POST", "/customer/webhooks/"+url.PathEscape(id)+"/test", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListWebhookDeliveries lists recent delivery attempts for a webhook.
// params.Limit defaults to 100, max 500.
func (c *VirtualSMS) ListWebhookDeliveries(ctx context.Context, id string, params ListWebhookDeliveriesParams) (*ListWebhookDeliveriesResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	q := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(params.Offset)},
	}
	var result ListWebhookDeliveriesResult
	if err := c.do(ctx, "GET", "/customer/webhooks/"+url.PathEscape(id)+"/deliveries", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
