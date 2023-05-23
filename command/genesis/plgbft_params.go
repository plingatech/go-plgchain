package genesis

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/plingatech/go-plgchain/consensus/plgbft/contractsapi"
	"github.com/plingatech/go-plgchain/consensus/plgbft/contractsapi/artifact"

	"github.com/plingatech/go-plgchain/chain"
	"github.com/plingatech/go-plgchain/command"
	"github.com/plingatech/go-plgchain/command/helper"

	"github.com/plingatech/go-plgchain/consensus/plgbft"
	"github.com/plingatech/go-plgchain/consensus/plgbft/bitmap"
	"github.com/plingatech/go-plgchain/contracts"
	"github.com/plingatech/go-plgchain/server"
	"github.com/plingatech/go-plgchain/types"
)

const (
	manifestPathFlag       = "manifest"
	validatorSetSizeFlag   = "validator-set-size"
	sprintSizeFlag         = "sprint-size"
	blockTimeFlag          = "block-time"
	bridgeFlag             = "bridge-json-rpc"
	trackerStartBlocksFlag = "tracker-start-blocks"
	trieRootFlag           = "trieroot"

	defaultManifestPath     = "./manifest.json"
	defaultEpochSize        = uint64(10)
	defaultSprintSize       = uint64(5)
	defaultValidatorSetSize = 100
	defaultBlockTime        = 2 * time.Second
	defaultBridge           = false
	defaultEpochReward      = 1

	contractDeployerAllowListAdminFlag   = "contract-deployer-allow-list-admin"
	contractDeployerAllowListEnabledFlag = "contract-deployer-allow-list-enabled"
	transactionsAllowListAdminFlag       = "transactions-allow-list-admin"
	transactionsAllowListEnabledFlag     = "transactions-allow-list-enabled"

	bootnodePortStart = 30342
)

var (
	errNoGenesisValidators = errors.New("genesis validators aren't provided")
)

