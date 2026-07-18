// Command rental demonstrates renting a number (Full Access tier: local SIM
// inventory, any service) and managing it through its lifecycle.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	virtualsms "github.com/virtualsms-io/go-sdk/v2"
)

func main() {
	apiKey := os.Getenv("VIRTUALSMS_API_KEY")
	if apiKey == "" {
		log.Fatal("set VIRTUALSMS_API_KEY — get one at https://virtualsms.io/dashboard")
	}

	client := virtualsms.New(apiKey)
	ctx := context.Background()

	// 1. See what's available before renting.
	avail, err := client.RentalsAvailable(ctx, virtualsms.RentalAvailableParams{
		Country: "US",
		Tier:    virtualsms.RentalTierFullAccess,
	})
	if err != nil {
		log.Fatalf("RentalsAvailable: %v", err)
	}
	fmt.Printf("Full-Access countries available: %d\n", avail.TotalAvailable)

	// 2. Create a 24h Full Access rental (any service on this number).
	rental, err := client.CreateRental(ctx, virtualsms.CreateRentalParams{
		Tier:          virtualsms.RentalTierFullAccess,
		Country:       "US",
		DurationHours: 24,
	})
	if err != nil {
		log.Fatalf("CreateRental: %v", err)
	}
	fmt.Printf("Rental %s: %s (expires %s)\n", rental.RentalID, rental.PhoneNumber, rental.ExpiresAt)

	// 3. List active rentals.
	rentals, err := client.ListRentals(ctx, "active")
	if err != nil {
		log.Fatalf("ListRentals: %v", err)
	}
	fmt.Printf("Active rentals: %d\n", len(rentals))

	// 4. Extend it by another 24h if still needed.
	if _, err := client.ExtendRental(ctx, rental.RentalID, 24); err != nil {
		log.Printf("ExtendRental: %v (may be outside the extend window, that's fine for this demo)", err)
	}

	// 5. Cancel for a full refund — only works within 20 minutes of
	// purchase and before the first SMS.
	if _, err := client.CancelRental(ctx, rental.RentalID); err != nil {
		log.Printf("CancelRental: %v (expected once the refund window has passed)", err)
	}
}
