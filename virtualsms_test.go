package virtualsms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestSmoke_GetBalance_ListServices_GetPrice is the minimum smoke test
// required by the SDK spec: get_balance + list_services succeed against a
// throwaway key. It also exercises get_price's two-call fail-closed
// pattern, since that's the highest-risk piece of client-side logic in
// this SDK.
//
// Runs against a local httptest server that mimics the three endpoints
// exactly (status codes + response shapes) rather than the live API, so it
// passes in CI without a real API key. If VIRTUALSMS_TEST_API_KEY and
// VIRTUALSMS_TEST_BASE_URL are both set in the environment, it runs
// against the real (or sandbox) API instead.
func TestSmoke_GetBalance_ListServices_GetPrice(t *testing.T) {
	apiKey := os.Getenv("VIRTUALSMS_TEST_API_KEY")
	baseURL := os.Getenv("VIRTUALSMS_TEST_BASE_URL")

	if apiKey == "" || baseURL == "" {
		apiKey = "test-key"
		srv := newMockServer(t)
		defer srv.Close()
		baseURL = srv.URL + "/api/v1"
	}

	client := New(apiKey, WithBaseURL(baseURL))
	ctx := context.Background()

	balance, err := client.GetBalance(ctx)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	if balance.BalanceUSD < 0 {
		t.Fatalf("GetBalance returned negative balance: %v", balance.BalanceUSD)
	}

	services, err := client.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) == 0 {
		t.Fatal("ListServices returned zero services")
	}

	price, err := client.GetPrice(ctx, "telegram", "US")
	if err != nil {
		t.Fatalf("GetPrice failed: %v", err)
	}
	if price.PriceUSD <= 0 {
		t.Fatalf("GetPrice returned non-positive price: %v", price.PriceUSD)
	}
	if !price.Available {
		t.Fatal("GetPrice expected Available=true for the mocked in-stock combo")
	}
}

// TestGetPrice_FailClosed verifies GetPrice never reports Available=true
// when the catalog shows zero stock — the fail-closed contract this SDK
// must replicate from the MCP server (see orders.go GetPrice doc comment).
func TestGetPrice_FailClosed(t *testing.T) {
	srv := newMockServer(t)
	defer srv.Close()

	client := New("test-key", WithBaseURL(srv.URL+"/api/v1"))
	price, err := client.GetPrice(context.Background(), "outofstock", "US")
	if err != nil {
		t.Fatalf("GetPrice failed: %v", err)
	}
	if price.Available {
		t.Fatal("GetPrice must report Available=false when catalog count is 0")
	}
}

func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/customer/balance", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"balance_usd": 12.5})
	})

	mux.HandleFunc("/api/v1/customer/services", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"success": true,
			"services": []map[string]any{
				{"service_id": "telegram", "service_name": "Telegram"},
				{"service_id": "whatsapp", "service_name": "WhatsApp"},
			},
		})
	})

	mux.HandleFunc("/api/v1/price", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"price": 0.9, "currency": "USD", "country": r.URL.Query().Get("country"), "service": r.URL.Query().Get("service")})
	})

	mux.HandleFunc("/api/v1/catalog/countries", func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		count := 9240
		if service == "outofstock" {
			count = 0
		}
		writeJSON(w, map[string]any{
			"success": true,
			"countries": []map[string]any{
				{"id": "US", "name": "United States", "price": 0.9, "count": count},
			},
		})
	})

	return httptest.NewServer(mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
