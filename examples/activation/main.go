// Command activation demonstrates the core SMS-verification flow:
// find a service+country, buy a number, wait for the code, cancel if
// needed.
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

	// 1. Check balance before spending.
	balance, err := client.GetBalance(ctx)
	if err != nil {
		log.Fatalf("GetBalance: %v", err)
	}
	fmt.Printf("Balance: $%.2f\n", balance.BalanceUSD)

	// 2. Find the service code if you only know the app name.
	matches, err := client.SearchServices(ctx, "telegram")
	if err != nil {
		log.Fatalf("SearchServices: %v", err)
	}
	if len(matches.Matches) == 0 {
		log.Fatal("no matching service found")
	}
	service := matches.Matches[0].Code
	fmt.Printf("Using service code: %s\n", service)

	// 3. Check price + real stock before buying.
	price, err := client.GetPrice(ctx, service, "US")
	if err != nil {
		log.Fatalf("GetPrice: %v", err)
	}
	if !price.Available {
		log.Fatal("no stock for telegram/US right now — try FindCheapest for alternatives")
	}
	fmt.Printf("Price: $%.2f\n", price.PriceUSD)

	// 4. Buy the number.
	order, err := client.CreateOrder(ctx, service, "US")
	if err != nil {
		log.Fatalf("CreateOrder: %v", err)
	}
	fmt.Printf("Order %s: %s\n", order.OrderID, order.PhoneNumber)

	// 5. Block until the SMS arrives (or 300s timeout — see WaitForSMS doc).
	result, err := client.WaitForSMS(ctx, order.OrderID, 120)
	if err != nil {
		log.Fatalf("WaitForSMS: %v", err)
	}
	if !result.Success {
		fmt.Println("No SMS yet — cancelling the order for a refund.")
		if _, cerr := client.CancelOrder(ctx, order.OrderID); cerr != nil {
			log.Fatalf("CancelOrder: %v", cerr)
		}
		return
	}
	fmt.Printf("Code: %s\n", result.Code)
}
