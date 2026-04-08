// monad-exporter exposes monad-status.sh–derived metrics for Prometheus.
//
// Environment:
//   MONAD_EXPORTER_LISTEN — listen address (default :9100). PORT is also honored (e.g. 9100 → :9100).
//   MONAD_HOME — default /home/monad
//   METRICS_URL — default http://127.0.0.1:8889/metrics
//   RPC_URL — default http://127.0.0.1:8080
//   EXTERNAL_RPC_TEMPLATE — default https://rpc-%s.monadinfra.com
//   CONTROLPANEL_SOCK — default $MONAD_HOME/monad-bft/controlpanel.sock
//   MPT_STORAGE — default /dev/triedb
//   RPC_LISTEN_PORT — port checked with ss (default 8080)
//   MONAD_EXPORTER_ENABLE_KEYSTORE — set to 1 to recover Secp pubkey via monad-keystore once at startup
//   MONAD_EXPORTER_SECP_PUBLIC_KEY — optional hex pubkey (skips monad-keystore if set when keystore enabled)
//   MONAD_EXPORTER_HTTP_TIMEOUT — default 15s
//   MONAD_EXPORTER_ENABLE_STAKING — set to 1 to query staking precompile via eth_call (RPC_URL)
//   MONAD_STAKING_ADDRESS — default 0x0000000000000000000000000000000000001000
//   VALIDATOR_ID — validator id for getValidator-based staking metrics (monad_exporter_staking_validator_*, active, info)
//
// Flags:
//   -config path — load environment variables from a .env file (does not override existing env vars).
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"monad-exporter/internal/collector"
	"monad-exporter/internal/config"
)

func main() {
	envFile := flag.String("config", "", "path to .env file (optional)")
	flag.Parse()
	if *envFile != "" {
		if err := config.LoadEnvFile(*envFile); err != nil {
			log.Fatalf("load env file %s: %v", *envFile, err)
		}
	}

	cfg := config.Load()
	if v := strings.TrimSpace(os.Getenv("MONAD_EXPORTER_SECP_PUBLIC_KEY")); v != "" {
		cfg.SecpPublicKey = v
	} else if cfg.EnableKeystore {
		cfg.SecpPublicKey = collector.LoadSecpPublicKey(cfg)
		if cfg.SecpPublicKey == "" {
			log.Printf("warning: MONAD_EXPORTER_ENABLE_KEYSTORE=1 but Secp public key is empty (set MONAD_EXPORTER_SECP_PUBLIC_KEY or KEYSTORE_PASSWORD in %s/.env and monad-keystore)", cfg.MonadHome)
		}
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewNodeCollector(cfg))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		Registry: reg,
	}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	addr := cfg.ListenAddr
	log.Printf("monad-exporter listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("server exit: %v", err)
		log.Printf("hint: choose another port with PORT=9101 or MONAD_EXPORTER_LISTEN=:9101, or stop the process already bound to %s", addr)
		os.Exit(1)
	}
}
