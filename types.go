package virtualsms

// ─── Catalog / pricing ──────────────────────────────────────────────────────

// Service is one SMS-verification service (e.g. Telegram, WhatsApp).
type Service struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// Country is one country available for number purchase/rental.
type Country struct {
	ISO  string `json:"iso"`
	Name string `json:"name"`
	Flag string `json:"flag,omitempty"`
}

// Price is the result of GetPrice: price plus real stock availability.
type Price struct {
	PriceUSD  float64 `json:"price_usd"`
	Currency  string  `json:"currency"`
	Available bool    `json:"available"`
}

// CatalogCountry is one row of the /catalog/countries response — the
// source of truth for real per-country stock (Count > 0 = in stock).
type CatalogCountry struct {
	ISO      string  `json:"iso"`
	Name     string  `json:"name"`
	PriceUSD float64 `json:"price_usd"`
	Count    int     `json:"count"`
}

// ─── Account ────────────────────────────────────────────────────────────────

// Balance is the account's current balance.
type Balance struct {
	BalanceUSD float64 `json:"balance_usd"`
}

// Profile is the full account profile.
type Profile struct {
	ID               string  `json:"id"`
	Email            string  `json:"email"`
	TelegramLinked   bool    `json:"telegram_linked"`
	TelegramUsername string  `json:"telegram_username,omitempty"`
	BalanceUSD       float64 `json:"balance_usd"`
	TotalSpentUSD    float64 `json:"total_spent_usd"`
	TotalCreditsUSD  float64 `json:"total_credits_usd"`
	TotalOrders      int     `json:"total_orders"`
	ActiveAPIKeys    int     `json:"active_api_keys"`
	CreatedAt        string  `json:"created_at"`
}

// Transaction is one row of transaction history.
type Transaction struct {
	ID            string  `json:"id"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Description   string  `json:"description,omitempty"`
	OrderID       string  `json:"order_id,omitempty"`
	BalanceBefore float64 `json:"balance_before"`
	BalanceAfter  float64 `json:"balance_after"`
	CreatedAt     string  `json:"created_at"`
}

// TransactionsPage is a paginated transaction history result.
type TransactionsPage struct {
	Count        int           `json:"count"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
	Transactions []Transaction `json:"transactions"`
}

// GetTransactionsParams filters GetTransactions.
type GetTransactionsParams struct {
	// Type filters by transaction type: "deposit", "purchase", "refund", "admin_credit".
	Type string
	// From is an RFC3339 or YYYY-MM-DD date string, inclusive lower bound.
	From string
	// To is an RFC3339 or YYYY-MM-DD date string, inclusive upper bound.
	To string
	// Limit is 1-200, default 50.
	Limit int
	// Offset is 0-indexed, default 0.
	Offset int
}

// Stats is the aggregated usage summary returned by GetStats.
type Stats struct {
	WindowDays       int            `json:"window_days"`
	BalanceUSD       float64        `json:"balance_usd"`
	TotalOrders      int            `json:"total_orders"`
	SuccessfulOrders int            `json:"successful_orders"`
	SuccessRate      *float64       `json:"success_rate"`
	TotalSpendUSD    float64        `json:"total_spend_usd"`
	StatusBreakdown  map[string]int `json:"status_breakdown"`
	TopServices      []StatsEntry   `json:"top_services"`
	TopCountries     []StatsEntry   `json:"top_countries"`
	Note             string         `json:"note,omitempty"`
}

// StatsEntry is one "key: count" row in a Stats breakdown.
type StatsEntry struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// ─── Orders ─────────────────────────────────────────────────────────────────

// SmsMessage is one inbound SMS on an order.
type SmsMessage struct {
	Content    string `json:"content"`
	Sender     string `json:"sender,omitempty"`
	ReceivedAt string `json:"received_at,omitempty"`
}

