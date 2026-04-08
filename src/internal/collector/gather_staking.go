package collector

import (
	"context"

	"monad-exporter/internal/config"
	"monad-exporter/internal/rpc"
	"monad-exporter/internal/staking"
)

func gatherStaking(ctx context.Context, cfg *config.Config, out *Snapshot, rc *rpc.Client) {
	if !cfg.EnableStaking {
		return
	}

	sc, err := staking.NewClient(rc, cfg.StakingAddress)
	if err != nil {
		out.errf("staking_client", "%v", err)
		return
	}

	execIDs, err := sc.CollectAllValidatorIDs(ctx, "getExecutionValidatorSet")
	if err != nil {
		out.errf("staking_execution_valset", "%v", err)
	} else {
		out.StakingValidatorsExecution = float64(len(execIDs))
		out.StakingExecutionCountOK = true
	}

	consIDs, err := sc.CollectAllValidatorIDs(ctx, "getConsensusValidatorSet")
	if err != nil {
		out.errf("staking_consensus_valset", "%v", err)
	} else {
		out.StakingValidatorsConsensus = float64(len(consIDs))
		out.StakingConsensusCountOK = true
	}

	var consSet map[uint64]struct{}
	if out.StakingConsensusCountOK {
		consSet = make(map[uint64]struct{}, len(consIDs))
		for _, id := range consIDs {
			consSet[id] = struct{}{}
		}
		if cfg.ValidatorID != 0 {
			out.StakingValidatorActiveConsensusOK = true
			if _, ok := consSet[cfg.ValidatorID]; ok {
				out.StakingValidatorActiveConsensus = 1
			} else {
				out.StakingValidatorActiveConsensus = 0
			}
		}
	}

	if out.StakingExecutionCountOK && out.StakingConsensusCountOK {
		var inactive int
		for _, id := range execIDs {
			if _, ok := consSet[id]; !ok {
				inactive++
			}
		}
		out.StakingValidatorsInactiveExec = float64(inactive)
	}

	if cfg.ValidatorID == 0 {
		out.errf("staking_validator_id", "set VALIDATOR_ID for per-validator staking metrics")
		return
	}

	val, err := sc.GetValidator(ctx, cfg.ValidatorID)
	if err != nil {
		out.errf("staking_getValidator", "%v", err)
		out.StakingValidatorOK = false
		return
	}
	out.StakingValidatorPoolStakeMON = staking.WeiToMon(val.StakeWei)
	out.StakingCommissionRatio = staking.CommissionRatio(val.CommissionWei)
	out.StakingValidatorUnclaimedRewardsMON = staking.WeiToMon(val.UnclaimedRewardsWei)
	out.StakingValidatorAuthAddress = val.AuthAddress.Hex()
	out.StakingValidatorOK = true
}
