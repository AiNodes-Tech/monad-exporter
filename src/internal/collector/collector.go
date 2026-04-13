package collector

import (
	"context"
	"math"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"monad-exporter/internal/config"
)

// NodeCollector implements prometheus.Collector for monad-status-derived metrics (prefix monad_exporter_).
type NodeCollector struct {
	cfg *config.Config
}

// NewNodeCollector creates a collector bound to config.
func NewNodeCollector(cfg *config.Config) *NodeCollector {
	return &NodeCollector{cfg: cfg}
}

// Describe implements prometheus.Collector.
func (c *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descs() {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *NodeCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	s := Gather(ctx, c.cfg)

	for src := range s.ScrapeErrors {
		ch <- prometheus.MustNewConstMetric(
			scrapeErrorDesc,
			prometheus.GaugeValue,
			1,
			src,
		)
	}

	ch <- prometheus.MustNewConstMetric(uptimeSecondsDesc, prometheus.GaugeValue, valOrNaN(s.UptimeOK, s.UptimeSeconds))
	ch <- prometheus.MustNewConstMetric(statesyncSyncingDesc, prometheus.GaugeValue, s.StatesyncSyncing)
	ch <- prometheus.MustNewConstMetric(statesyncProgressEstimateDesc, prometheus.GaugeValue, valOrNaN(s.StatesyncProgressOK, s.StatesyncProgressEstimate))
	ch <- prometheus.MustNewConstMetric(statesyncLastTargetDesc, prometheus.GaugeValue, valOrNaN(s.StatesyncProgressOK, s.StatesyncLastTarget))
	ch <- prometheus.MustNewConstMetric(statesyncPercentageDesc, prometheus.GaugeValue, s.StatesyncPercentage)

	ch <- prometheus.MustNewConstMetric(consensusLocalBlockDesc, prometheus.GaugeValue, valOrNaN(s.LocalBlockOK, float64(s.LocalBlock)))
	ch <- prometheus.MustNewConstMetric(consensusExternalBlockDesc, prometheus.GaugeValue, valOrNaN(s.ExternalBlockOK, float64(s.ExternalBlock)))
	ch <- prometheus.MustNewConstMetric(consensusBlockDifferenceDesc, prometheus.GaugeValue, valOrNaN(s.BlockDiffOK, s.BlockDiff))
	ch <- prometheus.MustNewConstMetric(consensusInSyncDesc, prometheus.GaugeValue, valOrNaN(s.InSyncOK, s.InSync))

	ch <- prometheus.MustNewConstMetric(peersDesc, prometheus.GaugeValue, valOrNaN(s.PeersOK, s.Peers))
	ch <- prometheus.MustNewConstMetric(forkEpochDesc, prometheus.GaugeValue, valOrNaN(s.ForkEpochOK, s.ForkEpoch))
	ch <- prometheus.MustNewConstMetric(forkRoundDesc, prometheus.GaugeValue, valOrNaN(s.ForkRoundOK, s.ForkRound))

	ch <- prometheus.MustNewConstMetric(rpcListenActiveDesc, prometheus.GaugeValue, s.RPCListenActive)
	ch <- prometheus.MustNewConstMetric(rpcEthBlockNumberDesc, prometheus.GaugeValue, valOrNaN(s.RPCEthBlockOK, float64(s.RPCEthBlock)))

	ch <- prometheus.MustNewConstMetric(serviceActiveDesc, prometheus.GaugeValue, s.ServiceBft, "monad-bft")
	ch <- prometheus.MustNewConstMetric(serviceActiveDesc, prometheus.GaugeValue, s.ServiceExecution, "monad-execution")
	ch <- prometheus.MustNewConstMetric(serviceActiveDesc, prometheus.GaugeValue, s.ServiceRpc, "monad-rpc")
	ch <- prometheus.MustNewConstMetric(serviceActiveDesc, prometheus.GaugeValue, s.ServiceArchiver, "monad-archiver")
	ch <- prometheus.MustNewConstMetric(serviceActiveDesc, prometheus.GaugeValue, s.ServiceOtel, "otelcol")

	ch <- prometheus.MustNewConstMetric(cruftActivatedDesc, prometheus.GaugeValue, s.CruftActivated)

	// Info metric (value 1) with main labels
	ch <- prometheus.MustNewConstMetric(
		nodeInfoDesc,
		prometheus.GaugeValue,
		1,
		truncate(s.Hostname, 128),
		truncate(s.Network, 64),
		truncate(s.Chain, 64),
		truncate(s.ChainID, 32),
		truncate(s.PackageVersion, 64),
		truncate(s.ClientVersion, 128),
		truncate(s.NetVersion, 32),
		truncate(s.SelfAddress, 128),
		truncate(s.TrieModel, 128),
		truncate(s.TrieDevice, 64),
	)

	ch <- prometheus.MustNewConstMetric(
		nodeExtraInfoDesc,
		prometheus.GaugeValue,
		1,
		truncate(s.RecordSeqNum, 32),
		truncate(s.SelfAuthPort, 16),
		truncate(s.NodeSignature, 256),
		truncate(s.SecpPubkey, 256),
		truncate(s.BLSPubkey, 256),
		truncate(s.LocalBlockRaw, 32),
		truncate(s.CruftNext, 128),
		truncate(s.CruftPrevious, 128),
	)

	if s.MPT != nil {
		ch <- prometheus.MustNewConstMetric(
			mptInfoDesc,
			prometheus.GaugeValue,
			1,
			truncate(s.MPT.HistoryCount, 32),
			truncate(s.MPT.HistoryEarliest, 32),
			truncate(s.MPT.HistoryLatest, 32),
			truncate(s.MPT.HistoryRetention, 32),
			truncate(s.MPT.LatestFinalized, 32),
			truncate(s.MPT.LatestVerified, 32),
			truncate(s.MPT.AutoExpireVersion, 32),
			truncate(s.MPT.Path, 128),
			truncate(s.MPT.UsedPercent, 16),
		)
	}

	en := c.cfg.EnableStaking
	execOK := en && s.StakingExecutionCountOK
	consOK := en && s.StakingConsensusCountOK
	inactiveOK := en && s.StakingExecutionCountOK && s.StakingConsensusCountOK
	valOK := en && s.StakingValidatorOK
	activeOK := en && s.StakingValidatorActiveConsensusOK
	delegOK := en && s.StakingAuthDelegatorOK
	balOK := en && s.StakingAuthWalletBalanceOK
	vid := strconv.FormatUint(c.cfg.ValidatorID, 10)
	ch <- prometheus.MustNewConstMetric(stakingValidatorsExecutionDesc, prometheus.GaugeValue, valOrNaN(execOK, s.StakingValidatorsExecution))
	ch <- prometheus.MustNewConstMetric(stakingValidatorsConsensusDesc, prometheus.GaugeValue, valOrNaN(consOK, s.StakingValidatorsConsensus))
	ch <- prometheus.MustNewConstMetric(stakingValidatorsInactiveExecDesc, prometheus.GaugeValue, valOrNaN(inactiveOK, s.StakingValidatorsInactiveExec))
	ch <- prometheus.MustNewConstMetric(stakingValidatorActiveDesc, prometheus.GaugeValue, valOrNaN(activeOK, s.StakingValidatorActiveConsensus), vid)
	ch <- prometheus.MustNewConstMetric(stakingValidatorInfoDesc, prometheus.GaugeValue, valOrNaN(valOK, 1), vid, truncate(s.StakingValidatorAuthAddress, 42))
	ch <- prometheus.MustNewConstMetric(stakingValidatorPoolStakeMonDesc, prometheus.GaugeValue, valOrNaN(valOK, s.StakingValidatorPoolStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingValidatorConsensusStakeMonDesc, prometheus.GaugeValue, valOrNaN(valOK, s.StakingValidatorConsensusStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingValidatorSnapshotStakeMonDesc, prometheus.GaugeValue, valOrNaN(valOK, s.StakingValidatorSnapshotStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingCommissionRatioDesc, prometheus.GaugeValue, valOrNaN(valOK, s.StakingCommissionRatio))
	ch <- prometheus.MustNewConstMetric(stakingValidatorUnclaimedRewardsMonDesc, prometheus.GaugeValue, valOrNaN(valOK, s.StakingValidatorUnclaimedRewardsMON))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorStakeMonDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorDeltaStakeMonDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorDeltaStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorNextDeltaStakeMonDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorNextDeltaStakeMON))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorDeltaEpochDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorDeltaEpoch))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorNextDeltaEpochDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorNextDeltaEpoch))
	ch <- prometheus.MustNewConstMetric(stakingAuthDelegatorUnclaimedRewardsMonDesc, prometheus.GaugeValue, valOrNaN(delegOK, s.StakingAuthDelegatorUnclaimedRewardsMON))
	ch <- prometheus.MustNewConstMetric(stakingAuthWalletBalanceMonDesc, prometheus.GaugeValue, valOrNaN(balOK, s.StakingAuthWalletBalanceMON))

	cgOK := c.cfg.EnableCoinGecko && s.CoinGeckoPriceUSDOK
	ch <- prometheus.MustNewConstMetric(monadPriceUSDDesc, prometheus.GaugeValue, valOrNaN(cgOK, s.CoinGeckoPriceUSD))
}

