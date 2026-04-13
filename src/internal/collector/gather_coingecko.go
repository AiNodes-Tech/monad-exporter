package collector

import (
	"context"
	"net/http"

	"monad-exporter/internal/coingecko"
	"monad-exporter/internal/config"
)

func gatherCoinGecko(ctx context.Context, cfg *config.Config, out *Snapshot, client *http.Client) {
	if cfg.CoinGeckoCoinID == "" {
		out.errf("coingecko", "empty MONAD_EXPORTER_COINGECKO_COIN_ID")
		return
	}
	p, err := coingecko.USDPrice(ctx, client, cfg.CoinGeckoBaseURL, cfg.CoinGeckoCoinID, cfg.CoinGeckoCacheTTL)
	if err != nil {
		out.errf("coingecko", "%v", err)
		return
	}
	out.CoinGeckoPriceUSDOK = true
	out.CoinGeckoPriceUSD = p
}
