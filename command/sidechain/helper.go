package sidechain

import (
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/plingatech/go-plgchain/command/plgbftsecrets"
	"github.com/plingatech/go-plgchain/consensus/plgbft"
	"github.com/plingatech/go-plgchain/consensus/plgbft/contractsapi"
	"github.com/plingatech/go-plgchain/consensus/plgbft/wallet"
	"github.com/plingatech/go-plgchain/contracts"
	"github.com/plingatech/go-plgchain/helper/hex"
	"github.com/plingatech/go-plgchain/txrelayer"
	"github.com/plingatech/go-plgchain/types"
	"github.com/umbracle/ethgo"
)

const (
	SelfFlag   = "self"
	EtherFlag  = "ether"
	AmountFlag = "amount"

	DefaultGasPrice = 1879048192 // 0x70000000
)

func CheckIfDirectoryExist(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("provided directory '%s' doesn't exist", dir)
	}

	return nil
}

func ValidateSecretFlags(dataDir, config string) error {
	if config == "" {
		if dataDir == "" {
			return plgbftsecrets.ErrInvalidParams
		} else {
			return CheckIfDirectoryExist(dataDir)
		}
	}

	return nil
}

// GetAccount resolves secrets manager and returns an account object
func GetAccount(accountDir, accountConfig string) (*wallet.Account, error) {
	// resolve secrets manager instance and allow usage of insecure local secrets manager
	secretsManager, err := plgbftsecrets.GetSecretsManager(accountDir, accountConfig, true)
	if err != nil {
		return nil, err
	}

	return wallet.NewAccountFromSecret(secretsManager)
}

// GetAccountFromDir returns an account object from local secrets manager
func GetAccountFromDir(accountDir string) (*wallet.Account, error) {
	return GetAccount(accountDir, "")
}

// GetValidatorInfo queries ChildValidatorSet smart contract and retrieves validator info for given address
func GetValidatorInfo(validatorAddr ethgo.Address, txRelayer txrelayer.TxRelayer) (*plgbft.ValidatorInfo, error) {
	getValidatorMethod := contractsapi.ChildValidatorSet.Abi.GetMethod("getValidator")

	encode, err := getValidatorMethod.Encode([]interface{}{validatorAddr})
	if err != nil {
		return nil, err
	}

	response, err := txRelayer.Call(ethgo.Address(contracts.SystemCaller),
		ethgo.Address(contracts.ValidatorSetContract), encode)
	if err != nil {
		return nil, err
	}

	byteResponse, err := hex.DecodeHex(response)
	if err != nil {
		return nil, fmt.Errorf("unable to decode hex response, %w", err)
	}

	decoded, err := getValidatorMethod.Outputs.Decode(byteResponse)
	if err != nil {
		return nil, err
	}

	decodedOutputsMap, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert decoded outputs to map")
	}

	return &plgbft.ValidatorInfo{
		Address:             validatorAddr.Address(),
		Stake:               decodedOutputsMap["stake"].(*big.Int),               //nolint:forcetypeassert
		TotalStake:          decodedOutputsMap["totalStake"].(*big.Int),          //nolint:forcetypeassert
		Commission:          decodedOutputsMap["commission"].(*big.Int),          //nolint:forcetypeassert
		WithdrawableRewards: decodedOutputsMap["withdrawableRewards"].(*big.Int), //nolint:forcetypeassert
		Active:              decodedOutputsMap["active"].(bool),                  //nolint:forcetypeassert
	}, nil
}

// GetDelegatorReward queries delegator reward for given validator and delegator addresses
func GetDelegatorReward(validatorAddr ethgo.Address, delegatorAddr ethgo.Address,
	txRelayer txrelayer.TxRelayer) (*big.Int, error) {
	input, err := contractsapi.ChildValidatorSet.Abi.Methods["getDelegatorReward"].Encode(
		[]interface{}{validatorAddr, delegatorAddr})
	if err != nil {
		return nil, fmt.Errorf("failed to encode input parameters for getDelegatorReward fn: %w", err)
	}

	response, err := txRelayer.Call(ethgo.Address(contracts.SystemCaller),
		ethgo.Address(contracts.ValidatorSetContract), input)
	if err != nil {
		return nil, err
	}

	delegatorReward, err := types.ParseUint256orHex(&response)
	if err != nil {
		return nil, fmt.Errorf("unable to decode hex response, %w", err)
	}

	return delegatorReward, nil
}
