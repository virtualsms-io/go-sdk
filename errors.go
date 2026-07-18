package virtualsms

import (
	"errors"
	"fmt"
)

// Sentinel errors. Compare with errors.Is(err, virtualsms.ErrBadKey) etc. —
// every failed call returns an *APIError or *ServerError that wraps one of
// these (except generic 4xx, which wraps none and is compared by message
// only).
var (
	// ErrBadKey is returned on HTTP 401: the API key is missing or invalid.
	ErrBadKey = errors.New("invalid or missing API key")

	// ErrInsufficientBalance is returned on HTTP 402: the account balance is
	// too low to complete the requested purchase.
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrNoNumbers is returned on HTTP 404: the requested resource (order,
	// rental, proxy, or a service+country combo with no available numbers)
	// was not found / not in stock.
	ErrNoNumbers = errors.New("not found / no numbers available")

	// ErrRateLimited is returned on HTTP 429. Never auto-retried by this
	// SDK — fighting the server's own rate limiter is wrong. Back off and
	// slow down.
	ErrRateLimited = errors.New("rate limit exceeded")
)

// APIError wraps a failed VirtualSMS API call with the HTTP status code and
// the raw backend message. Use errors.Is to check against the sentinel
// errors above, or inspect StatusCode/Message directly.
type APIError struct {
	StatusCode int
	Message    string
	sentinel   error
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("virtualsms: api error (status %d)", e.StatusCode)
	}
	return fmt.Sprintf("virtualsms: %s (status %d)", e.Message, e.StatusCode)
}

// Unwrap lets errors.Is(err, ErrBadKey) etc. work against wrapped APIErrors.
func (e *APIError) Unwrap() error { return e.sentinel }

// ServerError wraps a 5xx response. Retryable is true only for GET/HEAD
// requests — this SDK's own bounded retry logic already attempted those
// before surfacing the error, so Retryable=true here means "safe for the
// caller to retry again later," not "the SDK will retry it for you."
//
// Retryable is false for a 5xx on a mutating call (POST/PUT/PATCH/DELETE):
// the operation may have completed server-side despite the error. NEVER
// blindly retry a mutating call on a ServerError — verify first via a read
// call (ListOrders, GetOrder, ListRentals, etc.) whether it actually
// succeeded, since you may have been charged.
type ServerError struct {
	StatusCode int
	Message    string
	Mutating   bool
}

func (e *ServerError) Error() string {
	if e.Mutating {
		return fmt.Sprintf(
			"virtualsms: server error (%d) on a request that may have made a purchase or changed state; "+
				"do NOT blindly retry, verify first with a list/get call whether it succeeded: %s",
			e.StatusCode, e.Message,
		)
	}
	return fmt.Sprintf("virtualsms: server error (%d), safe to retry this read-only request: %s", e.StatusCode, e.Message)
}

// Retryable reports whether it is safe for the caller to retry the request
// that produced this error (true only for GET/HEAD requests).
func (e *ServerError) IsRetryable() bool { return !e.Mutating }
