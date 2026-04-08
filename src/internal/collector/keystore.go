package collector

import (
	"context"
	"os/exec"
	"strings"

	"monad-exporter/internal/config"
	"monad-exporter/internal/parsefiles"
)

// LoadSecpPublicKey runs monad-keystore once (password from $MONAD_HOME/.env). Call at startup only.
func LoadSecpPublicKey(cfg *config.Config) string {
	envPath := cfg.MonadHome + "/.env"
	pass := parsefiles.EnvValue(envPath, "KEYSTORE_PASSWORD")
	if pass == "" {
		return ""
	}
	args := []string{
		"recover",
		"--password", pass,
		"--keystore-path", cfg.MonadHome + "/monad-bft/config/id-secp",
		"--key-type", "secp",
	}
	cctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "monad-keystore", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Secp public key") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	}
	return ""
}