// generatePlgBftChainConfig creates and persists plgbft chain configuration to the provided file path
func (p *genesisParams) generatePlgBftChainConfig(o command.OutputFormatter) error {
	// load manifest file
	manifest, err := plgbft.LoadManifest(p.manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest file from provided path '%s': %w", p.manifestPath, err)
	}

	if len(manifest.GenesisValidators) == 0 {
		return errNoGenesisValidators
	}

	eventTrackerStartBlock, err := parseTrackerStartBlocks(params.eventTrackerStartBlocks)
	if err != nil {
		return err
	}

	var bridge *plgbft.BridgeConfig

	// populate bridge configuration
	if p.bridgeJSONRPCAddr != "" && manifest.RootchainConfig != nil {
		bridge = manifest.RootchainConfig.ToBridgeConfig()
		bridge.JSONRPCEndpoint = p.bridgeJSONRPCAddr
		bridge.EventTrackerStartBlocks = eventTrackerStartBlock
	}

	if _, err := o.Write([]byte("[GENESIS VALIDATORS]\n")); err != nil {
		return err
	}

	for _, v := range manifest.GenesisValidators {
		if _, err := o.Write([]byte(fmt.Sprintf("%v\n", v))); err != nil {
			return err
		}
	}

	plgBftConfig := &plgbft.PlgBFTConfig{
		InitialValidatorSet: manifest.GenesisValidators,
		BlockTime:           p.blockTime,
		EpochSize:           p.epochSize,
		SprintSize:          p.sprintSize,
		EpochReward:         p.epochReward,
		// use 1st account as governance address
		Governance:         manifest.GenesisValidators[0].Address,
		Bridge:             bridge,
		InitialTrieRoot:    types.StringToHash(p.initialStateRoot),
		MintableERC20Token: p.mintableNativeToken,
	}

	chainConfig := &chain.Chain{
		Name: p.name,
		Params: &chain.Params{
			ChainID: manifest.ChainID,
			Forks:   chain.AllForksEnabled,
			Engine: map[string]interface{}{
				string(server.PlgBFTConsensus): plgBftConfig,
			},
		},
		Bootnodes: p.bootnodes,
	}

	genesisValidators := make(map[types.Address]struct{}, len(manifest.GenesisValidators))
	totalStake := big.NewInt(0)

	for _, validator := range manifest.GenesisValidators {
		// populate premine info for validator accounts
		genesisValidators[validator.Address] = struct{}{}

		// increment total stake
		totalStake.Add(totalStake, validator.Stake)
	}

	// deploy genesis contracts
	allocs, err := p.deployContracts(totalStake)
	if err != nil {
		return err
	}

	premineInfos := make([]*PremineInfo, len(p.premine))
	premineValidatorsAddrs := []string{}
	// premine non-validator
	for i, premine := range p.premine {
		premineInfo, err := ParsePremineInfo(premine)
		if err != nil {
			return err
		}

		// collect validators addresses which got premined, as it is an error
		// genesis validators balances must be defined in manifest file and should not be changed in the genesis
		if _, ok := genesisValidators[premineInfo.Address]; ok {
			premineValidatorsAddrs = append(premineValidatorsAddrs, premineInfo.Address.String())
		} else {
			premineInfos[i] = premineInfo
		}
	}

	// if there are any premined validators in the genesis command, consider it as an error
	if len(premineValidatorsAddrs) > 0 {
		return fmt.Errorf("it is not allowed to override genesis validators balance outside from the manifest definition. "+
			"Validators which got premined: (%s)", strings.Join(premineValidatorsAddrs, ", "))
	}

	// populate genesis validators balances
	for _, validator := range manifest.GenesisValidators {
		allocs[validator.Address] = &chain.GenesisAccount{
			Balance: validator.Balance,
		}
	}

	// premine non-validator accounts
	for _, premine := range premineInfos {
		allocs[premine.Address] = &chain.GenesisAccount{
			Balance: premine.Amount,
		}
	}

	validatorMetadata := make([]*plgbft.ValidatorMetadata, len(manifest.GenesisValidators))

	for i, validator := range manifest.GenesisValidators {
		// update balance of genesis validator, because it could be changed via premine flag
		balance, err := chain.GetGenesisAccountBalance(validator.Address, allocs)
		if err != nil {
			return err
		}

		validator.Balance = balance

		// create validator metadata instance
		metadata, err := validator.ToValidatorMetadata()
		if err != nil {
			return err
		}

		validatorMetadata[i] = metadata

		// set genesis validators as boot nodes if boot nodes not provided via CLI
		if len(p.bootnodes) == 0 {
			chainConfig.Bootnodes = append(chainConfig.Bootnodes, validator.MultiAddr)
		}
	}

	genesisExtraData, err := generateExtraDataPlgBft(validatorMetadata)
	if err != nil {
		return err
	}

	// populate genesis parameters
	chainConfig.Genesis = &chain.Genesis{
		GasLimit:   p.blockGasLimit,
		Difficulty: 0,
		Alloc:      allocs,
		ExtraData:  genesisExtraData,
		GasUsed:    command.DefaultGenesisGasUsed,
		Mixhash:    plgbft.PlgBFTMixDigest,
	}

	if len(p.contractDeployerAllowListAdmin) != 0 {
		// only enable allow list if there is at least one address as **admin**, otherwise
		// the allow list could never be updated
		chainConfig.Params.ContractDeployerAllowList = &chain.AllowListConfig{
			AdminAddresses:   stringSliceToAddressSlice(p.contractDeployerAllowListAdmin),
			EnabledAddresses: stringSliceToAddressSlice(p.contractDeployerAllowListEnabled),
		}
	}

	if len(p.transactionsAllowListAdmin) != 0 {
		// only enable allow list if there is at least one address as **admin**, otherwise
		// the allow list could never be updated
		chainConfig.Params.TransactionsAllowList = &chain.AllowListConfig{
			AdminAddresses:   stringSliceToAddressSlice(p.transactionsAllowListAdmin),
			EnabledAddresses: stringSliceToAddressSlice(p.transactionsAllowListEnabled),
		}
	}

	return helper.WriteGenesisConfigToDisk(chainConfig, params.genesisPath)
}

