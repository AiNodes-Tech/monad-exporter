package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime settings from the environment (defaults match monad-status.sh).
type Config struct {
	ListenAddr          string
	MonadHome           string
	MetricsURL          string
	RPCURL              string
	ExternalRPCTemplate string
	HTTPTimeout         time.Duration
	ControlPanelSock    string
	MPTStorage          string
	RPCListenPort       string
	EnableKeystore      bool
	// SecpPublicKey is set once at process startup when EnableKeystore is true (see main).
	SecpPublicKey string

	// Staking (Monad precompile via eth_call)
	EnableStaking  bool
	StakingAddress string
	ValidatorID    uint64

	// CoinGecko (optional spot price for MON vs USD)
	EnableCoinGecko   bool
	CoinGeckoBaseURL  string
	CoinGeckoCoinID   string
	CoinGeckoCacheTTL time.Duration
}

func Load() *Config {
	c := &Config{
		ListenAddr:          firstNonEmpty(os.Getenv("MONAD_EXPORTER_LISTEN"), normalizePort(os.Getenv("PORT")), ":9100"),
		MonadHome:           firstNonEmpty(os.Getenv("MONAD_HOME"), "/home/monad"),
		MetricsURL:          firstNonEmpty(os.Getenv("METRICS_URL"), "http://127.0.0.1:8889/metrics"),
		RPCURL:              firstNonEmpty(os.Getenv("RPC_URL"), "http://127.0.0.1:8080"),
		ExternalRPCTemplate: firstNonEmpty(os.Getenv("EXTERNAL_RPC_TEMPLATE"), "https://rpc-%s.monadinfra.com"),
		HTTPTimeout:         15 * time.Second,
		ControlPanelSock:    "",
		MPTStorage:          firstNonEmpty(os.Getenv("MPT_STORAGE"), "/dev/triedb"),
		RPCListenPort:       firstNonEmpty(os.Getenv("RPC_LISTEN_PORT"), "8080"),
		EnableKeystore:      os.Getenv("MONAD_EXPORTER_ENABLE_KEYSTORE") == "1",
	}
	if v := os.Getenv("MONAD_EXPORTER_HTTP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.HTTPTimeout = d
		}
	}
	if v := os.Getenv("CONTROLPANEL_SOCK"); v != "" {
		c.ControlPanelSock = v
	} else {
		c.ControlPanelSock = c.MonadHome + "/monad-bft/controlpanel.sock"
	}
	c.ListenAddr = normalizeAddr(c.ListenAddr)

	c.EnableStaking = os.Getenv("MONAD_EXPORTER_ENABLE_STAKING") == "1"
	c.StakingAddress = firstNonEmpty(os.Getenv("MONAD_STAKING_ADDRESS"), "0x0000000000000000000000000000000000001000")
	if v := os.Getenv("VALIDATOR_ID"); v != "" {
		if id, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64); err == nil {
			c.ValidatorID = id
		}
	}

	c.EnableCoinGecko = os.Getenv("MONAD_EXPORTER_ENABLE_COINGECKO") == "1"
	c.CoinGeckoBaseURL = strings.TrimSuffix(firstNonEmpty(os.Getenv("MONAD_EXPORTER_COINGECKO_BASE_URL"), "https://api.coingecko.com/api/v3"), "/")
	c.CoinGeckoCoinID = strings.TrimSpace(firstNonEmpty(os.Getenv("MONAD_EXPORTER_COINGECKO_COIN_ID"), "monad"))
	c.CoinGeckoCacheTTL = 90 * time.Second
	if v := os.Getenv("MONAD_EXPORTER_COINGECKO_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.CoinGeckoCacheTTL = d
		}
	}

	return c
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func normalizePort(port string) string {
	if port == "" {
		return ""
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	if _, err := strconv.Atoi(port); err == nil {
		return ":" + port
	}
	return port
}

func normalizeAddr(s string) string {
	if s == "" {
		return ":9100"
	}
	if strings.Contains(s, ":") {
		return s
	}
	if _, err := strconv.Atoi(s); err == nil {
		return ":" + s
	}
	return s
}