// Order is a purchased verification number and its lifecycle state.
type Order struct {
	OrderID           string       `json:"order_id"`
	PhoneNumber       string       `json:"phone_number"`
	Service           string       `json:"service,omitempty"`
	Country           string       `json:"country,omitempty"`
	Price             float64      `json:"price,omitempty"`
	CreatedAt         string       `json:"created_at,omitempty"`
	ExpiresAt         string       `json:"expires_at,omitempty"`
	Status            string       `json:"status"`
	SMSCode           string       `json:"sms_code,omitempty"`
	SMSText           string       `json:"sms_text,omitempty"`
	Messages          []SmsMessage `json:"messages,omitempty"`
	SMSReceived       bool         `json:"sms_received,omitempty"`
	CancelAvailableAt string       `json:"cancel_available_at,omitempty"`
	SwapAvailableAt   string       `json:"swap_available_at,omitempty"`
}

// CancelResult is the outcome of CancelOrder.
type CancelResult struct {
	Success  bool `json:"success"`
	Refunded bool `json:"refunded"`
}

// RetryOrderResult is the outcome of RetryOrder.
type RetryOrderResult struct {
	Success bool   `json:"success"`
	OrderID string `json:"order_id"`
	Message string `json:"message"`
}

// GetSMSResult is the normalized, thin-wrapper result of GetSMS.
type GetSMSResult struct {
	Status      string       `json:"status"`
	PhoneNumber string       `json:"phone_number"`
	Messages    []SmsMessage `json:"messages,omitempty"`
	Code        string       `json:"code,omitempty"`
	SMSCode     string       `json:"sms_code,omitempty"`
	SMSText     string       `json:"sms_text,omitempty"`
}

// WaitForSMSResult is the outcome of WaitForSMS — either a delivered SMS
// (Success=true) or a timeout (Success=false, no error raised).
type WaitForSMSResult struct {
	Success        bool         `json:"success"`
	OrderID        string       `json:"order_id"`
	PhoneNumber    string       `json:"phone_number"`
	Status         string       `json:"status,omitempty"`
	Messages       []SmsMessage `json:"messages,omitempty"`
	Code           string       `json:"code,omitempty"`
	DeliveryMethod string       `json:"delivery_method,omitempty"`
	ElapsedSeconds int          `json:"elapsed_seconds,omitempty"`
	Error          string       `json:"error,omitempty"`
}

// OrderHistoryParams filters OrderHistory.
type OrderHistoryParams struct {
	Status    string
	Service   string
	Country   string
	SinceDays int
	// Limit defaults to 20, hard-capped at 50.
	Limit int
}

// OrderHistoryResult is the filtered, capped result of OrderHistory.
type OrderHistoryResult struct {
	Count        int                `json:"count"`
	TotalMatched int                `json:"total_matched"`
	Filters      OrderHistoryParams `json:"filters"`
	Orders       []Order            `json:"orders"`
}

// CancelAllOrdersResult is the outcome of CancelAllOrders — a
// gather-with-partial-failure fan-out, never abort-on-first-error.
type CancelAllOrdersResult struct {
	Cancelled       int                       `json:"cancelled"`
	Failed          int                       `json:"failed"`
	TotalActive     int                       `json:"total_active"`
	CancelledOrders []CancelledOrderEntry     `json:"cancelled_orders"`
	Failures        []CancelOrderFailureEntry `json:"failures"`
}

// CancelledOrderEntry is one successfully cancelled order in
// CancelAllOrdersResult.
type CancelledOrderEntry struct {
	OrderID  string `json:"order_id"`
	Refunded bool   `json:"refunded"`
}

// CancelOrderFailureEntry is one failed cancellation in
// CancelAllOrdersResult.
type CancelOrderFailureEntry struct {
	OrderID string `json:"order_id"`
	Error   string `json:"error"`
}

// ServiceMatch is one scored result from SearchServices.
type ServiceMatch struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	MatchScore float64 `json:"match_score"`
}

// SearchServicesResult is the outcome of SearchServices.
type SearchServicesResult struct {
	Query   string         `json:"query"`
	Matches []ServiceMatch `json:"matches"`
	Message string         `json:"message,omitempty"`
	Tip     string         `json:"tip,omitempty"`
}

