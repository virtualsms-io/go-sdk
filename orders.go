package virtualsms

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ─── Catalog (public, no auth) ────────────────────────────────────────────

// ListServices lists all SMS-verification services (Telegram, WhatsApp,
// etc). Public endpoint, no API key required.
func (c *VirtualSMS) ListServices(ctx context.Context) ([]Service, error) {
	var raw struct {
		Services []struct {
			ServiceID   string `json:"service_id"`
			Code        string `json:"code"`
			ServiceName string `json:"service_name"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"services"`
	}
	if err := c.do(ctx, "GET", "/customer/services", nil, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Service, 0, len(raw.Services))
	for _, s := range raw.Services {
		code := s.ServiceID
		if code == "" {
			code = s.Code
		}
		name := s.ServiceName
		if name == "" {
			name = s.Name
		}
		out = append(out, Service{Code: code, Name: name, Icon: s.Icon})
	}
	return out, nil
}

// ListCountries lists all available countries. Public endpoint, no API key
// required.
func (c *VirtualSMS) ListCountries(ctx context.Context) ([]Country, error) {
	var raw struct {
		Countries []struct {
			CountryID   string `json:"country_id"`
			ISO         string `json:"iso"`
			CountryName string `json:"country_name"`
			Name        string `json:"name"`
			Flag        string `json:"flag"`
		} `json:"countries"`
	}
	if err := c.do(ctx, "GET", "/customer/countries", nil, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Country, 0, len(raw.Countries))
	for _, cc := range raw.Countries {
		iso := cc.CountryID
		if iso == "" {
			iso = cc.ISO
		}
		name := cc.CountryName
		if name == "" {
			name = cc.Name
		}
		out = append(out, Country{ISO: iso, Name: name, Flag: cc.Flag})
	}
	return out, nil
}

// getCatalogCountries is the source of truth for real per-country stock
// (Count > 0 = in stock). Shared by GetPrice and FindCheapest.
func (c *VirtualSMS) getCatalogCountries(ctx context.Context, service string) ([]CatalogCountry, error) {
	q := url.Values{"service": {service}}
	var raw struct {
		Countries []struct {
			ID    string  `json:"id"`
			ISO   string  `json:"iso"`
			Name  string  `json:"name"`
			Price float64 `json:"price"`
			Count int     `json:"count"`
		} `json:"countries"`
	}
	if err := c.do(ctx, "GET", "/catalog/countries", q, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]CatalogCountry, 0, len(raw.Countries))
	for _, cc := range raw.Countries {
		iso := cc.ID
		if iso == "" {
			iso = cc.ISO
		}
		out = append(out, CatalogCountry{ISO: iso, Name: cc.Name, PriceUSD: cc.Price, Count: cc.Count})
	}
	return out, nil
}

// GetPrice checks the price and REAL stock for a service+country combo.
//
// /price alone returns no availability field, so this replicates the
// two-call fail-closed pattern used by the website and MCP server: it
// cross-checks the /catalog/countries per-country Count (Count > 0 = in
// stock) before ever reporting Available=true.
func (c *VirtualSMS) GetPrice(ctx context.Context, service, country string) (*Price, error) {
	q := url.Values{"service": {service}, "country": {country}}
	var raw struct {
		Price    float64 `json:"price"`
		PriceUSD float64 `json:"price_usd"`
		Currency string  `json:"currency"`
	}
	if err := c.do(ctx, "GET", "/price", q, nil, &raw); err != nil {
		if is404(err) {
			return &Price{Available: false}, nil
		}
		return nil, err
	}
	price := raw.Price
	if price == 0 && raw.PriceUSD != 0 {
		price = raw.PriceUSD
	}
	currency := raw.Currency
	if currency == "" {
		currency = "USD"
	}
	result := &Price{PriceUSD: price, Currency: currency, Available: false}

	catalog, err := c.getCatalogCountries(ctx, service)
	if err == nil {
		for _, row := range catalog {
			if strings.EqualFold(row.ISO, country) {
				result.Available = row.Count > 0
				break
			}
		}
	}
	// On catalog lookup failure, keep the fail-closed default (Available=false).
	return result, nil
}

// ─── Order lifecycle ────────────────────────────────────────────────────────

