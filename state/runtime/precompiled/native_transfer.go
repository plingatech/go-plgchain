package precompiled

import (
	"math/big"

	"github.com/plingatech/go-plgchain/chain"
	"github.com/plingatech/go-plgchain/contracts"
	"github.com/plingatech/go-plgchain/state/runtime"
	"github.com/plingatech/go-plgchain/types"
)

type nativeTransfer struct{}

func (c *nativeTransfer) gas(input []byte, _ *chain.ForksInTime) uint64 {
	return 21000
}

func (c *nativeTransfer) run(input []byte, caller types.Address, host runtime.Host) ([]byte, error) {
	if len(input) < 96 {
		return abiBoolFalse, runtime.ErrInvalidInputData
	}

	// check if caller is native token contract
	if caller != contracts.NativeERC20TokenContract {
		return abiBoolFalse, runtime.ErrUnauthorizedCaller
	}

	from := types.BytesToAddress(input[0:32])
	to := types.BytesToAddress(input[32:64])
	amount := new(big.Int).SetBytes(input[64:96])

	// state changes
	if err := host.Transfer(from, to, amount); err != nil {
		return abiBoolFalse, err
	}

	return abiBoolTrue, nil
}
