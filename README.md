# virtualsms-go

The official, native Go client for the [VirtualSMS](https://virtualsms.io) REST API v1 -  SMS
verification numbers, number rentals, and residential/mobile/datacenter proxies, all in one SDK.

This is a **REST v1-native client**. It talks directly to `https://virtualsms.io/api/v1/*` -  it is
not a wrapper around any legacy or third-party client library.

## Install

```bash
go get github.com/virtualsms-io/go-sdk
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

	virtualsms "github.com/virtualsms-io/go-sdk"
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
		fmt.Println("No SMS yet -  retry WaitForSMS, or CancelOrder for a refund.")
	}
}
```

More flows: see [`examples/`](./examples) -  activation, rental, and proxy.

## What's covered

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

- **Full Access** (`virtualsms.RentalTierFullAccess`) -  local SIM inventory, usable for any
  service, longer durations, optional auto-renew.
- **Platform** (`virtualsms.RentalTierPlatform`) -  sourced via our global supplier network, locked
  to one chosen service per number, 24/72/168h durations only.

Both tiers carry the same refund terms: full refund within 20 minutes of purchase and before the
first SMS.

## Errors

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
	// 503 body -- the backend has no distinct status code for this yet)
case errors.Is(err, virtualsms.ErrRateLimited):
	// back off, never auto-retry
case err != nil:
	var se *virtualsms.ServerError
	if errors.As(err, &se) && se.IsRetryable() {
		// safe to retry (GET only)
	}
	// otherwise: a 5xx on a mutating call may have gone through server-side - 
	// verify with a list/get call (ListOrders, GetOrder, ListRentals, ...)
	// before retrying, since you may have been charged.
}
```

The client also applies a bounded retry (3 attempts, exponential backoff) to **GET-only** requests
on network errors or 5xx responses. Mutating calls (create/cancel/swap/rotate/extend/...) are never
auto-retried -  a 5xx there does not prove the operation failed.

## Options

```go
client := virtualsms.New(
	apiKey,
	virtualsms.WithBaseURL("https://staging.virtualsms.io/api/v1"), // default: production
	virtualsms.WithTimeout(15 * time.Second),                       // default: 30s
	virtualsms.WithHTTPClient(customClient),                        // default: internal *http.Client
)
```

## Publishing

Publishing this SDK is a **git tag only** -  there is no CI publish workflow and no package-registry
account/token needed. [pkg.go.dev](https://pkg.go.dev) auto-indexes any public, tagged Go module on
first request. To cut a new version:

```bash
git tag v2.0.0
git push origin v2.0.0
```

Then visit `https://pkg.go.dev/github.com/virtualsms-io/go-sdk@v2.0.0` once (or wait for the first
`go get`) to trigger indexing.

## Versioning

**v2.0.0** is the first REST v1-native major release. It is a breaking change from any v1.x line
that may have wrapped a legacy activation-dispatcher API -  v2 talks to `/api/v1/*` REST endpoints
directly.

## License

MIT