// CreateOrder buys a virtual number for one-off SMS verification.
func (c *VirtualSMS) CreateOrder(ctx context.Context, service, country string) (*Order, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]string{"service": service, "country": country}
	var order Order
	if err := c.do(ctx, "POST", "/customer/purchase", nil, body, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrder returns the full order detail including any received SMS.
func (c *VirtualSMS) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var order Order
	if err := c.do(ctx, "GET", "/customer/order/"+url.PathEscape(orderID), nil, nil, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// codeRE extracts the most likely numeric verification code from an SMS
// body: first 4-8 digit run wins (covers "SMS code: 666512", "Your code is
// 1234", etc.) — mirrors extractCode in tools.ts.
var codeRE = regexp.MustCompile(`\b(\d{4,8})\b`)

func extractCode(text string) string {
	if text == "" {
		return ""
	}
	m := codeRE.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}

// GetSMS is a thin, normalized wrapper over GetOrder: it merges
// messages[]/legacy sms_code/sms_text into one shape and extracts the
// numeric code.
func (c *VirtualSMS) GetSMS(ctx context.Context, orderID string) (*GetSMSResult, error) {
	order, err := c.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	messages := order.Messages
	if len(messages) == 0 && (order.SMSText != "" || order.SMSCode != "") {
		content := order.SMSText
		if content == "" {
			content = order.SMSCode
		}
		messages = []SmsMessage{{Content: content}}
	}
	var firstContent string
	if len(messages) > 0 {
		firstContent = messages[0].Content
	}
	code := order.SMSCode
	if code == "" && firstContent != "" {
		code = extractCode(firstContent)
	}
	return &GetSMSResult{
		Status:      order.Status,
		PhoneNumber: order.PhoneNumber,
		Messages:    messages,
		Code:        code,
		SMSCode:     code,
		SMSText:     firstContent,
	}, nil
}

// WaitForSMS blocks until an SMS arrives on orderID or timeoutSeconds
// elapses, polling GetOrder every 5 seconds. On timeout it RETURNS a
// {Success: false} result rather than erroring — the caller can retry
// later with GetSMS or cancel with CancelOrder.
//
// timeoutSeconds <= 0 uses the SDK default of 300s (5 minutes) — a
// deliberately generous default vs. the MCP tool's own 60s default, since
// SDK callers are usually a human/script blocking on this rather than an
// LLM agent loop.
//
// This is a client-side polling helper (no dedicated backend endpoint).
// The optional WebSocket race documented in the SDK spec is not
// implemented in this v2.0.0 baseline — polling-only, as the spec permits
// (WS can land as a v2.1 addition).
func (c *VirtualSMS) WaitForSMS(ctx context.Context, orderID string, timeoutSeconds int) (*WaitForSMSResult, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	const pollInterval = 5 * time.Second
	start := time.Now()
	deadline := start.Add(time.Duration(timeoutSeconds) * time.Second)

	initial, err := c.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("virtualsms: failed to load order %s: %w", orderID, err)
	}
	phoneNumber := initial.PhoneNumber

	buildSuccess := func(messages []SmsMessage) *WaitForSMSResult {
		var firstContent string
		if len(messages) > 0 {
			firstContent = messages[0].Content
		}
		return &WaitForSMSResult{
			Success:        true,
			OrderID:        orderID,
			PhoneNumber:    phoneNumber,
			Status:         "sms_received",
			Messages:       messages,
			Code:           extractCode(firstContent),
			DeliveryMethod: "polling",
			ElapsedSeconds: int(time.Since(start).Seconds()),
		}
	}

	// Short-circuit: SMS already delivered before this call.
	if len(initial.Messages) > 0 {
		return buildSuccess(initial.Messages), nil
	}
	if initial.SMSCode != "" || initial.SMSText != "" {
		content := initial.SMSText
		if content == "" {
			content = initial.SMSCode
		}
		return buildSuccess([]SmsMessage{{Content: content}}), nil
	}

	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		wait := pollInterval
		if remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			break
		}
		if !sleepOrDone(ctx, wait) {
			return nil, ctx.Err()
		}

		status, err := c.GetOrder(ctx, orderID)
		if err != nil {
			// Transient lookup failures don't abort the poll loop; a
			// permanent failure will keep erroring until deadline.
			continue
		}
		if len(status.Messages) > 0 {
			return buildSuccess(status.Messages), nil
		}
		if status.SMSCode != "" || status.SMSText != "" {
			content := status.SMSText
			if content == "" {
				content = status.SMSCode
			}
			return buildSuccess([]SmsMessage{{Content: content}}), nil
		}
		if status.Status == "cancelled" || status.Status == "failed" {
			return nil, fmt.Errorf("virtualsms: order %s was %s before SMS arrived", orderID, status.Status)
		}
	}

	return &WaitForSMSResult{
		Success:     false,
		Error:       "timeout",
		OrderID:     orderID,
		PhoneNumber: phoneNumber,
	}, nil
}