func (p *genesisParams) deployContracts(totalStake *big.Int) (map[types.Address]*chain.GenesisAccount, error) {
	type contractInfo struct {
		artifact *artifact.Artifact
		address  types.Address
	}

	genesisContracts := []*contractInfo{
		{
			// ChildValidatorSet contract
			artifact: contractsapi.ChildValidatorSet,
			address:  contracts.ValidatorSetContract,
		},
		{
			// State receiver contract
			artifact: contractsapi.StateReceiver,
			address:  contracts.StateReceiverContract,
		},
		{
			// ChildERC20 token contract
			artifact: contractsapi.ChildERC20,
			address:  contracts.ChildERC20Contract,
		},
		{
			// ChildERC20Predicate contract
			artifact: contractsapi.ChildERC20Predicate,
			address:  contracts.ChildERC20PredicateContract,
		},
		{
			// ChildERC721 token contract
			artifact: contractsapi.ChildERC721,
			address:  contracts.ChildERC721Contract,
		},
		{
			// ChildERC721Predicate token contract
			artifact: contractsapi.ChildERC721Predicate,
			address:  contracts.ChildERC721PredicateContract,
		},
		{
			// ChildERC1155 contract
			artifact: contractsapi.ChildERC1155,
			address:  contracts.ChildERC1155Contract,
		},
		{
			// ChildERC1155Predicate token contract
			artifact: contractsapi.ChildERC1155Predicate,
			address:  contracts.ChildERC1155PredicateContract,
		},
		{
			// BLS contract
			artifact: contractsapi.BLS,
			address:  contracts.BLSContract,
		},
		{
			// Merkle contract
			artifact: contractsapi.Merkle,
			address:  contracts.MerkleContract,
		},
		{
			// L2StateSender contract
			artifact: contractsapi.L2StateSender,
			address:  contracts.L2StateSenderContract,
		},
	}

	if !params.mintableNativeToken {
		genesisContracts = append(genesisContracts,
			&contractInfo{artifact: contractsapi.NativeERC20, address: contracts.NativeERC20TokenContract})
	} else {
		genesisContracts = append(genesisContracts,
			&contractInfo{artifact: contractsapi.NativeERC20Mintable, address: contracts.NativeERC20TokenContract})
	}

	allocations := make(map[types.Address]*chain.GenesisAccount, len(genesisContracts))

	for _, contract := range genesisContracts {
		allocations[contract.address] = &chain.GenesisAccount{
			Balance: big.NewInt(0),
			Code:    contract.artifact.DeployedBytecode,
		}
	}

	// ChildValidatorSet must have funds pre-allocated, because of withdrawal workflow
	allocations[contracts.ValidatorSetContract].Balance = totalStake

	return allocations, nil
}

// generateExtraDataPlgBft populates Extra with specific fields required for plgbft consensus protocol
func generateExtraDataPlgBft(validators []*plgbft.ValidatorMetadata) ([]byte, error) {
	delta := &plgbft.ValidatorSetDelta{
		Added:   validators,
		Removed: bitmap.Bitmap{},
	}

	extra := plgbft.Extra{Validators: delta, Checkpoint: &plgbft.CheckpointData{}}

	return append(make([]byte, plgbft.ExtraVanity), extra.MarshalRLPTo(nil)...), nil
}

func stringSliceToAddressSlice(addrs []string) []types.Address {
	res := make([]types.Address, len(addrs))
	for indx, addr := range addrs {
		res[indx] = types.StringToAddress(addr)
	}

	return res
}
