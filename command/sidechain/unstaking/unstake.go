package unstaking

import (
	"fmt"
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

var params unstakeParams

func GetCommand() *cobra.Command {
	unstakeCmd := &cobra.Command{
		Use:     "unstake",
		Short:   "Unstakes the amount sent for validator or undelegates amount from validator",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(unstakeCmd)
	setFlags(unstakeCmd)

	return unstakeCmd
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
		"indicates if its a self unstake action",
	)

	cmd.Flags().Uint64Var(
		&params.amount,
		sidechainHelper.AmountFlag,
		0,
		"amount to unstake or undelegate amount from validator",
	)

	cmd.Flags().StringVar(
		&params.undelegateAddress,
		undelegateAddressFlag,
		"",
		"account address from which amount will be undelegated",
	)

	cmd.MarkFlagsMutuallyExclusive(sidechainHelper.SelfFlag, undelegateAddressFlag)
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
		if params.ether {
			encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["unstake"].Encode([]interface{}{params.amount * 1000000000000000000})
		} else {
			encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["unstake"].Encode([]interface{}{params.amount})
		}
		if err != nil {
			return err
		}
	} else {
		if params.ether {
			encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["undelegate"].Encode(
				[]interface{}{ethgo.HexToAddress(params.undelegateAddress), params.amount * 1000000000000000000})
		} else {
			encoded, err = contractsapi.ChildValidatorSet.Abi.Methods["undelegate"].Encode(
				[]interface{}{ethgo.HexToAddress(params.undelegateAddress), params.amount})
		}
		if err != nil {
			return err
		}
	}

	txn := &ethgo.Transaction{
		From:     validatorAccount.Ecdsa.Address(),
		Input:    encoded,
		To:       (*ethgo.Address)(&contracts.ValidatorSetContract),
		GasPrice: sidechainHelper.DefaultGasPrice,
	}

	receipt, err := txRelayer.SendTransaction(txn, validatorAccount.Ecdsa)
	if err != nil {
		return err
	}

	if receipt.Status == uint64(types.ReceiptFailed) {
		return fmt.Errorf("unstake transaction failed on block %d", receipt.BlockNumber)
	}

	result := &unstakeResult{
		validatorAddress: validatorAccount.Ecdsa.Address().String(),
	}

	var (
		unstakedEvent    contractsapi.UnstakedEvent
		undelegatedEvent contractsapi.UndelegatedEvent
		foundLog         bool
	)

	// check the logs to check for the result
	for _, log := range receipt.Logs {
		doesMatch, err := unstakedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if doesMatch { // its an unstake function call
			result.isSelfUnstake = true
			result.amount = unstakedEvent.Amount.Uint64()
			foundLog = true

			break
		}

		doesMatch, err = undelegatedEvent.ParseLog(log)
		if err != nil {
			return err
		}

		if doesMatch {
			result.amount = undelegatedEvent.Amount.Uint64()
			result.undelegatedFrom = undelegatedEvent.Validator.String()
			foundLog = true

			break
		}
	}

	if !foundLog {
		return fmt.Errorf("could not find an appropriate log in receipt that unstake or undelegate happened")
	}

	outputter.WriteCommandResult(result)

	return nil
}