// preCheckCooldown returns a non-nil error if availableAt is a valid
// RFC3339 timestamp still in the future — saves a round-trip on the
// typical "caller fires immediately after purchase" pattern. Returns nil
// (proceed, let the backend enforce it) if availableAt is empty/unparseable.
func preCheckCooldown(availableAt, action string) error {
	if availableAt == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, availableAt)
	if err != nil {
		return nil
	}
	if !time.Now().Before(t) {
		return nil
	}
	wait := time.Until(t).Round(time.Second)
	return fmt.Errorf("virtualsms: %s cooldown active, try again in %s (retry_at=%s)", action, wait, availableAt)
}

// CancelOrder cancels and refunds an order (before any SMS received).
// Pre-checks the order's CancelAvailableAt (120s post-purchase cooldown)
// client-side to save a round-trip; the backend enforces the cooldown
// regardless if the pre-check is skipped (lookup failure).
func (c *VirtualSMS) CancelOrder(ctx context.Context, orderID string) (*CancelResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	if order, err := c.GetOrder(ctx, orderID); err == nil {
		if cdErr := preCheckCooldown(order.CancelAvailableAt, "cancel"); cdErr != nil {
			return nil, cdErr
		}
	}
	var result CancelResult
	if err := c.do(ctx, "POST", "/customer/cancel/"+url.PathEscape(orderID), nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SwapNumber gets a new number for the same service/country at no extra
// charge. Pre-checks SwapAvailableAt the same way CancelOrder does.
func (c *VirtualSMS) SwapNumber(ctx context.Context, orderID string) (*Order, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	if order, err := c.GetOrder(ctx, orderID); err == nil {
		if cdErr := preCheckCooldown(order.SwapAvailableAt, "swap"); cdErr != nil {
			return nil, cdErr
		}
	}
	var order Order
	if err := c.do(ctx, "POST", "/customer/swap/"+url.PathEscape(orderID), nil, nil, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// RetryOrder asks the provider to resend the SMS to the SAME number (not a
// new number — see SwapNumber for that).
func (c *VirtualSMS) RetryOrder(ctx context.Context, orderID string) (*RetryOrderResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result RetryOrderResult
	if err := c.do(ctx, "POST", "/orders/"+url.PathEscape(orderID)+"/retry", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListOrders lists orders, optionally filtered by status. A 404 from the
// backend (endpoint may not exist on older deployments) is swallowed to an
// empty slice rather than raised.
func (c *VirtualSMS) ListOrders(ctx context.Context, status string) ([]Order, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var q url.Values
	if status != "" {
		q = url.Values{"status": {status}}
	}
	var raw []Order
	err := c.do(ctx, "GET", "/customer/orders", q, nil, &raw)
	if err != nil {
		if is404(err) {
			return []Order{}, nil
		}
		return nil, err
	}
	return raw, nil
}

func parseOrderTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// OrderHistory returns order history with client-side filtering
// (service/country/since_days) plus a hard result cap. No dedicated
// backend route — calls ListOrders then filters/caps locally.
func (c *VirtualSMS) OrderHistory(ctx context.Context, params OrderHistoryParams) (*OrderHistoryResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	orders, err := c.ListOrders(ctx, params.Status)
	if err != nil {
		return nil, err
	}

	var cutoff time.Time
	hasCutoff := params.SinceDays > 0
	if hasCutoff {
		cutoff = time.Now().Add(-time.Duration(params.SinceDays) * 24 * time.Hour)
	}
	serviceFilter := strings.ToLower(params.Service)
	countryFilter := strings.ToUpper(params.Country)

	filtered := make([]Order, 0, len(orders))
	for _, o := range orders {
		if hasCutoff {
			t, ok := parseOrderTime(o.CreatedAt)
			if !ok || t.Before(cutoff) {
				continue
			}
		}
		if serviceFilter != "" && strings.ToLower(o.Service) != serviceFilter {
			continue
		}
		if countryFilter != "" && strings.ToUpper(o.Country) != countryFilter {
			continue
		}
		filtered = append(filtered, o)
	}

	capped := filtered
	if len(capped) > limit {
		capped = capped[:limit]
	}

	return &OrderHistoryResult{
		Count:        len(capped),
		TotalMatched: len(filtered),
		Filters:      params,
		Orders:       capped,
	}, nil
}

// activeOrderStatuses are considered live/billable/cancellable.
var activeOrderStatuses = map[string]bool{
	"waiting": true, "pending": true, "sms_received": true, "created": true,
}

// CancelAllOrders bulk-cancels every active order. Uses
// gather-with-partial-failure semantics (never abort-on-first-error): one
// failed cancellation does not stop the others.
func (c *VirtualSMS) CancelAllOrders(ctx context.Context) (*CancelAllOrdersResult, error) {
	orders, err := c.ListOrders(ctx, "")
	if err != nil {
		return nil, err
	}
	active := make([]Order, 0, len(orders))
	for _, o := range orders {
		if activeOrderStatuses[o.Status] {
			active = append(active, o)
		}
	}
	result := &CancelAllOrdersResult{TotalActive: len(active)}
	if len(active) == 0 {
		return result, nil
	}

	for _, o := range active {
		res, cerr := c.CancelOrder(ctx, o.OrderID)
		if cerr != nil {
			result.Failures = append(result.Failures, CancelOrderFailureEntry{OrderID: o.OrderID, Error: cerr.Error()})
			continue
		}
		result.CancelledOrders = append(result.CancelledOrders, CancelledOrderEntry{OrderID: o.OrderID, Refunded: res.Refunded})
	}
	result.Cancelled = len(result.CancelledOrders)
	result.Failed = len(result.Failures)
	return result, nil
}

// SearchServices finds the right service code via natural language
// ("uber", "binance", "steam"). Client-side fuzzy match over
// ListServices() — no dedicated backend search route. Scoring: exact
// code/name match = 1.0; prefix match = 0.9; substring match = 0.7; else
// token-overlap ratio capped at 0.6. Only matches scoring >= 0.5 are
// returned, top 5, sorted descending.
func (c *VirtualSMS) SearchServices(ctx context.Context, query string) (*SearchServicesResult, error) {
	services, err := c.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	queryTokens := strings.Fields(q)

	type scored struct {
		ServiceMatch
	}
	var candidates []scored
	for _, s := range services {
		name := strings.ToLower(s.Name)
		code := strings.ToLower(s.Code)

		var score float64
		switch {
		case code == q || name == q:
			score = 1.0
		case strings.HasPrefix(code, q) || strings.HasPrefix(name, q):
			score = 0.9
		case strings.Contains(code, q) || strings.Contains(name, q):
			score = 0.7
		default:
			nameTokens := splitTokens(name)
			matches := 0
			for _, qt := range queryTokens {
				for _, nt := range nameTokens {
					if strings.Contains(nt, qt) || strings.Contains(qt, nt) {
						matches++
						break
					}
				}
			}
			if matches > 0 {
				denom := len(queryTokens)
				if len(nameTokens) > denom {
					denom = len(nameTokens)
				}
				score = (float64(matches) / float64(denom)) * 0.6
			}
		}
		if score > 0 {
			candidates = append(candidates, scored{ServiceMatch{Code: s.Code, Name: s.Name, MatchScore: roundTo2dp(score)}})
		}
	}

	var matches []ServiceMatch
	for _, cnd := range candidates {
		if cnd.MatchScore >= 0.5 {
			matches = append(matches, cnd.ServiceMatch)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].MatchScore > matches[j].MatchScore })
	if len(matches) > 5 {
		matches = matches[:5]
	}

	if len(matches) == 0 {
		return &SearchServicesResult{
			Query:   query,
			Matches: []ServiceMatch{},
			Message: "No matching services found",
			Tip:     "Try ListServices to browse all available services.",
		}, nil
	}
	return &SearchServicesResult{
		Query:   query,
		Matches: matches,
		Tip:     `Use the "Code" field as the service parameter in other methods.`,
	}, nil
}

func splitTokens(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})
}

func roundTo2dp(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// FindCheapest finds the cheapest in-stock countries for a service, sorted
// ascending by price. limit <= 0 defaults to 5. Client-side: calls the
// same catalog source GetPrice uses for real stock (getCatalogCountries),
// NOT a fan-out over GetPrice per country (that endpoint has no stock
// field).
func (c *VirtualSMS) FindCheapest(ctx context.Context, service string, limit int) (*FindCheapestResult, error) {
	if limit <= 0 {
		limit = 5
	}
	catalog, err := c.getCatalogCountries(ctx, service)
	if err != nil {
		return nil, err
	}

	var results []CheapestOption
	for _, cc := range catalog {
		if cc.Count > 0 {
			results = append(results, CheapestOption{
				Country:     cc.ISO,
				CountryName: cc.Name,
				PriceUSD:    cc.PriceUSD,
				Stock:       true,
			})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].PriceUSD < results[j].PriceUSD })

	total := len(results)
	if len(results) > limit {
		results = results[:limit]
	}

	if len(results) == 0 {
		return &FindCheapestResult{
			Service:                 service,
			CheapestOptions:         []CheapestOption{},
			TotalAvailableCountries: 0,
			Message:                 fmt.Sprintf(`No countries available for service %q. Use SearchServices to verify the service code, or ListServices to see all available services.`, service),
		}, nil
	}
	return &FindCheapestResult{
		Service:                 service,
		CheapestOptions:         results,
		TotalAvailableCountries: total,
	}, nil
}
