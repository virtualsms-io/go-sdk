package virtualsms

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the production VirtualSMS REST v1 API root.
	DefaultBaseURL = "https://virtualsms.io/api/v1"

	// DefaultTimeout matches the MCP client's default (client.ts:
	// timeoutSeconds ?? 30).
	DefaultTimeout = 30 * time.Second

	// getRetryMaxAttempts is 1 initial try + up to 2 retries, matching
	// client.ts GET_RETRY_MAX_ATTEMPTS.
	getRetryMaxAttempts = 3
	getRetryBaseDelay   = 300 * time.Millisecond
)

// VirtualSMS is the REST v1 API client. Construct with New. Safe for
// concurrent use by multiple goroutines.
type VirtualSMS struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a VirtualSMS client constructed via New.
type Option func(*VirtualSMS)

// WithBaseURL overrides the API root (default DefaultBaseURL). Also
// settable via the VIRTUALSMS_BASE_URL environment variable convention
// documented in the SDK spec — this SDK does not read env vars itself;
// callers wire that up explicitly if desired:
//
//	virtualsms.New(key, virtualsms.WithBaseURL(os.Getenv("VIRTUALSMS_BASE_URL")))
func WithBaseURL(baseURL string) Option {
	return func(c *VirtualSMS) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithTimeout overrides the per-request HTTP timeout (default
// DefaultTimeout / 30s).
func WithTimeout(timeout time.Duration) Option {
	return func(c *VirtualSMS) {
		if timeout > 0 {
			c.httpClient.Timeout = timeout
		}
	}
}

// WithHTTPClient overrides the underlying *http.Client entirely (e.g. to
// inject a custom transport, proxy, or test double). Its Timeout is
// subsequently overridden by WithTimeout / DefaultTimeout unless this
// option is applied last.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *VirtualSMS) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// New constructs a VirtualSMS API client. apiKey is required for
// authenticated endpoints (orders, rentals, proxies, account, webhooks,
// sessions); public endpoints (ListServices, ListCountries, GetPrice,
// RentalsPricing, RentalsAvailable, ProxyCatalog, ListProxyLocations,
// CheckNumber) work with an empty apiKey too.
//
//	client := virtualsms.New(apiKey)
//	client := virtualsms.New(apiKey, virtualsms.WithBaseURL("https://staging.virtualsms.io/api/v1"))
func New(apiKey string, opts ...Option) *VirtualSMS {
	c := &VirtualSMS{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIKey returns the API key this client was constructed with (may be
// empty). Used internally by WaitForSMS to decide whether to attempt a
// websocket subscription.
func (c *VirtualSMS) APIKey() string { return c.apiKey }

// BaseURL returns the configured API root.
func (c *VirtualSMS) BaseURL() string { return c.baseURL }

func (c *VirtualSMS) requireAPIKey() error {
	if c.apiKey == "" {
		return fmt.Errorf("virtualsms: API key is required for this operation; get one at https://virtualsms.io")
	}
	return nil
}

// newIdempotencyKey generates a fresh UUID v4 for the X-Idempotency-Key
// header, mirroring the MCP client's randomUUID() interceptor. Sent on
// every mutating request (POST/PUT/PATCH/DELETE) unless the caller doesn't
// need it — this SDK always attaches one since it has no per-call override
// surface today.
func newIdempotencyKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is exceptionally rare; fall back to a
		// timestamp-derived key rather than panicking.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// shouldRetryGet mirrors client.ts shouldRetryGet: only GET/HEAD, capped at
// getRetryMaxAttempts total attempts, retries on no-response (network
// error) or any 5xx. Never retries 4xx (including 429 — fighting the
// server's own rate limiter is wrong).
func shouldRetryGet(method string, status int, hasResponse bool, attemptsSoFar int) bool {
	m := strings.ToLower(method)
	if m != "get" && m != "head" {
		return false
	}
	if attemptsSoFar >= getRetryMaxAttempts {
		return false
	}
	if !hasResponse {
		return true
	}
	return status >= 500
}

// getRetryDelay mirrors client.ts getRetryDelayMs: exponential backoff,
// 300ms * 2^(attemptNumber-1), attemptNumber is 1-indexed.
func getRetryDelay(attemptNumber int) time.Duration {
	return getRetryBaseDelay * time.Duration(1<<uint(attemptNumber-1))
}

// backendError is the shape VirtualSMS error responses take:
// {"error": "..."} or {"message": "..."}.
type backendError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// do executes a single API call, applying auth, idempotency keys, the
// GET-only bounded retry, and status-code error mapping. body is
// JSON-marshaled if non-nil; out is JSON-unmarshaled into if non-nil and
// the response has a body.
func (c *VirtualSMS) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("virtualsms: marshal request body: %w", err)
		}
		bodyBytes = b
	}

	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	mutating := isMutating(method)
	for attempt := 1; ; attempt++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			return fmt.Errorf("virtualsms: build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}
		if mutating {
			req.Header.Set("X-Idempotency-Key", newIdempotencyKey())
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if shouldRetryGet(method, 0, false, attempt) {
				if !sleepOrDone(ctx, getRetryDelay(attempt)) {
					return fmt.Errorf("virtualsms: request cancelled: %w", ctx.Err())
				}
				continue
			}
			return fmt.Errorf("virtualsms: request failed: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			if shouldRetryGet(method, resp.StatusCode, true, attempt) {
				if !sleepOrDone(ctx, getRetryDelay(attempt)) {
					return fmt.Errorf("virtualsms: request cancelled: %w", ctx.Err())
				}
				continue
			}
			return fmt.Errorf("virtualsms: read response body: %w", readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, out); err != nil {
					return fmt.Errorf("virtualsms: decode response: %w", err)
				}
			}
			return nil
		}

		if shouldRetryGet(method, resp.StatusCode, true, attempt) {
			if !sleepOrDone(ctx, getRetryDelay(attempt)) {
				return fmt.Errorf("virtualsms: request cancelled: %w", ctx.Err())
			}
			continue
		}

		return mapStatusError(resp.StatusCode, respBody, mutating)
	}
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// mapStatusError converts a non-2xx response into the SDK's typed error
// model per §"Error model" in the SDK spec. No-supplier-name rule applies
// here too: the raw backend message is surfaced as-is (it never contains a
// supplier name by contract), never anything synthesized server-side by
// this SDK.
func mapStatusError(status int, respBody []byte, mutating bool) error {
	message := extractBackendMessage(respBody)

	switch status {
	case http.StatusUnauthorized:
		return &APIError{StatusCode: status, Message: "invalid or missing API key; get one at https://virtualsms.io", sentinel: ErrBadKey}
	case http.StatusPaymentRequired:
		return &APIError{StatusCode: status, Message: "insufficient balance; top up at https://virtualsms.io", sentinel: ErrInsufficientBalance}
	case http.StatusNotFound:
		msg := message
		if msg == "" {
			msg = "resource not found"
		}
		return &APIError{StatusCode: status, Message: msg, sentinel: ErrNotFound}
	case http.StatusTooManyRequests:
		return &APIError{StatusCode: status, Message: "rate limit exceeded, please slow down requests", sentinel: ErrRateLimited}
	}

	if status >= 500 {
		lowerMessage := strings.ToLower(message)
		if strings.Contains(lowerMessage, "out of stock") || strings.Contains(lowerMessage, "no numbers") {
			return &APIError{StatusCode: status, Message: "no numbers currently available: " + message, sentinel: ErrNoNumbers}
		}
		return &ServerError{StatusCode: status, Message: message, Mutating: mutating}
	}

	return &APIError{StatusCode: status, Message: message}
}

func extractBackendMessage(respBody []byte) string {
	if len(respBody) == 0 {
		return ""
	}
	var be backendError
	if err := json.Unmarshal(respBody, &be); err == nil {
		if be.Message != "" {
			return be.Message
		}
		if be.Error != "" {
			return be.Error
		}
	}
	return string(respBody)
}

// is404 reports whether err is a "not found" APIError — used by ListOrders
// to swallow a 404 into an empty slice (the endpoint may not exist on
// older deployments; mirrors client.ts lines 792-799).
func is404(err error) bool {
	var apiErr *APIError
	return asAPIError(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func asAPIError(err error, target **APIError) bool {
	ae, ok := err.(*APIError)
	if ok {
		*target = ae
	}
	return ok
}