// CheapestOption is one in-stock country/price row from FindCheapest.
type CheapestOption struct {
	Country     string  `json:"country"`
	CountryName string  `json:"country_name"`
	PriceUSD    float64 `json:"price_usd"`
	Stock       bool    `json:"stock"`
}

// FindCheapestResult is the outcome of FindCheapest.
type FindCheapestResult struct {
	Service                 string           `json:"service"`
	CheapestOptions         []CheapestOption `json:"cheapest_options"`
	TotalAvailableCountries int              `json:"total_available_countries"`
	Message                 string           `json:"message,omitempty"`
}

// ─── Rentals ────────────────────────────────────────────────────────────────
//
// Two rental tiers, reflected generically (no supplier names):
//   RentalTierFullAccess: local SIM inventory, any service, longer durations.
//   RentalTierPlatform:   sourced via our global supplier network, locked to
//                         ONE chosen service per number, 24/72/168h only.
// Refunds are NOT a tier differentiator: both get a full refund within 20
// minutes of purchase and before the first SMS.

// RentalTier selects which rental inventory to use.
type RentalTier string

const (
	RentalTierFullAccess RentalTier = "full_access"
	RentalTierPlatform   RentalTier = "platform"
)

// RentalPricingTier is one raw catalog pricing row (not authoritative for
// what's purchasable today — use RentalsAvailable for that).
type RentalPricingTier struct {
	RentalType    string  `json:"rental_type"`
	DurationHours int     `json:"duration_hours"`
	DurationLabel string  `json:"duration_label"`
	BasePrice     float64 `json:"base_price"`
	CountryCode   string  `json:"country_code"`
	ServiceID     string  `json:"service_id"`
}

// RentalDurationPrice is one duration/price pair.
type RentalDurationPrice struct {
	DurationHours int     `json:"duration_hours"`
	DurationLabel string  `json:"duration_label"`
	Price         float64 `json:"price"`
}

// RentalAvailabilityCountry is one country's rental availability + pricing.
type RentalAvailabilityCountry struct {
	CountryCode     string                           `json:"country_code"`
	CountryName     string                           `json:"country_name"`
	Flag            string                           `json:"flag,omitempty"`
	AvailableCount  int                              `json:"available_count"`
	Pricing         map[string][]RentalDurationPrice `json:"pricing"`
	ServiceCount    int                              `json:"service_count,omitempty"`
	PopularServices []string                         `json:"popular_services,omitempty"`
	MinPricePerDay  float64                          `json:"min_price_per_day,omitempty"`
}

// RentalAvailabilityResult is the outcome of RentalsAvailable.
type RentalAvailabilityResult struct {
	Countries           []RentalAvailabilityCountry `json:"countries"`
	TotalAvailable      int                         `json:"total_available"`
	FullAccessCountries []map[string]any            `json:"full_access_countries,omitempty"`
	Provider            string                      `json:"provider,omitempty"`
}

// RentalAvailableParams filters RentalsAvailable.
type RentalAvailableParams struct {
	Country string
	Service string
	// Type is "service" or "full".
	Type string
	// Tier defaults to RentalTierFullAccess.
	Tier RentalTier
}

// RentalCatalogService is one platform-tier service available in a
// country, with stock and retail price.
type RentalCatalogService struct {
	ServiceID     string  `json:"service_id"`
	ServiceName   string  `json:"service_name"`
	PhysicalCount int     `json:"physical_count"`
	OurPrice      float64 `json:"our_price,omitempty"`
	BasePrice     float64 `json:"base_price,omitempty"`
	Popular       bool    `json:"popular"`
	IconURL       string  `json:"icon_url,omitempty"`
}

// RentalPriceResult is the outcome of RentalsPrice.
type RentalPriceResult struct {
	Price         float64 `json:"price"`
	DurationHours int     `json:"duration_hours"`
}

