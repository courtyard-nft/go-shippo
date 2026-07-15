package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/courtyard-nft/go-shippo/models"
)

func newTestTransaction(id string, created time.Time) json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"object_id":      id,
		"object_created": created.Format(time.RFC3339),
	})
	return data
}

// newTransactionListServer serves /transactions/ pages of the given results,
// chaining each page to the next via the "next" URL. It returns the server
// and a counter of pages requested.
func newTransactionListServer(t *testing.T, pages [][]json.RawMessage) (*httptest.Server, *int) {
	t.Helper()
	pagesRequested := 0

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/transactions/", func(w http.ResponseWriter, r *http.Request) {
		page := 0
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		if page >= len(pages) {
			t.Errorf("requested page %d beyond available pages", page)
			http.NotFound(w, r)
			return
		}
		pagesRequested++

		output := models.ListAPIOutput{Results: pages[page]}
		if page < len(pages)-1 {
			next := fmt.Sprintf("%s/transactions/?results=100&page=%d", server.URL, page+1)
			output.NextPageURL = &next
		}
		if err := json.NewEncoder(w).Encode(output); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	})

	return server, &pagesRequested
}

func newTestClient(baseURL string) *Client {
	c := NewClient("test-token", "")
	c.baseURL = baseURL
	return c
}

func TestListTransactionsCreatedAfter_StopsAtCutoffWithoutFetchingOlderPages(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cutoff := now.AddDate(0, 0, -90)

	pages := [][]json.RawMessage{
		{
			newTestTransaction("tx-newest", now),
			newTestTransaction("tx-recent", now.AddDate(0, 0, -30)),
		},
		{
			newTestTransaction("tx-boundary", now.AddDate(0, 0, -89)),
			newTestTransaction("tx-at-cutoff", cutoff),
			newTestTransaction("tx-old", now.AddDate(0, 0, -120)),
		},
		{
			newTestTransaction("tx-ancient", now.AddDate(0, 0, -365)),
		},
	}

	server, pagesRequested := newTransactionListServer(t, pages)
	defer server.Close()

	txs, err := newTestClient(server.URL).ListTransactionsCreatedAfter(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txs) != 3 {
		t.Fatalf("got %d transactions; want 3", len(txs))
	}
	wantIDs := []string{"tx-newest", "tx-recent", "tx-boundary"}
	for i, want := range wantIDs {
		if txs[i].ObjectID != want {
			t.Errorf("txs[%d].ObjectID = %q; want %q", i, txs[i].ObjectID, want)
		}
	}
	if *pagesRequested != 2 {
		t.Errorf("requested %d pages; want 2 (should stop before fetching older pages)", *pagesRequested)
	}
}

func TestListTransactionsCreatedAfter_PaginatesAllPagesWhenAllNewerThanCutoff(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cutoff := now.AddDate(0, 0, -90)

	pages := [][]json.RawMessage{
		{newTestTransaction("tx-1", now)},
		{newTestTransaction("tx-2", now.AddDate(0, 0, -10))},
		{newTestTransaction("tx-3", now.AddDate(0, 0, -20))},
	}

	server, pagesRequested := newTransactionListServer(t, pages)
	defer server.Close()

	txs, err := newTestClient(server.URL).ListTransactionsCreatedAfter(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txs) != 3 {
		t.Fatalf("got %d transactions; want 3", len(txs))
	}
	if *pagesRequested != 3 {
		t.Errorf("requested %d pages; want 3", *pagesRequested)
	}
}

func TestListTransactionsCreatedAfter_ReturnsEmptyListWhenFirstTransactionIsOld(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cutoff := now.AddDate(0, 0, -90)

	pages := [][]json.RawMessage{
		{newTestTransaction("tx-old", now.AddDate(0, 0, -120))},
		{newTestTransaction("tx-older", now.AddDate(0, 0, -150))},
	}

	server, pagesRequested := newTransactionListServer(t, pages)
	defer server.Close()

	txs, err := newTestClient(server.URL).ListTransactionsCreatedAfter(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txs) != 0 {
		t.Fatalf("got %d transactions; want 0", len(txs))
	}
	if *pagesRequested != 1 {
		t.Errorf("requested %d pages; want 1", *pagesRequested)
	}
}

func TestListTransactionsCreatedAfter_ReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream connect error", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := newTestClient(server.URL).ListTransactionsCreatedAfter(context.Background(), time.Now().AddDate(0, 0, -90))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