func valOrNaN(ok bool, v float64) float64 {
	if !ok {
		return math.NaN()
	}
	return v
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return ""
	}
	return s[:max-3] + "..."
}

func (c *NodeCollector) descs() []*prometheus.Desc {
	return []*prometheus.Desc{
		scrapeErrorDesc,
		uptimeSecondsDesc,
		statesyncSyncingDesc,
		statesyncProgressEstimateDesc,
		statesyncLastTargetDesc,
		statesyncPercentageDesc,
		consensusLocalBlockDesc,
		consensusExternalBlockDesc,
		consensusBlockDifferenceDesc,
		consensusInSyncDesc,
		peersDesc,
		forkEpochDesc,
		forkRoundDesc,
		rpcListenActiveDesc,
		rpcEthBlockNumberDesc,
		serviceActiveDesc,
		cruftActivatedDesc,
		nodeInfoDesc,
		nodeExtraInfoDesc,
		mptInfoDesc,
		stakingValidatorsExecutionDesc,
		stakingValidatorsConsensusDesc,
		stakingValidatorsInactiveExecDesc,
		stakingValidatorActiveDesc,
		stakingValidatorInfoDesc,
		stakingValidatorPoolStakeMonDesc,
		stakingValidatorConsensusStakeMonDesc,
		stakingValidatorSnapshotStakeMonDesc,
		stakingCommissionRatioDesc,
		stakingValidatorUnclaimedRewardsMonDesc,
		stakingAuthDelegatorStakeMonDesc,
		stakingAuthDelegatorDeltaStakeMonDesc,
		stakingAuthDelegatorNextDeltaStakeMonDesc,
		stakingAuthDelegatorDeltaEpochDesc,
		stakingAuthDelegatorNextDeltaEpochDesc,
		stakingAuthDelegatorUnclaimedRewardsMonDesc,
		stakingAuthWalletBalanceMonDesc,
		monadPriceUSDDesc,
	}
}