// CreateRentalParams describes a rental purchase (either tier).
type CreateRentalParams struct {
	// Tier is required: RentalTierFullAccess or RentalTierPlatform.
	Tier RentalTier
	// Country is required. ISO-2 for full_access; ISO-2 for platform (this
	// SDK resolves the internal numeric ID for you via the static lookup
	// table in rentals.go).
	Country string
	// DurationHours is required.
	DurationHours int
	// Service is required for the platform tier, optional for full_access.
	Service string
	// AutoRenew applies to the full_access tier only, default false.
	AutoRenew bool
}

// CreateRentalResult is the outcome of CreateRental.
type CreateRentalResult struct {
	Success     bool    `json:"success"`
	RentalID    string  `json:"rental_id"`
	PhoneNumber string  `json:"phone_number"`
	RentalType  string  `json:"rental_type,omitempty"`
	Service     string  `json:"service,omitempty"`
	Duration    string  `json:"duration,omitempty"`
	Price       float64 `json:"price,omitempty"`
	StartedAt   string  `json:"started_at,omitempty"`
	ExpiresAt   string  `json:"expires_at"`
	AutoRenew   bool    `json:"auto_renew,omitempty"`
	Status      string  `json:"status,omitempty"`
	RetailCost  float64 `json:"retail_cost,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

// Rental is an active or past rental.
type Rental struct {
	ID            string  `json:"id"`
	PhoneNumber   string  `json:"phone_number"`
	RentalType    string  `json:"rental_type"`
	ServiceID     string  `json:"service_id,omitempty"`
	DurationHours int     `json:"duration_hours"`
	StartedAt     string  `json:"started_at"`
	ExpiresAt     string  `json:"expires_at"`
	Price         float64 `json:"price"`
	AutoRenew     bool    `json:"auto_renew"`
	Status        string  `json:"status"`
	SMSReceived   int     `json:"sms_received"`
	SMSForwarded  int     `json:"sms_forwarded"`
	LastSMSAt     string  `json:"last_sms_at,omitempty"`
	Provider      string  `json:"provider"`
}

// RentalActionResult is the outcome of ExtendRental / CancelRental.
type RentalActionResult struct {
	Success      bool    `json:"success"`
	RentalID     string  `json:"rental_id"`
	Status       string  `json:"status,omitempty"`
	Refund       float64 `json:"refund,omitempty"`
	NewExpiresAt string  `json:"new_expires_at,omitempty"`
	Price        float64 `json:"price,omitempty"`
	HoursUsed    string  `json:"hours_used,omitempty"`
	Message      string  `json:"message,omitempty"`
}

// ─── Proxies ────────────────────────────────────────────────────────────────

// ProxyPoolType is one of the four purchasable proxy pools.
type ProxyPoolType string

const (
	ProxyPoolResidential        ProxyPoolType = "residential"
	ProxyPoolResidentialPremium ProxyPoolType = "residential_premium"
	ProxyPoolMobile             ProxyPoolType = "mobile"
	ProxyPoolDatacenter         ProxyPoolType = "datacenter"
)

// ProxyCatalogCountry is one country entry in a proxy pool's catalog.
type ProxyCatalogCountry struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	IPCount   int    `json:"ip_count"`
}

// ProxyCatalogPoolType is one pool type in the proxy catalog.
type ProxyCatalogPoolType struct {
	ID         string                `json:"id"`
	Label      string                `json:"label"`
	PricePerGB float64               `json:"price_per_gb"`
	Countries  []ProxyCatalogCountry `json:"countries"`
}

// ProxyListItem is one owned proxy, including its live credentials.
type ProxyListItem struct {
	ProxyID       string  `json:"proxy_id"`
	PoolType      string  `json:"pool_type"`
	CountryCode   string  `json:"country_code"`
	CountryName   string  `json:"country_name,omitempty"`
	GBTotal       float64 `json:"gb_total"`
	GBUsed        float64 `json:"gb_used"`
	GBRemaining   float64 `json:"gb_remaining"`
	ProxyHost     string  `json:"proxy_host"`
	ProxyPort     int     `json:"proxy_port"`
	ProxyLogin    string  `json:"proxy_login"`
	ProxyPassword string  `json:"proxy_password"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
}

