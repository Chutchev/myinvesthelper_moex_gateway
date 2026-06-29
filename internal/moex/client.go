package moex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

const (
	userAgent        = "myinvesthelper-gateway/1.0"
	universeTimeout  = 20 * time.Second
	maxResponseBytes = 16 << 20 // 16 MiB
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
	timeout time.Duration
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return newHTTPClient(baseURL, &http.Client{}, timeout)
}

func newHTTPClient(baseURL string, client *http.Client, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		timeout: timeout,
	}
}

func (c *HTTPClient) FetchUniverse(ctx context.Context, limit int) ([]byte, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("securities.columns", "SECID,SHORTNAME,LOTSIZE,FACEVALUE,FACEUNIT,MATDATE,COUPONPERCENT,COUPONVALUE,ISSUESIZE")
	query.Set("marketdata.columns", "SECID,LAST,LCURRENTPRICE,YIELD,DURATION,ACCRUEDINT,VALTODAY,NUMTRADES,SYSTIME")

	path := "/engines/stock/markets/bonds/boards/TQCB/securities.json"
	return c.get(ctx, path, query, universeTimeout)
}

func (c *HTTPClient) FetchDescription(ctx context.Context, isin string) ([]byte, error) {
	path := fmt.Sprintf("/securities/%s.json", isin)
	return c.get(ctx, path, nil, c.timeout)
}

func (c *HTTPClient) FetchMarketData(ctx context.Context, isin string) ([]byte, error) {
	path := fmt.Sprintf("/engines/stock/markets/bonds/securities/%s.json", isin)
	return c.get(ctx, path, nil, c.timeout)
}

func (c *HTTPClient) FetchBondization(ctx context.Context, isin string) ([]byte, error) {
	path := fmt.Sprintf("/statistics/engines/stock/markets/bonds/bondization/%s.json", isin)
	return c.get(ctx, path, nil, c.timeout)
}

func (c *HTTPClient) get(ctx context.Context, path string, query url.Values, timeout time.Duration) ([]byte, error) {
	url := c.baseURL + path
	if query != nil && len(query) > 0 {
		url += "?" + query.Encode()
	}

	for attempt := 1; attempt <= 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("%w: create request: %v", apperrors.ErrHTTPError, err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.client.Do(req)
		if err != nil {
			cancel()
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("%w: %s", apperrors.ErrTimeout, url)
			}
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			if attempt == 1 {
				continue
			}
			return nil, fmt.Errorf("request failed: %v", err)
		}

		if resp.StatusCode >= 500 && attempt == 1 {
			_ = resp.Body.Close()
			cancel()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			cancel()
			return nil, fmt.Errorf("%w: status %d", apperrors.ErrHTTPError, resp.StatusCode)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		_ = resp.Body.Close()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("%w: read body: %v", apperrors.ErrHTTPError, err)
		}
		if len(body) > maxResponseBytes {
			return nil, fmt.Errorf("%w: response exceeds %d bytes", apperrors.ErrHTTPError, maxResponseBytes)
		}
		return body, nil
	}
	return nil, fmt.Errorf("%w: max retries reached", apperrors.ErrHTTPError)
}
