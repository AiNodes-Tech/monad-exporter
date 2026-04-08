package staking

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"monad-exporter/internal/rpc"
)

// DefaultStakingAddress is the Monad staking precompile.
const DefaultStakingAddress = "0x0000000000000000000000000000000000001000"

// Client wraps eth_call to the staking precompile.
type Client struct {
	RPC     *rpc.Client
	Address common.Address
	ABI     abi.ABI
}

func NewClient(rpcClient *rpc.Client, stakingAddr string) (*Client, error) {
	if stakingAddr == "" {
		stakingAddr = DefaultStakingAddress
	}
	if !strings.HasPrefix(stakingAddr, "0x") {
		stakingAddr = "0x" + stakingAddr
	}
	parsed, err := abi.JSON(strings.NewReader(stakingABI))
	if err != nil {
		return nil, err
	}
	return &Client{
		RPC:     rpcClient,
		Address: common.HexToAddress(stakingAddr),
		ABI:     parsed,
	}, nil
}

func (c *Client) pack(method string, args ...interface{}) ([]byte, error) {
	return c.ABI.Pack(method, args...)
}

func (c *Client) call(ctx context.Context, method string, args ...interface{}) ([]byte, error) {
	data, err := c.pack(method, args...)
	if err != nil {
		return nil, err
	}
	return c.RPC.EthCall(ctx, c.Address.Hex(), data)
}

// ValidatorView is decoded getValidator output (numeric fields as big.Int wei where applicable).
type ValidatorView struct {
	AuthAddress         common.Address
	Flags               uint64
	StakeWei            *big.Int
	AccRewardPerToken   *big.Int
	CommissionWei       *big.Int
	UnclaimedRewardsWei *big.Int
	ConsensusStakeWei   *big.Int
	ConsensusCommission *big.Int
	SnapshotStakeWei    *big.Int
	SnapshotCommission  *big.Int
	SecpPubkey          []byte
	BlsPubkey           []byte
}

func (c *Client) GetValidator(ctx context.Context, validatorID uint64) (*ValidatorView, error) {
	raw, err := c.call(ctx, "getValidator", validatorID)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty getValidator response")
	}
	out, err := c.ABI.Unpack("getValidator", raw)
	if err != nil {
		return nil, err
	}
	return unpackValidator(out)
}

func unpackValidator(out []interface{}) (*ValidatorView, error) {
	if len(out) < 12 {
		return nil, fmt.Errorf("getValidator: expected 12 values, got %d", len(out))
	}
	v := &ValidatorView{
		AuthAddress:         out[0].(common.Address),
		Flags:               toU64(out[1]),
		StakeWei:            asBigInt(out[2]),
		AccRewardPerToken:   asBigInt(out[3]),
		CommissionWei:       asBigInt(out[4]),
		UnclaimedRewardsWei: asBigInt(out[5]),
		ConsensusStakeWei:   asBigInt(out[6]),
		ConsensusCommission: asBigInt(out[7]),
		SnapshotStakeWei:    asBigInt(out[8]),
		SnapshotCommission:  asBigInt(out[9]),
		SecpPubkey:          out[10].([]byte),
		BlsPubkey:           out[11].([]byte),
	}
	return v, nil
}

// DelegatorView is decoded getDelegator output.
type DelegatorView struct {
	StakeWei           *big.Int
	AccRewardPerToken  *big.Int
	UnclaimedRewardsWei *big.Int
	DeltaStakeWei      *big.Int
	NextDeltaStakeWei  *big.Int
	DeltaEpoch         uint64
	NextDeltaEpoch     uint64
}

func (c *Client) GetDelegator(ctx context.Context, validatorID uint64, delegator common.Address) (*DelegatorView, error) {
	raw, err := c.call(ctx, "getDelegator", validatorID, delegator)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty getDelegator response")
	}
	out, err := c.ABI.Unpack("getDelegator", raw)
	if err != nil {
		return nil, err
	}
	if len(out) < 7 {
		return nil, fmt.Errorf("getDelegator: expected 7 values, got %d", len(out))
	}
	return &DelegatorView{
		StakeWei:            asBigInt(out[0]),
		AccRewardPerToken:   asBigInt(out[1]),
		UnclaimedRewardsWei: asBigInt(out[2]),
		DeltaStakeWei:       asBigInt(out[3]),
		NextDeltaStakeWei:   asBigInt(out[4]),
		DeltaEpoch:          toU64(out[5]),
		NextDeltaEpoch:      toU64(out[6]),
	}, nil
}

// WithdrawalView is getWithdrawalRequest output.
type WithdrawalView struct {
	WithdrawalAmountWei *big.Int
	AccRewardPerToken   *big.Int
	WithdrawEpoch       uint64
}

