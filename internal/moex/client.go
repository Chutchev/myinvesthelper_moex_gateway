package moex

import (
	"context"
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type Client interface {
	FetchUniverse(ctx context.Context, limit int) ([]byte, error)
	FetchDescription(ctx context.Context, isin string) ([]byte, error)
	FetchMarketData(ctx context.Context, isin string) ([]byte, error)
	FetchBondization(ctx context.Context, isin string) ([]byte, error)
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string, client *http.Client) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPClient{baseURL: baseURL, client: client}
}

func (c *HTTPClient) FetchUniverse(context.Context, int) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

func (c *HTTPClient) FetchDescription(context.Context, string) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

func (c *HTTPClient) FetchMarketData(context.Context, string) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

func (c *HTTPClient) FetchBondization(context.Context, string) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}
