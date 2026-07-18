package virtualsms

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Fixed gateway ports. Rotating vs. sticky is encoded entirely in the
// username's sessid/sessttl params, NOT by port selection. Mirrors
// frontend/src/components/my-numbers-v2/ProxyEndpointGenerator.tsx.
const (
	proxyHTTPPort   = 823
	proxySOCKS5Port = 824
)

// ListProxyCatalog lists proxy pool types, countries, and price/GB. Public
// endpoint, ~10min server-side cache.
func (c *VirtualSMS) ListProxyCatalog(ctx context.Context) ([]ProxyCatalogPoolType, error) {
	var raw struct {
		PoolTypes []ProxyCatalogPoolType `json:"pool_types"`
	}
	if err := c.do(ctx, "GET", "/proxies/catalog", nil, nil, &raw); err != nil {
		return nil, err
	}
	return raw.PoolTypes, nil
}

// ListProxies lists owned proxies with their live credentials.
func (c *VirtualSMS) ListProxies(ctx context.Context) ([]ProxyListItem, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var raw []ProxyListItem
	if err := c.do(ctx, "GET", "/proxies", nil, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// BuyProxy purchases proxy traffic (GB) for a pool type.
func (c *VirtualSMS) BuyProxy(ctx context.Context, params BuyProxyParams) (*ProxyPurchaseResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	if params.GB <= 0 {
		return nil, fmt.Errorf("virtualsms: gb must be positive")
	}
	body := map[string]any{
		"pool_type": params.PoolType,
		"gb":        params.GB,
	}
	if params.CountryCode != "" {
		body["country_code"] = params.CountryCode
	}
	if params.IdempotencyKey != "" {
		body["idempotency_key"] = params.IdempotencyKey
	}
	var result ProxyPurchaseResult
	if err := c.do(ctx, "POST", "/proxies", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RotateProxy gets a fresh exit IP for an existing proxy. port <= 0 omits
// the field (server picks).
func (c *VirtualSMS) RotateProxy(ctx context.Context, proxyID string, port int) (*ProxyRotateResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if port > 0 {
		body["port"] = port
	}
	var result ProxyRotateResult
	if err := c.do(ctx, "POST", "/proxies/"+url.PathEscape(proxyID)+"/rotate", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProxyUsage returns cached GB used/remaining (refreshed ~5min, no
// upstream call).
func (c *VirtualSMS) GetProxyUsage(ctx context.Context, proxyID string) (*ProxyUsage, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result ProxyUsage
	if err := c.do(ctx, "GET", "/proxies/"+url.PathEscape(proxyID)+"/usage", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProxyUsageHistory returns a per-day GB/requests series. rangeParam is
// "7d" (default) or "30d".
func (c *VirtualSMS) GetProxyUsageHistory(ctx context.Context, proxyID, rangeParam string) (*ProxyUsageHistoryResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var q url.Values
	if rangeParam != "" {
		q = url.Values{"range": {rangeParam}}
	}
	var result ProxyUsageHistoryResult
	if err := c.do(ctx, "GET", "/proxies/"+url.PathEscape(proxyID)+"/usage-history", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetProxyTargeting persists default geo-targeting on a proxy sub-user.
// Country-only is free; cities/asns bill the customer's own funded GB at
// 2x on non-premium pools (free on residential_premium — the response's
// Premium2x field reflects this).
func (c *VirtualSMS) SetProxyTargeting(ctx context.Context, proxyID string, params SetProxyTargetingParams) (*ProxyTargetingResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]any{
		"country_code": params.CountryCode,
	}
	if len(params.Cities) > 0 {
		body["cities"] = params.Cities
	}
	if len(params.ASNs) > 0 {
		body["asns"] = params.ASNs
	}
	var result ProxyTargetingResult
	if err := c.do(ctx, "POST", "/proxies/"+url.PathEscape(proxyID)+"/targeting", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TestProxy dials out through the proxy and reports the exit
// IP/country/city/ISP/latency. Server-side rate-limited to ~1 call per 20s
// per proxy.
func (c *VirtualSMS) TestProxy(ctx context.Context, proxyID string, params TestProxyParams) (*ProxyTestResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]any{"country": params.Country}
	if params.Session != "" {
		body["session"] = params.Session
	}
	if params.Protocol != "" {
		body["protocol"] = params.Protocol
	}
	var result ProxyTestResult
	if err := c.do(ctx, "POST", "/proxies/"+url.PathEscape(proxyID)+"/test", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListProxyLocations discovers valid cities/states/asns/zips for a
// pool_type+country. Public endpoint, no auth, 6h server-side cache. NOT
// available for ProxyPoolResidentialPremium.
func (c *VirtualSMS) ListProxyLocations(ctx context.Context, params ListProxyLocationsParams) ([]ProxyLocationItem, error) {
	q := url.Values{
		"pool_type": {params.PoolType},
		"country":   {params.Country},
		"kind":      {params.Kind},
	}
	var raw struct {
		Items []ProxyLocationItem `json:"items"`
	}
	if err := c.do(ctx, "GET", "/proxies/locations", q, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Items, nil
}

// ─── generate_proxy_endpoint (client-side, pure function) ─────────────────
//
// Mirrors frontend/src/components/my-numbers-v2/ProxyEndpointGenerator.tsx
// buildUsername()/buildEndpoint() byte-identically — this is a shared
// client-side contract, not a backend call, so drift here silently breaks
// connection strings. No network call except the ListProxies lookup for
// credentials.

func buildProxyUsername(login, countryCode, targetBy, locationCode string, stickyIndex, stickyMinutes int) string {
	u := login + "__cr." + strings.ToLower(countryCode)
	loc := strings.TrimSpace(locationCode)
	if loc != "" && targetBy != "country" {
		switch targetBy {
		case "state":
			u += ";state." + strings.ToLower(loc)
		case "city":
			u += ";city." + strings.ToLower(loc)
		case "zip":
			u += ";zip." + loc
		case "asn":
			u += ";asn." + loc
		}
	}
	if stickyIndex > 0 {
		u += fmt.Sprintf(";sessid.s%d;sessttl.%d", stickyIndex, stickyMinutes)
	}
	return u
}

func buildProxyEndpointString(host string, port int, user, pass, format, protocol string) string {
	switch format {
	case "user:pass@host:port":
		return fmt.Sprintf("%s:%s@%s:%d", user, pass, host, port)
	case "curl":
		scheme := "http"
		if protocol == "SOCKS5" {
			scheme = "socks5h"
		}
		return fmt.Sprintf(`curl -x "%s://%s:%s@%s:%d" https://api.ipify.org`, scheme, user, pass, host, port)
	default: // "host:port:user:pass"
		return fmt.Sprintf("%s:%d:%s:%s", host, port, user, pass)
	}
}

// GenerateProxyEndpoint composes ready-to-use connection string(s) for a
// proxy you already own. Pure computation — no purchase, nothing
// persisted. Looks up the proxy's credentials via ListProxies first.
func (c *VirtualSMS) GenerateProxyEndpoint(ctx context.Context, params GenerateProxyEndpointParams) (*ProxyEndpointResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	proxies, err := c.ListProxies(ctx)
	if err != nil {
		return nil, err
	}
	var proxy *ProxyListItem
	for i := range proxies {
		if proxies[i].ProxyID == params.ProxyID {
			proxy = &proxies[i]
			break
		}
	}
	if proxy == nil {
		return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("proxy %s does not exist on this account", params.ProxyID), sentinel: ErrNoNumbers}
	}

	targetBy := params.TargetBy
	if targetBy == "" {
		targetBy = "country"
	}
	session := params.Session
	if session == "" {
		session = "rotating"
	}
	protocol := params.Protocol
	if protocol == "" {
		protocol = "HTTP"
	}
	format := params.Format
	if format == "" {
		format = "host:port:user:pass"
	}
	ttl := params.StickyTTLMinutes
	if ttl <= 0 {
		ttl = 10
	}
	count := params.Count
	if count <= 0 {
		count = 1
	}
	if count > 100 {
		count = 100
	}
	port := proxyHTTPPort
	if protocol == "SOCKS5" {
		port = proxySOCKS5Port
	}

	premium2x := targetBy != "country" && strings.TrimSpace(params.LocationCode) != "" && proxy.PoolType != string(ProxyPoolResidentialPremium)

	var endpoints []string
	if session == "rotating" {
		user := buildProxyUsername(proxy.ProxyLogin, params.CountryCode, targetBy, params.LocationCode, 0, 0)
		ep := buildProxyEndpointString(proxy.ProxyHost, port, user, proxy.ProxyPassword, format, protocol)
		for i := 0; i < count; i++ {
			endpoints = append(endpoints, ep)
		}
	} else {
		for i := 1; i <= count; i++ {
			user := buildProxyUsername(proxy.ProxyLogin, params.CountryCode, targetBy, params.LocationCode, i, ttl)
			endpoints = append(endpoints, buildProxyEndpointString(proxy.ProxyHost, port, user, proxy.ProxyPassword, format, protocol))
		}
	}

	result := &ProxyEndpointResult{
		ProxyID:      proxy.ProxyID,
		PoolType:     proxy.PoolType,
		Host:         proxy.ProxyHost,
		Port:         port,
		Protocol:     protocol,
		Session:      session,
		CountryCode:  params.CountryCode,
		TargetBy:     targetBy,
		LocationCode: params.LocationCode,
		Premium2x:    premium2x,
		Endpoints:    endpoints,
	}
	if session == "sticky" {
		result.StickyTTLMinutes = ttl
	}
	return result, nil
}
