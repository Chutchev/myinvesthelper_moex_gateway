package cbr

import (
	"net/http"
	"testing"
)

func TestNewHTTPClientUsesDefaultClientWhenNil(t *testing.T) {
	client := NewHTTPClient("https://cbr.test/rate", "https://cbr.test/forecast", nil)
	if client.client != http.DefaultClient {
		t.Fatalf("client = %p, want http.DefaultClient %p", client.client, http.DefaultClient)
	}
}
