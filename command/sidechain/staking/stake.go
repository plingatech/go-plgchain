package staking

import (
	"fmt"
	"math/big"
	"time"

	"github.com/plingatech/go-plgchain/command"
	"github.com/plingatech/go-plgchain/command/helper"
	"github.com/plingatech/go-plgchain/command/plgbftsecrets"
	sidechainHelper "github.com/plingatech/go-plgchain/command/sidechain"
	"github.com/plingatech/go-plgchain/consensus/plgbft/contractsapi"
	"github.com/plingatech/go-plgchain/contracts"
	"github.com/plingatech/go-plgchain/txrelayer"
	"github.com/plingatech/go-plgchain/types"
	"github.com/spf13/cobra"
	"github.com/umbracle/ethgo"
)

var (
	params stakeParams
)

func GetCommand() *cobra.Command {
	stakeCmd := &cobra.Command{
		Use:     "stake",
		Short:   "Stakes the amount sent for validator or delegates its stake to another account",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(stakeCmd)
	setFlags(stakeCmd)

	return stakeCmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.accountDir,
		plgbftsecrets.AccountDirFlag,
		"",
		plgbftsecrets.AccountDirFlagDesc,
	)

	cmd.Flags().StringVar(
		&params.accountConfig,
		plgbftsecrets.AccountConfigFlag,
		"",
		plgbftsecrets.AccountConfigFlagDesc,
	)

	cmd.Flags().BoolVar(
		&params.ether,
		sidechainHelper.EtherFlag,
		false,
		"indicates if its a ether unit default wei",
	)

	cmd.Flags().BoolVar(
		&params.self,
		sidechainHelper.SelfFlag,
		false,
		"indicates if its a self stake action",
	)

	cmd.Flags().Uint64Var(
		&params.amount,
		sidechainHelper.AmountFlag,
		0,
		"amount to stake or delegate to another account",
	)

	cmd.Flags().StringVar(
		&params.delegateAddress,
		delegateAddressFlag,
		"",
		"account address to which stake should be delegated",
	)

	cmd.MarkFlagsMutuallyExclusive(sidechainHelper.SelfFlag, delegateAddressFlag)
	cmd.MarkFlagsMutuallyExclusive(plgbftsecrets.AccountDirFlag, plgbftsecrets.AccountConfigFlag)
	cmd.MarkFlagsMutuallyExclusive(sidechainHelper.EtherFlag)
}

func runPreRun(cmd *cobra.Command, _ []string) error {
	params.jsonRPC = helper.GetJSONRPCAddress(cmd)

	return params.validateFlags()
}

func runCommand(cmd *cobra.Command, _ []string) error {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	validatorAccount, err := sidechainHelper.GetAccount(params.accountDir, params.accountConfig)
	if err != nil {
		return err
	}

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPC),
		txrelayer.WithReceiptTimeout(150*time.Millisecond))
	if err != nil {
		return err
	}

	var encoded []byte
	if params.self {
		encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["stake"].Encode([]interface{}{})
		if err != nil {
			return err
		}
	} else {
		delegateToAddress := types.StringToAddress(params.delegateAddress)
		encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["delegate"].Encode(
			[]interface{}{ethgo.Address(delegateToAddress), false})
		if err != nil {
			return err
		}
	}

	multiplier := big.NewInt(1000000000000000000)
	value := new(big.Int).SetUint64(params.amount)

	if params.ether {
		value = new(big.Int).Mul(new(big.Int).SetUint64(params.amount), multiplier)
	}

	txn := &ethgo.Transaction{
		From:     validatorAccount.Ecdsa.Address(),
		Input:    encoded,
		To:       (*ethgo.Address)(&contracts.ValidatorSetContract),
		Value:    value,
		GasPrice: sidechainHelper.DefaultGasPrice,
	}

	receipt, err := txRelayer.SendTransaction(txn, validatorAccount.Ecdsa)
	if err != nil {
		return err
	}

	if receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("staking transaction failed on block %d", receipt.BlockNumber)
	}

	result := &stakeResult{
		validatorAddress: validatorAccount.Ecdsa.Address().String(),
	}

	var (
		stakedEvent    contractsapi.StakedEvent
		delegatedEvent contractsapi.DelegatedEvent
		foundLog       bool
	)

	// check the logs to check for the result
	for _, log := range receipt.Logs {
		doesMatch, err := stakedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if doesMatch { // its a stake function call
			result.isSelfStake = true
			result.amount = stakedEvent.Amount.Uint64()
			foundLog = true

			break
		}

		doesMatch, err = delegatedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if doesMatch {
			result.amount = delegatedEvent.Amount.Uint64()
			result.delegatedTo = delegatedEvent.Validator.String()
			foundLog = true

			break
		}
	}

	if !foundLog {
		return fmt.Errorf("could not find an appropriate log in receipt that stake or delegate happened")
	}

	outputter.WriteCommandResult(result)

	return nil
}
