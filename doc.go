// Package virtualsms is the official Go client for the VirtualSMS REST API (https://virtualsms.io).
// It provides SMS verification, number rentals, and matching-country proxies backed by real
// carrier-issued mobile numbers, not VoIP, across 2500+ services in 145+ countries.
//
// This is a REST v1-native client: it talks directly to
// https://virtualsms.io/api/v1/*. It is NOT a drop-in replacement for any
// legacy sms-activate-compatible client library.
//
// # Getting started
//
// Get an API key at https://virtualsms.io/dashboard, then:
//
//	client := virtualsms.New("your-api-key")
//
//	order, err := client.CreateOrder(ctx, "telegram", "US")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	result, err := client.WaitForSMS(ctx, order.OrderID, 0) // 0 = default 300s
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result.Code)
//
// # Errors
//
// Failed calls return errors wrapping one of the sentinel values in
// errors.go (ErrBadKey, ErrInsufficientBalance, ErrNoNumbers,
// ErrRateLimited) so callers can branch with errors.Is. See errors.go for
// the full status-code mapping and the retry/idempotency contract.
//
// # Homepage
//
// https://virtualsms.io: dashboard, API key management, and full REST v1
// reference documentation live there.
package virtualsms
