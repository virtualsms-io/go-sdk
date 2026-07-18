# VirtualSMS Go SDK

## What is VirtualSMS?

Official Go SDK for the VirtualSMS API. VirtualSMS is an account verification platform for
individuals, developers, and AI agents: one-time SMS verification, dedicated number rentals,
matching-country proxies, and private cloud browser sessions (beta), all behind one API, one
MCP server, and one prepaid balance. This package is a native Go client over the REST API,
backed by real carrier-issued mobile numbers (real physical SIM cards, not VoIP) across
2500+ services in 145+ countries.

Built for developers and AI agents: REST API, hosted MCP server, SDKs.

This is a **REST v1-native client**. It talks directly to `https://virtualsms.io/api/v1/*`, and
is not a wrapper around any legacy or third-party client library.

## Install

```bash
go get github.com/virtualsms-io/go-sdk/v2
```

## Quickstart

1. **Get an API key** at [virtualsms.io/dashboard](https://virtualsms.io/dashboard).
2. **Buy a number** and **wait for the code**:

```go
package main

import (
	"context"
	"fmt"
	"log"

	virtualsms "github.com/virtualsms-io/go-sdk/v2"
)

func main() {
	client := virtualsms.New("your-api-key")
	ctx := context.Background()

	order, err := client.CreateOrder(ctx, "telegram", "US")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Number:", order.PhoneNumber)

	result, err := client.WaitForSMS(ctx, order.OrderID, 120) // seconds; 0 = default 300s
	if err != nil {
		log.Fatal(err)
	}
	if result.Success {
		fmt.Println("Code:", result.Code)
	} else {
		fmt.Println("No SMS yet, retry WaitForSMS, or CancelOrder for a refund.")
	}
}
```

More flows: see [`examples/`](./examples), activation, rental, and proxy.

## Capabilities

1. **One-time SMS verification.** Receive a code for a service like WhatsApp, Telegram, Discord,
   or a dating app, on demand, from $0.05 per code.
2. **Dedicated number rentals.** Hold one number for 1-30 days and receive SMS from any service
   on that number, from $0.25/day.
3. **Matching-country proxies.** Pair a number with an IP from the same country, across 223
   proxy countries, from $1.10/GB.
4. **Private cloud browser sessions (beta).** Start a country-matched browser in a live viewer
   for the signup step itself, invite-only.

## Why real SIM cards

VirtualSMS runs on real carrier-issued mobile numbers, backed by real physical SIM cards,
not VoIP. Services like WhatsApp, Telegram, Discord, and dating apps run a carrier lookup
before they send a code, and VoIP or virtual numbers fail that check more often than a real
SIM does. A physical SIM on a real carrier network reads like any other phone on that network,
carriers like Vodafone, O2, and T-Mobile depending on the country, which is part of why
VirtualSMS holds a 95%+ success rate across 2500+ services in 145+ countries.

## API coverage

46 methods across seven groups, generated from the canonical [VirtualSMS SDK v2.0.0
spec](https://virtualsms.io):

| Group | Methods |
|---|---|
| Activations / Orders | `ListServices`, `ListCountries`, `GetPrice`, `CreateOrder`, `GetOrder`, `GetSMS`, `WaitForSMS`, `CancelOrder`, `SwapNumber`, `RetryOrder`, `ListOrders`, `OrderHistory`, `CancelAllOrders`, `SearchServices`, `FindCheapest` |
| Rentals | `RentalsPricing`, `RentalsAvailable`, `RentalsServices`, `RentalsPrice`, `CreateRental`, `ListRentals`, `GetRental`, `ExtendRental`, `CancelRental` |
| Proxies | `ListProxyCatalog`, `ListProxies`, `BuyProxy`, `RotateProxy`, `GetProxyUsage`, `GetProxyUsageHistory`, `SetProxyTargeting`, `TestProxy`, `ListProxyLocations`, `GenerateProxyEndpoint` |
| Account | `GetBalance`, `GetProfile`, `GetTransactions`, `GetStats` |
| Session (beta) | `StartManualRegistrationSession` |
| Tools | `CheckNumber` |
| Webhooks | `ListWebhooks`, `CreateWebhook`, `GetWebhook`, `UpdateWebhook`, `DeleteWebhook`, `TestWebhook`, `ListWebhookDeliveries` |

### Rentals: two tiers

- **Full Access** (`virtualsms.RentalTierFullAccess`), local SIM inventory, usable for any
  service, longer durations, optional auto-renew.
- **Platform** (`virtualsms.RentalTierPlatform`), sourced via our global supplier network, locked
  to one chosen service per number, 24/72/168h durations only.

Both tiers carry the same refund terms: full refund within 20 minutes of purchase and before the
first SMS.

### Errors

Every failed call returns an error wrapping one of five sentinel values, so you can branch with
`errors.Is`:

```go
order, err := client.CreateOrder(ctx, "telegram", "US")
switch {
case errors.Is(err, virtualsms.ErrBadKey):
	// invalid/missing API key
case errors.Is(err, virtualsms.ErrInsufficientBalance):
	// top up before retrying
case errors.Is(err, virtualsms.ErrNotFound):
	// order/rental/proxy/webhook id does not exist
case errors.Is(err, virtualsms.ErrNoNumbers):
	// no numbers/stock available for this service+country (sniffed from a
	// 503 body, the backend has no distinct status code for this yet)
case errors.Is(err, virtualsms.ErrRateLimited):
	// back off, never auto-retry
case err != nil:
	var se *virtualsms.ServerError
	if errors.As(err, &se) && se.IsRetryable() {
		// safe to retry (GET only)
	}
	// otherwise: a 5xx on a mutating call may have gone through server-side,
	// verify with a list/get call (ListOrders, GetOrder, ListRentals, ...)
	// before retrying, since you may have been charged.
}
```

The client also applies a bounded retry (3 attempts, exponential backoff) to **GET-only** requests
on network errors or 5xx responses. Mutating calls (create/cancel/swap/rotate/extend/...) are never
auto-retried, a 5xx there does not prove the operation failed.

### Options

```go
client := virtualsms.New(
	apiKey,
	virtualsms.WithBaseURL("https://staging.virtualsms.io/api/v1"), // default: production
	virtualsms.WithTimeout(15 * time.Second),                       // default: 30s
	virtualsms.WithHTTPClient(customClient),                        // default: internal *http.Client
)
```

## AI agents and MCP

This SDK is the API-client half: a native Go wrapper around the REST API for services and CLIs
that call Go directly. VirtualSMS also runs a separate hosted MCP server so an AI agent
(Claude, Cursor, or any MCP-compatible client) can request a number, wait for a code, or manage
a rental the same way a developer would call the API directly.

## FAQ

### What is VirtualSMS?

VirtualSMS is an account verification platform for individuals, developers, and AI agents. It combines one-time SMS verification, dedicated number rentals, matching-country proxies, and private cloud browser sessions behind one API, one MCP server, and one prepaid balance.

### Does VirtualSMS use real SIM cards or VoIP numbers?

VirtualSMS uses real carrier-issued mobile numbers, backed by real physical SIM cards, not VoIP. Many services, including WhatsApp, Telegram, Discord, and dating apps, reject VoIP and virtual numbers at signup; a real physical SIM on a real carrier network passes that check far more often, which is reflected in a 95%+ success rate.

### Which services and countries does VirtualSMS support?

VirtualSMS covers 2500+ services across 145+ countries for SMS verification and number rentals, plus matching-country proxies across 223 proxy countries. Coverage spans messaging apps, social platforms, marketplaces, dating apps, and financial services.

### Can I rent a number, or only buy one-time codes?

Both. Buy a single one-time code from $0.05, or rent a dedicated number for 1-30 days from $0.25/day to receive SMS from any service on that number for the rental window.

### Does VirtualSMS work with AI agents and MCP?

Yes. VirtualSMS exposes a hosted MCP server plus a REST API and official SDKs in nine languages, so an AI agent can request a number, wait for a code, or manage a rental the same way a developer would call the API directly.

### How much does VirtualSMS cost?

Pricing is pay-as-you-go from one prepaid balance: SMS verification from $0.05 per code, number rentals from $0.25/day, and proxies from $1.10/GB. There is no subscription requirement.

### Is there a free API key?

Yes. Creating a VirtualSMS account issues an API key immediately, at no cost. You only spend from your prepaid balance when you place an order: an activation, a rental, or a proxy.

## Links

- **Homepage:** [virtualsms.io](https://virtualsms.io)
- **Docs:** [virtualsms.io/docs](https://virtualsms.io/docs)
- **pkg.go.dev:** [pkg.go.dev/github.com/virtualsms-io/go-sdk/v2](https://pkg.go.dev/github.com/virtualsms-io/go-sdk/v2)
- **MCP server:** [virtualsms.io/mcp](https://virtualsms.io/mcp)
- **Pricing:** [virtualsms.io/pricing](https://virtualsms.io/pricing)
- **REST API:** [virtualsms.io/api/v1](https://virtualsms.io/api/v1)
- **Other SDKs:** PHP, Node.js/TypeScript, Python, Ruby, .NET, Rust, Swift, and Java, all under [github.com/virtualsms-io](https://github.com/virtualsms-io)

## Publishing

Publishing this SDK is a **git tag only**, there is no CI publish workflow and no package-registry
account/token needed. [pkg.go.dev](https://pkg.go.dev) auto-indexes any public, tagged Go module on
first request. To cut a new version:

```bash
git tag v2.0.0
git push origin v2.0.0
```

Then visit `https://pkg.go.dev/github.com/virtualsms-io/go-sdk/v2@v2.0.0` once (or wait for the first
`go get`) to trigger indexing.

## Versioning

**v2.0.0** is the first REST v1-native major release. It is a breaking change from any v1.x line
that may have wrapped a legacy activation-dispatcher API, v2 talks to `/api/v1/*` REST endpoints
directly.

## Development

Run `sh scripts/check-positioning.sh` before committing copy changes. It fails on stale service
or country counts and other banned positioning wording.

## License

MIT
