package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/courtyard-nft/go-shippo/models"
)

func TestNewClientWithBaseURL_UsesGivenBaseURL(t *testing.T) {
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost {
			t.Errorf("method = %q; want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/addresses/" {
			t.Errorf("path = %q; want %q", r.URL.Path, "/addresses/")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := NewClientWithBaseURL("test-token", "", server.URL+"/").CreateAddress(&models.AddressInput{
		Street1: "123 Main St",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !received {
		t.Fatal("server did not receive request")
	}
}
