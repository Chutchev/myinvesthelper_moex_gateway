package moex

import (
	"net/http"
	"testing"
)

func TestNewHTTPClientUsesDefaultClientWhenNil(t *testing.T) {
	client := NewHTTPClient("https://moex.test", nil)
	if client.client != http.DefaultClient {
		t.Fatalf("client = %p, want http.DefaultClient %p", client.client, http.DefaultClient)
	}
}
