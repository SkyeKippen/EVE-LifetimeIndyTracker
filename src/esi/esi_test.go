package esi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeESI answers with the given body per path, splitting "paged" routes
// into two pages.
func fakeESI(t *testing.T, routes map[string]string, paged map[string][2]string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" && r.URL.Path != "/universe/structures" {
			t.Errorf("%s: missing token", r.URL.Path)
		}
		if pages, ok := paged[r.URL.Path]; ok {
			w.Header().Set("X-Pages", "2")
			if r.URL.Query().Get("page") == "2" {
				fmt.Fprint(w, pages[1])
			} else {
				fmt.Fprint(w, pages[0])
			}
			return
		}
		body, ok := routes[r.URL.Path]
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		if body == "403" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL}
}

func TestCorporationWalletJournalReadsEveryPage(t *testing.T) {
	entry := `{"amount":-256000000,"balance":336717024.8701,"context_id":6893305979,"context_id_type":"market_transaction_id","date":"2026-10-13T22:20:25Z","description":"Market escrow release","first_party_id":2122970353,"id":%d,"reason":"","ref_type":"market_escrow","second_party_id":2122970353}`
	c := fakeESI(t, nil, map[string][2]string{
		"/corporations/98000001/wallets/1/journal": {"[" + fmt.Sprintf(entry, 1) + "]", "[" + fmt.Sprintf(entry, 2) + "]"},
	})
	entries, err := c.GetCorporationWalletJournal(context.Background(), 98000001, 1, "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Id != 2 || entries[0].Balance != 336717024.8701 || entries[0].RefType != "market_escrow" {
		t.Fatalf("got %+v", entries)
	}
}

func TestBalancesLoyaltyAndDivisions(t *testing.T) {
	c := fakeESI(t, map[string]string{
		"/characters/1/wallet":         "1234.56",
		"/corporations/2/wallets":      `[{"division":1,"balance":10.5},{"division":2,"balance":0}]`,
		"/characters/1/loyalty/points": `[{"corporation_id":1000125,"loyalty_points":5000}]`,
		"/corporations/2/divisions":    `{"hangar":[{"division":1,"name":"Main"}],"wallet":[{"division":1,"name":"Master Wallet"}]}`,
	}, nil)
	ctx := context.Background()
	if balance, err := c.GetCharacterWalletBalance(ctx, 1, "token"); err != nil || balance != 1234.56 {
		t.Fatalf("character balance %v, %v", balance, err)
	}
	if balances, err := c.GetCorporationWalletBalances(ctx, 2, "token"); err != nil || len(balances) != 2 || balances[0].Balance != 10.5 {
		t.Fatalf("corporation balances %+v, %v", balances, err)
	}
	if lp, err := c.GetCharacterLoyaltyPoints(ctx, 1, "token"); err != nil || lp[0].LoyaltyPoints != 5000 {
		t.Fatalf("loyalty %+v, %v", lp, err)
	}
	if divisions, err := c.GetCorporationDivisions(ctx, 2, "token"); err != nil || divisions.Wallet[0].Name != "Master Wallet" {
		t.Fatalf("divisions %+v, %v", divisions, err)
	}
}

func TestCharacterJobsUseStationAsLocation(t *testing.T) {
	c := fakeESI(t, map[string]string{
		"/characters/1/industry/jobs": `[{"job_id":5,"activity_id":1,"status":"delivered","station_id":1035466617946,"facility_id":1035466617946,"start_date":"2026-10-01T00:00:00Z","end_date":"2026-10-02T00:00:00Z"}]`,
	}, nil)
	jobs, err := c.GetCharacterIndustryJobs(context.Background(), 1, "token")
	if err != nil {
		t.Fatal(err)
	}
	if jobs[0].LocationId != 1035466617946 {
		t.Fatalf("location %d", jobs[0].LocationId)
	}
}

func TestCorporationMiningLedgerReadsEachObserver(t *testing.T) {
	c := fakeESI(t, map[string]string{
		"/corporation/2/mining/observers":               `[{"observer_id":1035000000001,"observer_type":"structure","last_updated":"2026-10-13"}]`,
		"/corporation/2/mining/observers/1035000000001": `[{"character_id":1,"recorded_corporation_id":2,"type_id":45490,"quantity":1000,"last_updated":"2026-10-13"}]`,
	}, nil)
	ledgers, err := c.GetCorporationMiningLedger(context.Background(), 2, "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(ledgers) != 1 || ledgers[0].Entries[0].Quantity != 1000 {
		t.Fatalf("got %+v", ledgers)
	}
}

func TestAccessibleStructuresSkipsForbidden(t *testing.T) {
	c := fakeESI(t, map[string]string{
		"/universe/structures/1035000000001": `{"name":"Home Base","owner_id":2,"solar_system_id":30000142,"type_id":35827}`,
		"/universe/structures/1035000000002": "403",
	}, nil)
	assets := []Asset{
		{ItemId: 1050000000000, LocationId: 1035000000001}, // container in a structure
		{ItemId: 1050000000001, LocationId: 1050000000000}, // item in that container
		{ItemId: 1050000000002, LocationId: 60003760},      // NPC station
	}
	orders := []MarketOrder{{LocationId: 1035000000002}}
	ids := StructureIdsOf(assets, orders, nil)
	if len(ids) != 2 {
		t.Fatalf("candidates %v", ids)
	}
	structures, err := c.GetAccessibleStructures(context.Background(), ids, "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(structures) != 1 || structures[0].Name != "Home Base" || structures[0].StructureId != 1035000000001 {
		t.Fatalf("got %+v", structures)
	}
}

func TestForbiddenIsReported(t *testing.T) {
	c := fakeESI(t, map[string]string{"/corporations/2/assets": "403"}, nil)
	_, err := c.GetCorporationAssets(context.Background(), 2, "token")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}