// BuyProxyParams describes a proxy traffic purchase.
type BuyProxyParams struct {
	PoolType       ProxyPoolType
	GB             float64
	CountryCode    string // soft preference only
	IdempotencyKey string
}

// ProxyPurchaseResult is the outcome of BuyProxy.
type ProxyPurchaseResult struct {
	ProxyID        string  `json:"proxy_id"`
	PoolType       string  `json:"pool_type"`
	GBAdded        float64 `json:"gb_added"`
	GBRemaining    float64 `json:"gb_remaining"`
	CountryCode    string  `json:"country_code"`
	ProxyLogin     string  `json:"proxy_login"`
	ProxyPassword  string  `json:"proxy_password"`
	ProxyHost      string  `json:"proxy_host"`
	ProxyPort      int     `json:"proxy_port"`
	ProxyPortSocks int     `json:"proxy_port_socks,omitempty"`
	Price          float64 `json:"price"`
	Balance        float64 `json:"balance,omitempty"`
}

// ProxyRotateResult is the outcome of RotateProxy.
type ProxyRotateResult struct {
	Rotated bool   `json:"rotated"`
	Port    int    `json:"port"`
	Message string `json:"message"`
}

// ProxyUsage is cached (~5min) GB used/remaining for a proxy.
type ProxyUsage struct {
	GBUsed      float64 `json:"gb_used"`
	GBRemaining float64 `json:"gb_remaining"`
	Requests    int     `json:"requests"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

// ProxyUsageHistoryPoint is one day's GB/request usage.
type ProxyUsageHistoryPoint struct {
	Date     string  `json:"date"`
	GB       float64 `json:"gb"`
	Requests int     `json:"requests"`
}

// ProxyUsageHistoryTotals summarizes a ProxyUsageHistoryResult series.
type ProxyUsageHistoryTotals struct {
	GB       float64 `json:"gb"`
	Requests int     `json:"requests"`
}

// ProxyUsageHistoryResult is the outcome of GetProxyUsageHistory.
type ProxyUsageHistoryResult struct {
	Series []ProxyUsageHistoryPoint `json:"series"`
	Totals ProxyUsageHistoryTotals  `json:"totals"`
}

// SetProxyTargetingParams describes persistent default geo-targeting for a
// proxy. Country-only is free; Cities/ASNs bill 2x GB on non-premium pools
// (free on residential_premium).
type SetProxyTargetingParams struct {
	CountryCode string
	Cities      []string
	ASNs        []int
}

// ProxyTargetingResult is the outcome of SetProxyTargeting.
type ProxyTargetingResult struct {
	OK          bool   `json:"ok"`
	CountryCode string `json:"country_code"`
	// Premium2x is true when city/state/zip/asn targeting was requested on
	// a non-premium pool: it bills the customer's own funded GB at 2x.
	Premium2x bool `json:"premium_2x"`
}

// TestProxyParams describes a proxy dial-out test.
type TestProxyParams struct {
	Country string
	// Session is "rotating" or "sticky".
	Session string
	// Protocol is "http" or "socks5".
	Protocol string
}

// ProxyTestResult is the outcome of TestProxy.
type ProxyTestResult struct {
	OK          bool    `json:"ok"`
	ExitIP      string  `json:"exit_ip,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	CountryName string  `json:"country_name,omitempty"`
	City        string  `json:"city,omitempty"`
	Region      string  `json:"region,omitempty"`
	ISP         string  `json:"isp,omitempty"`
	ASN         string  `json:"asn,omitempty"`
	LatencyMs   float64 `json:"latency_ms,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// ProxyLocationItem is one discoverable city/state/asn/zip for a
// pool_type+country.
type ProxyLocationItem struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListProxyLocationsParams describes a proxy location discovery query.
// Not available for ProxyPoolResidentialPremium.
type ListProxyLocationsParams struct {
	// PoolType is "residential", "mobile", or "datacenter".
	PoolType string
	Country  string
	// Kind is "cities", "states", "asns", or "zipcodes".
	Kind string
}

// GenerateProxyEndpointParams composes a ready-to-use connection string.
// This is a CLIENT-SIDE, pure computation (no network call) that must stay
// byte-identical to the frontend's ProxyEndpointGenerator logic — see
// generateProxyUsername in proxies.go.
type GenerateProxyEndpointParams struct {
	ProxyID     string
	CountryCode string
	// TargetBy is "country" (default), "state", "city", "zip", or "asn".
	TargetBy string
	// LocationCode is required if TargetBy != "country".
	LocationCode string
	// Session is "rotating" (default) or "sticky".
	Session string
	// StickyTTLMinutes defaults to 10.
	StickyTTLMinutes int
	// Count defaults to 1, max 100.
	Count int
	// Protocol is "HTTP" (default) or "SOCKS5".
	Protocol string
	// Format is "host:port:user:pass" (default), "user:pass@host:port", or "curl".
	Format string
}

// ProxyEndpointResult is the outcome of GenerateProxyEndpoint.
type ProxyEndpointResult struct {
	ProxyID          string   `json:"proxy_id"`
	PoolType         string   `json:"pool_type"`
	Host             string   `json:"host"`
	Port             int      `json:"port"`
	Protocol         string   `json:"protocol"`
	Session          string   `json:"session"`
	StickyTTLMinutes int      `json:"sticky_ttl_minutes,omitempty"`
	CountryCode      string   `json:"country_code"`
	TargetBy         string   `json:"target_by"`
	LocationCode     string   `json:"location_code,omitempty"`
	Premium2x        bool     `json:"premium_2x"`
	Endpoints        []string `json:"endpoints"`
}

// ─── Session ────────────────────────────────────────────────────────────────

// StartSessionParams describes a manual registration browser session
// request. Beta, invite-only feature.
type StartSessionParams struct {
	ServiceName string
	Country     string
	// DeviceMode is "desktop" or "mobile".
	DeviceMode string
	// WithProxy defaults to true if Country is set.
	WithProxy *bool
	TargetURL string
	OrderID   string
	// Mode is "attach" or "fresh" (default fresh).
	Mode string
}

// BrowserSessionTimelineEvent is one entry in a BrowserSessionResult's
// timeline.
type BrowserSessionTimelineEvent struct {
	At     string `json:"at"`
	Event  string `json:"event"`
	Detail string `json:"detail,omitempty"`
}

// BrowserSessionResult describes a manual registration session. Only
// ViewerURL (our own proxied live-viewer link) is ever populated — no raw
// upstream debug URL is exposed.
type BrowserSessionResult struct {
	ID          string                        `json:"id"`
	Status      string                        `json:"status"`
	ServiceName string                        `json:"service_name,omitempty"`
	CountryCode string                        `json:"country_code,omitempty"`
	DeviceMode  string                        `json:"device_mode,omitempty"`
	WithProxy   bool                          `json:"with_proxy,omitempty"`
	ViewerURL   string                        `json:"viewer_url,omitempty"`
	TargetURL   string                        `json:"target_url,omitempty"`
	OrderID     string                        `json:"order_id,omitempty"`
	PhoneNumber string                        `json:"phone_number,omitempty"`
	Timeline    []BrowserSessionTimelineEvent `json:"timeline,omitempty"`
}

// ─── Tools ──────────────────────────────────────────────────────────────────

// NumberCheckResult is the outcome of CheckNumber.
type NumberCheckResult struct {
	Valid         bool   `json:"valid"`
	E164          string `json:"e164"`
	National      string `json:"national,omitempty"`
	CountryCode   string `json:"country_code"`
	CountryName   string `json:"country_name"`
	CountryPrefix string `json:"country_prefix,omitempty"`
	Location      string `json:"location,omitempty"`
	Carrier       string `json:"carrier,omitempty"`
	LineType      string `json:"line_type"`
	SpamRisk      string `json:"spam_risk"`
	Cached        bool   `json:"cached"`
	Message       string `json:"message,omitempty"`
}

// ─── Webhooks ───────────────────────────────────────────────────────────────

// WebhookEndpoint is a registered webhook subscription. Secret is only
// ever populated on the response to CreateWebhook — store it immediately,
// it is never returned again.
type WebhookEndpoint struct {
	ID                      string   `json:"id"`
	URL                     string   `json:"url"`
	Description             string   `json:"description,omitempty"`
	Events                  []string `json:"events"`
	Active                  bool     `json:"active"`
	Paused                  bool     `json:"paused"`
	Threshold               float64  `json:"threshold,omitempty"`
	FailureCountConsecutive int      `json:"failure_count_consecutive"`
	LastDeliveredAt         string   `json:"last_delivered_at,omitempty"`
	LastErrorAt             string   `json:"last_error_at,omitempty"`
	LastErrorCode           string   `json:"last_error_code,omitempty"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
	// Secret is populated only in the CreateWebhook response.
	Secret string `json:"secret,omitempty"`
}

