package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var usdCache = struct {
	sync.Mutex
	baseURL, coinID string
	ttl             time.Duration
	val             float64
	at              time.Time
}{}

// ParseSimplePriceResponse extracts USD price for coinID from CoinGecko /simple/price JSON.
func ParseSimplePriceResponse(data []byte, coinID string) (float64, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return 0, err
	}
	rawInner, ok := top[coinID]
	if !ok || len(rawInner) == 0 {
		return 0, fmt.Errorf("missing coin %q in response", coinID)
	}
	var inner struct {
		USD float64 `json:"usd"`
	}
	if err := json.Unmarshal(rawInner, &inner); err != nil {
		return 0, err
	}
	return inner.USD, nil
}

// USDPrice returns spot price in USD using GET /simple/price, with in-memory TTL caching.
// On fetch failure, returns the last cached value for the same baseURL/coinID if any.
func USDPrice(ctx context.Context, httpClient *http.Client, baseURL, coinID string, ttl time.Duration) (float64, error) {
	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return 0, fmt.Errorf("empty coin id")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	now := time.Now()

	usdCache.Lock()
	match := usdCache.baseURL == baseURL && usdCache.coinID == coinID && usdCache.ttl == ttl
	if match && ttl > 0 && !usdCache.at.IsZero() && now.Sub(usdCache.at) < ttl {
		v := usdCache.val
		usdCache.Unlock()
		return v, nil
	}
	usdCache.Unlock()

	u, err := url.Parse(baseURL)
	if err != nil {
		return 0, fmt.Errorf("coingecko base url: %w", err)
	}
	u = u.JoinPath("simple", "price")
	q := u.Query()
	q.Set("ids", coinID)
	q.Set("vs_currencies", "usd")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return staleOrErr(baseURL, coinID, ttl, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return staleOrErr(baseURL, coinID, ttl, err)
	}
	if resp.StatusCode != http.StatusOK {
		return staleOrErr(baseURL, coinID, ttl, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}
	price, err := ParseSimplePriceResponse(body, coinID)
	if err != nil {
		return staleOrErr(baseURL, coinID, ttl, err)
	}

	usdCache.Lock()
	usdCache.baseURL = baseURL
	usdCache.coinID = coinID
	usdCache.ttl = ttl
	usdCache.val = price
	usdCache.at = now
	usdCache.Unlock()
	return price, nil
}

func staleOrErr(baseURL, coinID string, ttl time.Duration, err error) (float64, error) {
	usdCache.Lock()
	defer usdCache.Unlock()
	if usdCache.baseURL == baseURL && usdCache.coinID == coinID && usdCache.ttl == ttl && !usdCache.at.IsZero() {
		return usdCache.val, nil
	}
	return 0, err
}
