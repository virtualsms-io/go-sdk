package virtualsms

import (
	"context"
)

// StartManualRegistrationSession starts a country-matched cloud browser
// session the caller drives manually via the returned ViewerURL.
//
// Beta, invite-only feature. On a 403/404/503 (beta-gate signals) this
// returns a clean "invite-only beta" error rather than a raw HTTP error.
func (c *VirtualSMS) StartManualRegistrationSession(ctx context.Context, params StartSessionParams) (*BrowserSessionResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	withProxy := params.Country != ""
	if params.WithProxy != nil {
		withProxy = *params.WithProxy
	}
	mode := params.Mode
	if mode == "" {
		mode = "fresh"
	}
	body := map[string]any{
		"serviceName": params.ServiceName,
		"country":     params.Country,
		"deviceMode":  params.DeviceMode,
		"withProxy":   withProxy,
		"targetUrl":   params.TargetURL,
		"orderId":     params.OrderID,
		"mode":        mode,
	}

	var raw struct {
		Session *BrowserSessionResult `json:"session"`
		BrowserSessionResult
	}
	err := c.do(ctx, "POST", "/browser-sessions/start", nil, body, &raw)
	if err != nil {
		if isSessionsUnavailableError(err) {
			return nil, sessionsBetaError()
		}
		return nil, err
	}
	if raw.Session != nil {
		return raw.Session, nil
	}
	return &raw.BrowserSessionResult, nil
}

// isSessionsUnavailableError reports whether err looks like the manual
// registration sessions beta gate rejecting the request (403/404/503).
// Mirrors isSessionsUnavailableError in tools.ts.
func isSessionsUnavailableError(err error) bool {
	if ae, ok := err.(*APIError); ok {
		switch ae.StatusCode {
		case 403, 404:
			return true
		}
	}
	if se, ok := err.(*ServerError); ok && se.StatusCode == 503 {
		return true
	}
	return false
}

func sessionsBetaError() error {
	return &APIError{
		StatusCode: 0,
		Message:    "Manual registration sessions are an invite-only beta feature. Join https://t.me/VirtualSMS_io to request access.",
	}
}
