package virtualsms

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// platformTierCountryIDs maps ISO-3166 alpha-2 country codes to the
// internal numeric ID the platform-tier rental create endpoint requires.
// Required only for CreateRental with RentalTierPlatform — every other
// rentals endpoint resolves country_code server-side. Not every ISO code
// the catalog lists is rental-capable; an unmapped code means that country
// isn't available for platform-tier rentals. Ported verbatim from the MCP
// client (client.ts PLATFORM_TIER_COUNTRY_IDS) — this is the same mapping
// already shipped in the customer-facing frontend bundle for the same
// purpose.
var platformTierCountryIDs = map[string]int{
	"RU": 0, "UA": 1, "KZ": 2, "CN": 3, "PH": 4, "MM": 5, "ID": 6, "MY": 7, "KE": 8, "TZ": 9,
	"VN": 10, "KG": 11, "IL": 13, "HK": 14, "PL": 15, "GB": 16, "MG": 17, "CD": 18, "NG": 19,
	"MO": 20, "EG": 21, "IN": 22, "IE": 23, "KH": 24, "LA": 25, "HT": 26, "CI": 27, "GM": 28,
	"RS": 29, "YE": 30, "ZA": 31, "RO": 32, "CO": 33, "EE": 34, "AZ": 35, "CA": 36, "MA": 37,
	"GH": 38, "AR": 39, "UZ": 40, "CM": 41, "TD": 42, "DE": 43, "LT": 44, "HR": 45, "SE": 46,
	"IQ": 47, "NL": 48, "LV": 49, "AT": 50, "BY": 51, "TH": 52, "SA": 53, "MX": 54, "TW": 55,
	"ES": 56, "IR": 57, "DZ": 58, "SI": 59, "BD": 60, "SN": 61, "TR": 62, "CZ": 63, "LK": 64,
	"PE": 65, "PK": 66, "NZ": 67, "GN": 68, "ML": 69, "VE": 70, "ET": 71, "MN": 72, "BR": 73,
	"AF": 74, "UG": 75, "AO": 76, "CY": 77, "FR": 78, "PG": 79, "MZ": 80, "NP": 81, "BE": 82,
	"BG": 83, "HU": 84, "MD": 85, "IT": 86, "PY": 87, "HN": 88, "TN": 89, "NI": 90, "TL": 91,
	"BO": 92, "CR": 93, "GT": 94, "AE": 95, "ZW": 96, "PR": 97, "SD": 98, "TG": 99, "KW": 100,
	"SV": 101, "LY": 102, "JM": 103, "TT": 104, "EC": 105, "SZ": 106, "OM": 107, "BA": 108,
	"DO": 109, "SY": 110, "QA": 111, "PA": 112, "CU": 113, "MR": 114, "SL": 115, "JO": 116,
	"PT": 117, "BB": 118, "BI": 119, "BJ": 120, "BN": 121, "BS": 122, "BW": 123, "CF": 125,
	"GD": 127, "GE": 128, "GR": 129, "GW": 130, "GY": 131, "IS": 132, "KM": 133, "KN": 134,
	"LR": 135, "LS": 136, "MW": 137, "NA": 138, "NE": 139, "RW": 140, "SK": 141, "SR": 142,
	"TJ": 143, "MC": 144, "BH": 145, "RE": 146, "ZM": 147, "US": 187,
}