var (
	scrapeErrorDesc = prometheus.NewDesc(
		"monad_exporter_scrape_error",
		"1 if a scrape sub-source failed (see logs / operational checks).",
		[]string{"source"}, nil,
	)
	uptimeSecondsDesc = prometheus.NewDesc(
		"monad_exporter_uptime_seconds",
		"Node uptime in seconds from monad_total_uptime_us.",
		nil, nil,
	)
	statesyncSyncingDesc = prometheus.NewDesc(
		"monad_exporter_statesync_syncing",
		"monad_statesync_syncing from local metrics endpoint.",
		nil, nil,
	)
	statesyncProgressEstimateDesc = prometheus.NewDesc(
		"monad_exporter_statesync_progress_estimate",
		"monad_statesync_progress_estimate from local metrics endpoint.",
		nil, nil,
	)
	statesyncLastTargetDesc = prometheus.NewDesc(
		"monad_exporter_statesync_last_target",
		"monad_statesync_last_target from local metrics endpoint.",
		nil, nil,
	)
	statesyncPercentageDesc = prometheus.NewDesc(
		"monad_exporter_statesync_percentage",
		"Floor(progress/target*100, 4 decimals) when target non-zero.",
		nil, nil,
	)
	consensusLocalBlockDesc = prometheus.NewDesc(
		"monad_exporter_consensus_local_block",
		"Latest history block from monad-mpt (same as monad-status local block).",
		nil, nil,
	)
	consensusExternalBlockDesc = prometheus.NewDesc(
		"monad_exporter_consensus_external_block",
		"eth_blockNumber from external RPC (rpc-<network>.monadinfra.com).",
		nil, nil,
	)
	consensusBlockDifferenceDesc = prometheus.NewDesc(
		"monad_exporter_consensus_block_difference",
		"external_block - local_block when both valid and local > 0.",
		nil, nil,
	)
	consensusInSyncDesc = prometheus.NewDesc(
		"monad_exporter_consensus_in_sync",
		"1 if absolute block difference < 10, else 0.",
		nil, nil,
	)
	peersDesc = prometheus.NewDesc(
		"monad_exporter_peers",
		"Peer count from monad-debug-node get-peers (lines containing 'address').",
		nil, nil,
	)
	forkEpochDesc = prometheus.NewDesc(
		"monad_exporter_forkpoint_epoch",
		"epoch from forkpoint.toml.",
		nil, nil,
	)
	forkRoundDesc = prometheus.NewDesc(
		"monad_exporter_forkpoint_round",
		"round from forkpoint.toml.",
		nil, nil,
	)
	rpcListenActiveDesc = prometheus.NewDesc(
		"monad_exporter_rpc_listen_active",
		"1 if ss reports LISTEN on RPC port (default 8080).",
		nil, nil,
	)
	rpcEthBlockNumberDesc = prometheus.NewDesc(
		"monad_exporter_rpc_eth_block_number",
		"eth_blockNumber from local JSON-RPC.",
		nil, nil,
	)
	serviceActiveDesc = prometheus.NewDesc(
		"monad_exporter_service_active",
		"1 if systemctl is-active --quiet for the unit.",
		[]string{"unit"}, nil,
	)
	cruftActivatedDesc = prometheus.NewDesc(
		"monad_exporter_monad_cruft_timer_line_present",
		"1 if systemctl list-timers output contains monad-cruft.timer line.",
		nil, nil,
	)
	nodeInfoDesc = prometheus.NewDesc(
		"monad_exporter_node_info",
		"Static node metadata (value 1).",
		[]string{"hostname", "network", "chain", "chain_id", "package_version", "client_version", "net_version", "self_address", "triedb_model", "triedb_device"},
		nil,
	)
	nodeExtraInfoDesc = prometheus.NewDesc(
		"monad_exporter_node_extra_info",
		"Additional metadata (value 1).",
		[]string{"record_seq_num", "self_auth_port", "node_signature", "secp_pubkey", "bls_pubkey", "local_block_raw", "cruft_next", "cruft_previous"},
		nil,
	)
	mptInfoDesc = prometheus.NewDesc(
		"monad_exporter_mpt_info",
		"MPT / triedb summary from monad-mpt (value 1).",
		[]string{"history_count", "history_earliest", "history_latest", "history_retention", "latest_finalized", "latest_verified", "auto_expire_version", "path", "used_percent"},
		nil,
	)
	stakingValidatorsExecutionDesc = prometheus.NewDesc(
		"monad_exporter_staking_validators_execution_total",
		"Count of validator ids from getExecutionValidatorSet (paginated).",
		nil, nil,
	)
	stakingValidatorsConsensusDesc = prometheus.NewDesc(
		"monad_exporter_staking_validators_consensus_total",
		"Count of validator ids from getConsensusValidatorSet (paginated).",
		nil, nil,
	)
	stakingValidatorsInactiveExecDesc = prometheus.NewDesc(
		"monad_exporter_staking_validators_inactive_execution_total",
		"Validators in execution set but not in current consensus set (set difference).",
		nil, nil,
	)
	stakingValidatorActiveDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_active",
		"1 if VALIDATOR_ID is in getConsensusValidatorSet, else 0.",
		[]string{"validator_id"}, nil,
	)
	stakingValidatorInfoDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_info",
		"Validator metadata from getValidator (value 1).",
		[]string{"validator_id", "address"}, nil,
	)
	stakingValidatorPoolStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_pool_stake_mon",
		"Validator execution pool stake from getValidator.stake, in MON.",
		nil, nil,
	)
	stakingValidatorConsensusStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_consensus_stake_mon",
		"Validator consensus stake from getValidator.consensusStake, in MON.",
		nil, nil,
	)
	stakingValidatorSnapshotStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_snapshot_stake_mon",
		"Validator snapshot stake from getValidator.snapshotStake, in MON.",
		nil, nil,
	)
	stakingCommissionRatioDesc = prometheus.NewDesc(
		"monad_exporter_staking_commission_ratio",
		"Validator commission rate (commission/1e18 from getValidator), 0..1.",
		nil, nil,
	)
	stakingValidatorUnclaimedRewardsMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_validator_unclaimed_rewards_mon",
		"Validator-level unclaimedRewards from getValidator (not delegator position), in MON.",
		nil, nil,
	)
	stakingAuthDelegatorStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_stake_mon",
		"Auth wallet self-delegation stake from getDelegator(validatorId, authAddress).stake, in MON.",
		nil, nil,
	)
	stakingAuthDelegatorDeltaStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_delta_stake_mon",
		"Pending stake delta for auth as delegator (getDelegator.deltaStake), in MON.",
		nil, nil,
	)
	stakingAuthDelegatorNextDeltaStakeMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_next_delta_stake_mon",
		"Next-epoch pending stake delta for auth as delegator (getDelegator.nextDeltaStake), in MON.",
		nil, nil,
	)
	stakingAuthDelegatorDeltaEpochDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_delta_epoch",
		"Epoch associated with deltaStake for auth as delegator (getDelegator.deltaEpoch).",
		nil, nil,
	)
	stakingAuthDelegatorNextDeltaEpochDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_next_delta_epoch",
		"Epoch associated with nextDeltaStake for auth as delegator (getDelegator.nextDeltaEpoch).",
		nil, nil,
	)
	stakingAuthDelegatorUnclaimedRewardsMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_delegator_unclaimed_rewards_mon",
		"Unclaimed rewards for auth as delegator to self (getDelegator.unclaimedRewards), in MON.",
		nil, nil,
	)
	stakingAuthWalletBalanceMonDesc = prometheus.NewDesc(
		"monad_exporter_staking_auth_wallet_balance_mon",
		"Native MON balance on validator auth address (eth_getBalance latest), in MON.",
		nil, nil,
	)
	monadPriceUSDDesc = prometheus.NewDesc(
		"monad_exporter_monad_price_usd",
		"Spot price of configured CoinGecko coin in USD (/simple/price, cached).",
		nil, nil,
	)
)

// Ensure interface.
var _ prometheus.Collector = (*NodeCollector)(nil)
