package staking

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestUnpackValidator(t *testing.T) {
	out := []interface{}{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		uint64(1),
		big.NewInt(1e18),
		big.NewInt(0),
		new(big.Int).Mul(big.NewInt(1e17), big.NewInt(1)), // 0.1 * 1e18 commission
		big.NewInt(0),
		big.NewInt(2e18),
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		[]byte{1, 2},
		[]byte{3},
	}
	v, err := unpackValidator(out)
	if err != nil {
		t.Fatal(err)
	}
	if v.Flags != 1 {
		t.Fatalf("flags %d", v.Flags)
	}
	if WeiToMon(v.StakeWei) != 1.0 {
		t.Fatalf("stake mon %v", WeiToMon(v.StakeWei))
	}
}
