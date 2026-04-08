package collector

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"monad-exporter/internal/config"
	"monad-exporter/internal/executil"
	"monad-exporter/internal/mpt"
	"monad-exporter/internal/parsefiles"
	"monad-exporter/internal/promtext"
	"monad-exporter/internal/rpc"
)

// Snapshot holds one scrape of node state (mirrors monad-status.sh sources).
type Snapshot struct {
	ScrapeErrors map[string]string

	UptimeSeconds float64
	UptimeOK      bool

	StatesyncSyncing          float64
	StatesyncProgressEstimate float64
	StatesyncLastTarget       float64
	StatesyncProgressOK       bool
	StatesyncPercentage       float64

	LocalBlockRaw string
	LocalBlock    uint64
	LocalBlockOK  bool

	ExternalBlock   uint64
	ExternalBlockOK bool

	BlockDiff    float64
	BlockDiffOK  bool
	InSync       float64 // 1 or 0
	InSyncOK     bool

	Peers   float64
	PeersOK bool

	ForkEpoch   float64
	ForkRound   float64
	ForkEpochOK bool
	ForkRoundOK bool

	RPCListenActive float64 // 1 or 0
	RPCEthBlock     uint64
	RPCEthBlockOK   bool
	ClientVersion   string
	NetVersion      string

	Hostname       string
	Network        string
	Chain          string
	ChainID        string
	PackageVersion string
	SecpPubkey     string
	BLSPubkey      string
	NodeSignature  string
	SelfAddress    string
	RecordSeqNum   string
	SelfAuthPort   string

	ServiceBft       float64
	ServiceExecution float64
	ServiceRpc       float64
	ServiceArchiver  float64
	ServiceOtel      float64

	TrieDevice string
	TrieModel  string

	MPT *mpt.Info

	CruftActivated float64
	CruftPrevious  string
	CruftNext      string

	// Staking (eth_call to Monad staking precompile; MONAD_EXPORTER_ENABLE_STAKING=1)
	StakingExecutionCountOK       bool
	StakingConsensusCountOK       bool
	StakingValidatorsExecution    float64
	StakingValidatorsConsensus    float64
	StakingValidatorsInactiveExec float64
	StakingValidatorOK           bool // getValidator
	StakingValidatorPoolStakeMON float64
	StakingCommissionRatio        float64
	StakingValidatorUnclaimedRewardsMON float64
	StakingValidatorActiveConsensus     float64 // 1 or 0: id in getConsensusValidatorSet
	StakingValidatorActiveConsensusOK   bool
	StakingValidatorAuthAddress         string // getValidator authAddress (hex)
}

func newSnapshot() *Snapshot {
	return &Snapshot{ScrapeErrors: make(map[string]string)}
}

func (s *Snapshot) errf(source, format string, args ...interface{}) {
	s.ScrapeErrors[source] = fmt.Sprintf(format, args...)
}