func (c *Client) GetWithdrawalRequest(ctx context.Context, validatorID uint64, delegator common.Address, withdrawID uint8) (*WithdrawalView, error) {
	raw, err := c.call(ctx, "getWithdrawalRequest", validatorID, delegator, withdrawID)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return &WithdrawalView{WithdrawalAmountWei: big.NewInt(0), AccRewardPerToken: big.NewInt(0)}, nil
	}
	out, err := c.ABI.Unpack("getWithdrawalRequest", raw)
	if err != nil {
		return nil, err
	}
	if len(out) < 3 {
		return nil, fmt.Errorf("getWithdrawalRequest: expected 3 values, got %d", len(out))
	}
	return &WithdrawalView{
		WithdrawalAmountWei: asBigInt(out[0]),
		AccRewardPerToken:   asBigInt(out[1]),
		WithdrawEpoch:       toU64(out[2]),
	}, nil
}

// PaginatedValSet is one page from get*ValidatorSet.
type PaginatedValSet struct {
	IsDone    bool
	NextIndex uint32
	ValIDs    []uint64
}

func (c *Client) GetExecutionValidatorSet(ctx context.Context, startIndex uint32) (*PaginatedValSet, error) {
	return c.getValidatorSetPage(ctx, "getExecutionValidatorSet", startIndex)
}

func (c *Client) GetConsensusValidatorSet(ctx context.Context, startIndex uint32) (*PaginatedValSet, error) {
	return c.getValidatorSetPage(ctx, "getConsensusValidatorSet", startIndex)
}

func (c *Client) getValidatorSetPage(ctx context.Context, method string, startIndex uint32) (*PaginatedValSet, error) {
	raw, err := c.call(ctx, method, startIndex)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty %s response", method)
	}
	out, err := c.ABI.Unpack(method, raw)
	if err != nil {
		return nil, err
	}
	if len(out) < 3 {
		return nil, fmt.Errorf("%s: expected 3 values", method)
	}
	isDone := out[0].(bool)
	nextIndex := uint32(0)
	switch x := out[1].(type) {
	case uint32:
		nextIndex = x
	case uint64:
		nextIndex = uint32(x)
	case *big.Int:
		nextIndex = uint32(x.Uint64())
	default:
		return nil, fmt.Errorf("%s: unexpected nextIndex type", method)
	}
	var valIDs []uint64
	switch s := out[2].(type) {
	case []uint64:
		valIDs = s
	case []*big.Int:
		for _, b := range s {
			valIDs = append(valIDs, b.Uint64())
		}
	default:
		return nil, fmt.Errorf("%s: unexpected valIds type", method)
	}
	return &PaginatedValSet{IsDone: isDone, NextIndex: nextIndex, ValIDs: valIDs}, nil
}

// CollectAllValidatorIDs paginates until isDone (next start = nextIndex+1 per Monad docs).
func (c *Client) CollectAllValidatorIDs(ctx context.Context, method string) ([]uint64, error) {
	var all []uint64
	start := uint32(0)
	for iter := 0; iter < 10000; iter++ {
		var page *PaginatedValSet
		var err error
		if method == "getExecutionValidatorSet" {
			page, err = c.GetExecutionValidatorSet(ctx, start)
		} else {
			page, err = c.GetConsensusValidatorSet(ctx, start)
		}
		if err != nil {
			return nil, err
		}
		all = append(all, page.ValIDs...)
		if page.IsDone {
			break
		}
		start = page.NextIndex + 1
	}
	return all, nil
}

func asBigInt(v interface{}) *big.Int {
	if v == nil {
		return big.NewInt(0)
	}
	switch x := v.(type) {
	case *big.Int:
		if x == nil {
			return big.NewInt(0)
		}
		return new(big.Int).Set(x)
	default:
		return big.NewInt(0)
	}
}

func toU64(v interface{}) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case uint32:
		return uint64(x)
	case *big.Int:
		if x == nil {
			return 0
		}
		return x.Uint64()
	default:
		return 0
	}
}

// WeiToMon converts wei to MON (float) for Prometheus gauges (may lose precision for huge values).
func WeiToMon(w *big.Int) float64 {
	if w == nil || w.Sign() == 0 {
		return 0
	}
	f := new(big.Float).SetInt(w)
	f.Quo(f, big.NewFloat(1e18))
	v, _ := f.Float64()
	return v
}

// CommissionRatio returns commission / 1e18 as 0..1 (or >1 if misconfigured).
func CommissionRatio(commissionWei *big.Int) float64 {
	if commissionWei == nil || commissionWei.Sign() == 0 {
		return 0
	}
	f := new(big.Float).SetInt(commissionWei)
	f.Quo(f, big.NewFloat(1e18))
	v, _ := f.Float64()
	return v
}

// SumPendingWithdrawals sums withdrawalAmount for withdrawId 0..maxID-1.
func (c *Client) SumPendingWithdrawals(ctx context.Context, validatorID uint64, delegator common.Address, maxID int) (*big.Int, error) {
	sum := big.NewInt(0)
	for i := 0; i < maxID; i++ {
		wv, err := c.GetWithdrawalRequest(ctx, validatorID, delegator, uint8(i))
		if err != nil {
			return nil, err
		}
		if wv.WithdrawalAmountWei != nil && wv.WithdrawalAmountWei.Sign() > 0 {
			sum.Add(sum, wv.WithdrawalAmountWei)
		}
	}
	return sum, nil
}