// CreateWebhookParams describes a new webhook subscription.
type CreateWebhookParams struct {
	// URL is required, must be https://, no localhost/IP literals.
	URL         string
	Description string
	// Events is required, non-empty, subset of the allowed event types.
	Events []string
	// Threshold is required if Events includes "balance.low"; 0 < n <= 99999.99.
	Threshold float64
}

// UpdateWebhookParams is a partial update — at least one field must be set
// (use pointers to distinguish "not provided" from zero values).
type UpdateWebhookParams struct {
	URL         *string
	Description *string
	Events      []string
	Threshold   *float64
	Active      *bool
	Paused      *bool
}

// ListWebhooksResult is the outcome of ListWebhooks.
type ListWebhooksResult struct {
	Success  bool              `json:"success"`
	Webhooks []WebhookEndpoint `json:"webhooks"`
	Count    int               `json:"count"`
}

// GetWebhookResult is the outcome of GetWebhook.
type GetWebhookResult struct {
	Success bool            `json:"success"`
	Webhook WebhookEndpoint `json:"webhook"`
}

// DeleteWebhookResult is the outcome of DeleteWebhook.
type DeleteWebhookResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// TestWebhookResult is the outcome of TestWebhook.
type TestWebhookResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DeliveryID string `json:"delivery_id"`
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
}