func Gather(ctx context.Context, cfg *config.Config) *Snapshot {
	out := newSnapshot()
	if h, err := os.Hostname(); err == nil {
		out.Hostname = h
	}

	envPath := cfg.MonadHome + "/.env"
	nodeToml := cfg.MonadHome + "/monad-bft/config/node.toml"
	forkToml := cfg.MonadHome + "/monad-bft/config/forkpoint/forkpoint.toml"
	blsPath := cfg.MonadHome + "/pubkey-secp-bls"

	out.Chain = parsefiles.EnvValue(envPath, "CHAIN")
	out.Network = parsefiles.TomlField(nodeToml, "network_name")
	out.ChainID = parsefiles.TomlField(nodeToml, "chain_id")
	out.SelfAddress = parsefiles.TomlField(nodeToml, "self_address")
	out.RecordSeqNum = parsefiles.TomlField(nodeToml, "self_record_seq_num")
	out.NodeSignature = parsefiles.TomlField(nodeToml, "self_name_record_sig")
	out.SelfAuthPort = parsefiles.TomlField(nodeToml, "self_auth_port")

	if v := parsefiles.TomlField(forkToml, "epoch"); v != "" {
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			out.ForkEpoch = n
			out.ForkEpochOK = true
		}
	}
	if v := parsefiles.TomlField(forkToml, "round"); v != "" {
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			out.ForkRound = n
			out.ForkRoundOK = true
		}
	}

	if b, err := os.ReadFile(blsPath); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "BLS public key:") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					out.BLSPubkey = parts[len(parts)-1]
				}
				break
			}
		}
	}

	// Local Prometheus metrics (8889)
	client := &http.Client{Timeout: cfg.HTTPTimeout}
	body, err := fetchBody(ctx, client, cfg.MetricsURL)
	if err != nil {
		out.errf("metrics", "%v", err)
	} else {
		if v, ok, _ := promtext.ParseMetricValue(body, "monad_total_uptime_us"); ok {
			out.UptimeSeconds = v / 1e6
			out.UptimeOK = true
		}
		if v, ok, _ := promtext.ParseMetricValue(body, "monad_statesync_syncing"); ok {
			out.StatesyncSyncing = v
		}
		if v, ok, _ := promtext.ParseMetricValue(body, "monad_statesync_progress_estimate"); ok {
			out.StatesyncProgressEstimate = v
			out.StatesyncProgressOK = true
		}
		if v, ok, _ := promtext.ParseMetricValue(body, "monad_statesync_last_target"); ok {
			out.StatesyncLastTarget = v
			out.StatesyncProgressOK = true
		}
		out.StatesyncPercentage = statesyncPercent(out.StatesyncProgressEstimate, out.StatesyncLastTarget)
	}

	// JSON-RPC
	rc := &rpc.Client{BaseURL: cfg.RPCURL, HTTP: client}
	if v, err := rc.ClientVersion(ctx); err != nil {
		out.errf("rpc_clientVersion", "%v", err)
	} else {
		out.ClientVersion = v
	}
	if v, err := rc.NetVersion(ctx); err != nil {
		out.errf("rpc_net_version", "%v", err)
	} else {
		out.NetVersion = v
	}
	if n, err := rc.EthBlockNumber(ctx); err != nil {
		out.errf("rpc_eth_blockNumber", "%v", err)
	} else {
		out.RPCEthBlock = n
		out.RPCEthBlockOK = true
	}

	// External block
	extURL := fmt.Sprintf(cfg.ExternalRPCTemplate, out.Network)
	if out.Network != "" {
		ext := &rpc.Client{BaseURL: extURL, HTTP: client}
		if n, err := ext.EthBlockNumber(ctx); err != nil {
			out.errf("external_rpc", "%v", err)
		} else {
			out.ExternalBlock = n
			out.ExternalBlockOK = true
		}
	} else {
		out.errf("external_rpc", "network_name empty")
	}

	// MPT local block
	mptOut, err := executil.Run(ctx, cfg.HTTPTimeout, "monad-mpt", "--storage", cfg.MPTStorage)
	if err != nil {
		out.errf("monad_mpt", "%v", err)
	} else {
		out.LocalBlockRaw = mpt.HistoryLatestFromOutput(mptOut)
		if out.LocalBlockRaw != "" && out.LocalBlockRaw != "n/a" {
			if u, err := strconv.ParseUint(strings.TrimSpace(out.LocalBlockRaw), 10, 64); err == nil {
				out.LocalBlock = u
				out.LocalBlockOK = true
			}
		}
		out.MPT = mpt.ParseInfo(mptOut)
	}

	// Block diff & sync (same rules as monad-status.sh)
	out.BlockDiff, out.BlockDiffOK = blockDiff(out.LocalBlockOK, out.LocalBlock, out.ExternalBlockOK, out.ExternalBlock)
	out.InSync, out.InSyncOK = inSync(out.BlockDiffOK, out.BlockDiff)

	// ss RPC listen
	active, err := rpcPortListening(ctx, cfg.HTTPTimeout, cfg.RPCListenPort)
	if err != nil {
		out.errf("ss", "%v", err)
		out.RPCListenActive = 0
	} else if active {
		out.RPCListenActive = 1
	} else {
		out.RPCListenActive = 0
	}

	// Peers
	peerOut, err := executil.Run(ctx, cfg.HTTPTimeout, "monad-debug-node", "-c", cfg.ControlPanelSock, "get-peers")
	if err != nil {
		out.errf("monad_debug_node", "%v", err)
	} else {
		c := 0
		for _, line := range strings.Split(peerOut, "\n") {
			if strings.Contains(line, "address") {
				c++
			}
		}
		out.Peers = float64(c)
		out.PeersOK = true
	}

	// systemd services
	out.ServiceBft = boolFloat(executil.SystemctlActive(ctx, cfg.HTTPTimeout, "monad-bft"))
	out.ServiceExecution = boolFloat(executil.SystemctlActive(ctx, cfg.HTTPTimeout, "monad-execution"))
	out.ServiceRpc = boolFloat(executil.SystemctlActive(ctx, cfg.HTTPTimeout, "monad-rpc"))
	out.ServiceArchiver = boolFloat(executil.SystemctlActive(ctx, cfg.HTTPTimeout, "monad-archiver"))
	out.ServiceOtel = boolFloat(executil.SystemctlActive(ctx, cfg.HTTPTimeout, "otelcol"))

	// dpkg package version
	pkgOut, err := executil.Run(ctx, cfg.HTTPTimeout, "sh", "-c", "dpkg -l | grep ' monad '")
	if err != nil {
		out.errf("dpkg", "%v", err)
	} else {
		fields := strings.Fields(pkgOut)
		if len(fields) >= 3 {
			out.PackageVersion = fields[2]
		}
	}

	// triedb device (same as ls -l /dev/triedb | awk '{print $NF}')
	if lsOut, err := executil.Run(ctx, cfg.HTTPTimeout, "ls", "-l", "/dev/triedb"); err == nil {
		parts := strings.Fields(lsOut)
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			last = strings.TrimPrefix(last, "/dev/")
			out.TrieDevice = last
		}
	}
	if out.TrieDevice != "" {
		pk, err := executil.Run(ctx, cfg.HTTPTimeout, "lsblk", "-no", "PKNAME", "/dev/"+out.TrieDevice)
		dev := strings.TrimSpace(pk)
		if err == nil && dev != "" {
			modelOut, err2 := executil.Run(ctx, cfg.HTTPTimeout, "lsblk", "-no", "MODEL", "/dev/"+dev)
			if err2 == nil && strings.TrimSpace(modelOut) != "" {
				lines := strings.SplitN(strings.TrimSpace(modelOut), "\n", 2)
				out.TrieModel = strings.TrimSpace(lines[0])
			}
		}
	}

	// cruft timer
	cruftOut, err := executil.Run(ctx, cfg.HTTPTimeout, "systemctl", "list-timers", "--all", "monad-cruft.timer")
	if err != nil {
		out.errf("cruft_timer", "%v", err)
	} else {
		prev, next, act := parseCruftTimers(cruftOut)
		out.CruftPrevious = prev
		out.CruftNext = next
		if act {
			out.CruftActivated = 1
		} else {
			out.CruftActivated = 0
		}
	}

	out.SecpPubkey = cfg.SecpPublicKey
	if cfg.EnableKeystore && out.SecpPubkey == "" {
		out.errf("keystore", "empty secp pubkey")
	}

	if cfg.EnableStaking {
		gatherStaking(ctx, cfg, out, rc)
	}

	return out
}

func fetchBody(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}

func statesyncPercent(prog, target float64) float64 {
	if target == 0 {
		return 0
	}
	pct := (prog / target) * 100
	return math.Floor(pct*10000) / 10000
}

func blockDiff(localOK bool, local uint64, extOK bool, external uint64) (float64, bool) {
	if !localOK || !extOK || local == 0 {
		return 0, false
	}
	return float64(int64(external) - int64(local)), true
}

func inSync(diffOK bool, diff float64) (float64, bool) {
	if !diffOK {
		return 0, false
	}
	ad := diff
	if ad < 0 {
		ad = -ad
	}
	if ad < 10 {
		return 1, true
	}
	return 0, true
}

func boolFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func rpcPortListening(ctx context.Context, timeout time.Duration, port string) (bool, error) {
	sport := ":" + strings.TrimPrefix(port, ":")
	out, err := executil.Run(ctx, timeout, "ss", "-ltn", "sport", "=", sport)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "LISTEN"), nil
}

func parseCruftTimers(output string) (previous, next string, activated bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "monad-cruft.timer") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			next = strings.Join(fields[0:4], " ")
		}
		if len(fields) > 4 {
			previous = strings.Join(fields[4:], " ")
		}
		return previous, next, true
	}
	return "", "", false
}
