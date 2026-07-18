// Command proxy demonstrates buying proxy traffic, listing owned proxies,
// and composing a ready-to-use connection string.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	virtualsms "github.com/virtualsms-io/go-sdk"
)

func main() {
	apiKey := os.Getenv("VIRTUALSMS_API_KEY")
	if apiKey == "" {
		log.Fatal("set VIRTUALSMS_API_KEY — get one at https://virtualsms.io/dashboard")
	}

	client := virtualsms.New(apiKey)
	ctx := context.Background()

	// 1. Browse the catalog: pool types, countries, price/GB.
	catalog, err := client.ListProxyCatalog(ctx)
	if err != nil {
		log.Fatalf("ListProxyCatalog: %v", err)
	}
	fmt.Printf("Proxy pool types available: %d\n", len(catalog))

	// 2. Buy 1GB of residential proxy traffic.
	purchase, err := client.BuyProxy(ctx, virtualsms.BuyProxyParams{
		PoolType: virtualsms.ProxyPoolResidential,
		GB:       1,
	})
	if err != nil {
		log.Fatalf("BuyProxy: %v", err)
	}
	fmt.Printf("Proxy %s purchased: %.1fGB remaining\n", purchase.ProxyID, purchase.GBRemaining)

	// 3. Compose a ready-to-use connection string (no purchase, pure
	// computation over the credentials you already own).
	endpoint, err := client.GenerateProxyEndpoint(ctx, virtualsms.GenerateProxyEndpointParams{
		ProxyID:     purchase.ProxyID,
		CountryCode: "US",
		Protocol:    "HTTP",
		Format:      "host:port:user:pass",
	})
	if err != nil {
		log.Fatalf("GenerateProxyEndpoint: %v", err)
	}
	fmt.Printf("Connection string: %s\n", endpoint.Endpoints[0])

	// 4. Verify it works by dialing out through it.
	test, err := client.TestProxy(ctx, purchase.ProxyID, virtualsms.TestProxyParams{Country: "US"})
	if err != nil {
		log.Fatalf("TestProxy: %v", err)
	}
	fmt.Printf("Exit IP: %s (%s)\n", test.ExitIP, test.CountryName)
}
