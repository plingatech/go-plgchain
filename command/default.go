package command

import (
	"github.com/plingatech/go-plgchain/server"
	"github.com/umbracle/ethgo"
)

const (
	DefaultGenesisFileName = "genesis.json"
	DefaultChainName       = "go-plgchain"
	DefaultChainID         = 424
	DefaultConsensus       = server.PlgBFTConsensus
	DefaultGenesisGasUsed  = 458752  // 0x70000
	DefaultGenesisGasLimit = 5242880 // 0x500000
)

var (
	DefaultStake          = ethgo.Ether(1e6)
	DefaultPremineBalance = ethgo.Ether(1e6)
)

const (
	JSONOutputFlag  = "json"
	GRPCAddressFlag = "grpc-address"
	JSONRPCFlag     = "jsonrpc"
)

// GRPCAddressFlagLEGACY Legacy flag that needs to be present to preserve backwards
// compatibility with running clients
const (
	GRPCAddressFlagLEGACY = "grpc"
)
