package cbr

import (
	"context"
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type Client interface {
	FetchKeyRatePage(ctx context.Context) ([]byte, error)
	FetchForecastPage(ctx context.Context) ([]byte, error)
	FetchForecastWorkbook(ctx context.Context, url string) ([]byte, error)
}

type HTTPClient struct {
	keyRateURL  string
	forecastURL string
	client      *http.Client
}

func NewHTTPClient(keyRateURL, forecastURL string, client *http.Client) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPClient{
		keyRateURL:  keyRateURL,
		forecastURL: forecastURL,
		client:      client,
	}
}

func (c *HTTPClient) FetchKeyRatePage(context.Context) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

func (c *HTTPClient) FetchForecastPage(context.Context) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

func (c *HTTPClient) FetchForecastWorkbook(context.Context, string) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}