// RentalsPricing lists raw Full-Access pricing tiers (catalog dump, not
// authoritative for what's purchasable today — use RentalsAvailable for
// that). Public endpoint.
func (c *VirtualSMS) RentalsPricing(ctx context.Context) ([]RentalPricingTier, error) {
	var raw []RentalPricingTier
	if err := c.do(ctx, "GET", "/rentals/pricing", nil, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// RentalsAvailable lists country availability + pricing per tier. Public
// endpoint. params.Tier defaults to RentalTierFullAccess.
func (c *VirtualSMS) RentalsAvailable(ctx context.Context, params RentalAvailableParams) (*RentalAvailabilityResult, error) {
	q := url.Values{}
	if params.Country != "" {
		q.Set("country", params.Country)
	}
	if params.Service != "" {
		q.Set("service", params.Service)
	}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	// "platform" tier maps to the backend's opaque provider=network token.
	if params.Tier == RentalTierPlatform {
		q.Set("provider", "network")
	}
	var result RentalAvailabilityResult
	if err := c.do(ctx, "GET", "/rentals/available", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RentalsServices lists platform-tier services available in a country with
// stock and retail price. Public endpoint. durationHours <= 0 defaults to
// 24. Applies an explicit field allowlist: never forwards an internal
// supplier-code field the backend response may include.
func (c *VirtualSMS) RentalsServices(ctx context.Context, countryCode string, durationHours int) ([]RentalCatalogService, error) {
	if durationHours <= 0 {
		durationHours = 24
	}
	q := url.Values{"country_code": {countryCode}, "duration": {strconv.Itoa(durationHours)}}
	var raw []struct {
		ServiceID     string  `json:"service_id"`
		ServiceName   string  `json:"service_name"`
		PhysicalCount int     `json:"physical_count"`
		OurPrice      float64 `json:"our_price"`
		BasePrice     float64 `json:"base_price"`
		Popular       bool    `json:"popular"`
		IconURL       string  `json:"icon_url"`
	}
	if err := c.do(ctx, "GET", "/rentals/services", q, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]RentalCatalogService, 0, len(raw))
	for _, s := range raw {
		out = append(out, RentalCatalogService{
			ServiceID:     s.ServiceID,
			ServiceName:   s.ServiceName,
			PhysicalCount: s.PhysicalCount,
			OurPrice:      s.OurPrice,
			BasePrice:     s.BasePrice,
			Popular:       s.Popular,
			IconURL:       s.IconURL,
		})
	}
	return out, nil
}

// RentalsPrice gets the catalog price for a (service, country, duration)
// platform-tier combo. Public endpoint.
func (c *VirtualSMS) RentalsPrice(ctx context.Context, service, countryCode string, durationHours int) (*RentalPriceResult, error) {
	q := url.Values{
		"service":      {service},
		"country_code": {countryCode},
		"duration":     {strconv.Itoa(durationHours)},
	}
	var result RentalPriceResult
	if err := c.do(ctx, "GET", "/rentals/price", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRental creates a rental in either tier. For RentalTierPlatform,
// params.Country must be an ISO-2 code present in the internal
// platform-tier lookup table (see RentalsAvailable with Tier=RentalTierPlatform
// to discover supported countries) — this SDK resolves the numeric ID for
// you, callers never need to know or pass it.
func (c *VirtualSMS) CreateRental(ctx context.Context, params CreateRentalParams) (*CreateRentalResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	switch params.Tier {
	case RentalTierPlatform:
		return c.createPlatformRental(ctx, params)
	case RentalTierFullAccess, "":
		return c.createFullAccessRental(ctx, params)
	default:
		return nil, fmt.Errorf("virtualsms: unknown rental tier %q", params.Tier)
	}
}

func (c *VirtualSMS) createFullAccessRental(ctx context.Context, params CreateRentalParams) (*CreateRentalResult, error) {
	rentalType := "full"
	if params.Service != "" {
		rentalType = "service"
	}
	body := map[string]any{
		"country":        params.Country,
		"rental_type":    rentalType,
		"duration_hours": params.DurationHours,
		"auto_renew":     params.AutoRenew,
	}
	if params.Service != "" {
		body["service"] = params.Service
	}
	var result CreateRentalResult
	if err := c.do(ctx, "POST", "/rentals", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *VirtualSMS) createPlatformRental(ctx context.Context, params CreateRentalParams) (*CreateRentalResult, error) {
	countryID, ok := platformTierCountryIDs[strings.ToUpper(params.Country)]
	if !ok {
		return nil, fmt.Errorf(
			"virtualsms: platform-tier rentals are not available for country_code %q; "+
				"use RentalsAvailable(tier=platform) to see supported countries", params.Country)
	}
	body := map[string]any{
		"service":        params.Service,
		"country":        countryID,
		"duration_hours": params.DurationHours,
		"provider":       "network",
	}
	var raw struct {
		Success     *bool   `json:"success"`
		RentalID    string  `json:"rental_id"`
		PhoneNumber string  `json:"phone_number"`
		ExpiresAt   string  `json:"expires_at"`
		RetailCost  float64 `json:"retail_cost"`
		Currency    string  `json:"currency"`
	}
	if err := c.do(ctx, "POST", "/rentals/provider", nil, body, &raw); err != nil {
		return nil, err
	}
	success := true
	if raw.Success != nil {
		success = *raw.Success
	}
	return &CreateRentalResult{
		Success:     success,
		RentalID:    raw.RentalID,
		PhoneNumber: raw.PhoneNumber,
		ExpiresAt:   raw.ExpiresAt,
		RetailCost:  raw.RetailCost,
		Currency:    raw.Currency,
		Status:      "active",
	}, nil
}

// ListRentals lists rentals, optionally filtered by status (server default
// is "active" when empty).
func (c *VirtualSMS) ListRentals(ctx context.Context, status string) ([]Rental, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var q url.Values
	if status != "" {
		q = url.Values{"status": {status}}
	}
	var raw []Rental
	if err := c.do(ctx, "GET", "/rentals", q, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetRental returns one rental by id, or nil if not found. No dedicated
// GET-by-id backend route exists (confirmed intentional) — this is a
// client-side ListRentals(status="all") + find.
func (c *VirtualSMS) GetRental(ctx context.Context, rentalID string) (*Rental, error) {
	all, err := c.ListRentals(ctx, "all")
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == rentalID {
			return &all[i], nil
		}
	}
	return nil, nil
}

// ExtendRental extends an active rental, charged at the current catalog
// price.
func (c *VirtualSMS) ExtendRental(ctx context.Context, rentalID string, durationHours int) (*RentalActionResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	body := map[string]any{"duration_hours": durationHours}
	var result RentalActionResult
	if err := c.do(ctx, "POST", "/rentals/"+url.PathEscape(rentalID)+"/extend", nil, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelRental gives a full refund — only within 20 minutes of purchase
// and before the first SMS, either tier.
func (c *VirtualSMS) CancelRental(ctx context.Context, rentalID string) (*RentalActionResult, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var result RentalActionResult
	if err := c.do(ctx, "POST", "/rentals/"+url.PathEscape(rentalID)+"/cancel", nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReleaseRental is intentionally NOT implemented in v2.0.0.
//
// It is gated on the MCP surface behind VIRTUALSMS_ENABLE_RELEASE pending a
// pricing decision (undocumented 10%-fee + store-credit refund policy).
// Do not add this until the feature is ungated upstream — see the SDK
// spec's appendix for details.