// WebhookDelivery is one delivery attempt record.
type WebhookDelivery struct {
	ID             string `json:"id"`
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Attempt        int    `json:"attempt"`
	Status         string `json:"status"`
	ResponseStatus int    `json:"response_status,omitempty"`
	ResponseBody   string `json:"response_body,omitempty"`
	ScheduledFor   string `json:"scheduled_for,omitempty"`
	DeliveredAt    string `json:"delivered_at,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreatedAt      string `json:"created_at"`
	Payload        any    `json:"payload,omitempty"`
}

// ListWebhookDeliveriesParams paginates ListWebhookDeliveries.
type ListWebhookDeliveriesParams struct {
	// Limit defaults to 100, max 500.
	Limit  int
	Offset int
}

// ListWebhookDeliveriesResult is the outcome of ListWebhookDeliveries.
type ListWebhookDeliveriesResult struct {
	Success    bool              `json:"success"`
	Deliveries []WebhookDelivery `json:"deliveries"`
	Count      int               `json:"count"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// AllowedWebhookEvents lists the event types the VirtualSMS webhooks API
// accepts. Kept here as a convenience for validation before a round-trip;
// the backend is the ultimate source of truth and may add new types.
var AllowedWebhookEvents = []string{
	"order.created",
	"order.sms_received",
	"order.cancelled",
	"order.expired",
	"rental.created",
	"rental.sms_received",
	"rental.expired",
	"rental.cancelled",
	"proxy.purchased",
	"balance.low",
}
