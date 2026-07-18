package virtualsms

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetBalance checks the account's current balance.
func (c *VirtualSMS) GetBalance(ctx context.Context) (*Balance, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var b Balance
	if err := c.do(ctx, "GET", "/customer/balance", nil, nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetProfile returns the full account profile.
func (c *VirtualSMS) GetProfile(ctx context.Context) (*Profile, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	var p Profile
	if err := c.do(ctx, "GET", "/customer/profile", nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetTransactions returns a paginated transaction history page.
func (c *VirtualSMS) GetTransactions(ctx context.Context, params GetTransactionsParams) (*TransactionsPage, error) {
	if err := c.requireAPIKey(); err != nil {
		return nil, err
	}
	q := url.Values{}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	if params.From != "" {
		q.Set("from", params.From)
	}
	if params.To != "" {
		q.Set("to", params.To)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(params.Offset))

	var page TransactionsPage
	if err := c.do(ctx, "GET", "/customer/transactions", q, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetStats aggregates usage stats over a lookback window (default 30
// days). Client-side: calls GetBalance + ListOrders, then aggregates
// locally (status/service/country breakdowns, spend excluding cancelled
// orders, success rate over terminal-state orders only).
func (c *VirtualSMS) GetStats(ctx context.Context, sinceDays int) (*Stats, error) {
	if sinceDays <= 0 {
		sinceDays = 30
	}
	cutoff := time.Now().Add(-time.Duration(sinceDays) * 24 * time.Hour)

	balance, err := c.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	orders, err := c.ListOrders(ctx, "")
	if err != nil {
		return nil, err
	}

	var inWindow []Order
	for _, o := range orders {
		t, ok := parseOrderTime(o.CreatedAt)
		if ok && !t.Before(cutoff) {
			inWindow = append(inWindow, o)
		}
	}

	byStatus := map[string]int{}
	byService := map[string]int{}
	byCountry := map[string]int{}
	var totalSpend float64
	var successful, terminal int

	terminalStatuses := map[string]bool{"completed": true, "sms_received": true, "expired": true, "cancelled": true}

	for _, o := range inWindow {
		byStatus[o.Status]++
		if o.Service != "" {
			byService[o.Service]++
		}
		if o.Country != "" {
			byCountry[o.Country]++
		}
		if o.Status != "cancelled" {
			totalSpend += o.Price
		}
		if terminalStatuses[o.Status] {
			terminal++
			if o.Status == "completed" || o.Status == "sms_received" {
				successful++
			}
		}
	}

	var successRate *float64
	if terminal > 0 {
		r := roundTo1dp(float64(successful) / float64(terminal) * 100)
		successRate = &r
	}

	stats := &Stats{
		WindowDays:       sinceDays,
		BalanceUSD:       balance.BalanceUSD,
		TotalOrders:      len(inWindow),
		SuccessfulOrders: successful,
		SuccessRate:      successRate,
		TotalSpendUSD:    roundTo2dp(totalSpend),
		StatusBreakdown:  byStatus,
		TopServices:      topEntries(byService, 5),
		TopCountries:     topEntries(byCountry, 5),
	}
	if len(orders) >= 50 {
		stats.Note = "Server caps order history at 50 rows. Stats may undercount if your activity exceeds 50 orders in the window."
	}
	return stats, nil
}

func roundTo1dp(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

func topEntries(m map[string]int, n int) []StatsEntry {
	entries := make([]StatsEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, StatsEntry{Key: k, Count: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Key < entries[j].Key // stable/deterministic tiebreak
	})
	if len(entries) > n {
		entries = entries[:n]
	}
	return entries
}

// ─── check_number (public tool) ────────────────────────────────────────────

// CheckNumber does a carrier + line-type lookup for an arbitrary E.164
// number (e.g. "+447911123456"). Public endpoint, no API key required.
func (c *VirtualSMS) CheckNumber(ctx context.Context, number string) (*NumberCheckResult, error) {
	if strings.TrimSpace(number) == "" {
		return nil, fmt.Errorf("virtualsms: number is required")
	}
	q := url.Values{"number": {number}}
	var result NumberCheckResult
	if err := c.do(ctx, "GET", "/tools/number-check", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
