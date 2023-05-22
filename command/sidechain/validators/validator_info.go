package validators

import (
	"fmt"

	"github.com/plingatech/go-plgchain/command"
	"github.com/plingatech/go-plgchain/command/helper"
	"github.com/plingatech/go-plgchain/command/plgbftsecrets"
	sidechainHelper "github.com/plingatech/go-plgchain/command/sidechain"
	"github.com/plingatech/go-plgchain/txrelayer"
	"github.com/spf13/cobra"
)

var (
	params validatorInfoParams
)

func GetCommand() *cobra.Command {
	validatorInfoCmd := &cobra.Command{
		Use:     "validator-info",
		Short:   "Gets validator info",
		PreRunE: runPreRun,
		RunE:    runCommand,
	}

	helper.RegisterJSONRPCFlag(validatorInfoCmd)
	setFlags(validatorInfoCmd)

	return validatorInfoCmd
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

	cmd.MarkFlagsMutuallyExclusive(plgbftsecrets.AccountDirFlag, plgbftsecrets.AccountConfigFlag)
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

	txRelayer, err := txrelayer.NewTxRelayer(txrelayer.WithIPAddress(params.jsonRPC))
	if err != nil {
		return err
	}

	validatorAddr := validatorAccount.Ecdsa.Address()

	validatorInfo, err := sidechainHelper.GetValidatorInfo(validatorAddr, txRelayer)
	if err != nil {
		return fmt.Errorf("failed to get validator info for %s: %w", validatorAddr, err)
	}

	outputter.WriteCommandResult(&validatorsInfoResult{
		address:             validatorInfo.Address.String(),
		stake:               validatorInfo.Stake.Uint64(),
		totalStake:          validatorInfo.TotalStake.Uint64(),
		commission:          validatorInfo.Commission.Uint64(),
		withdrawableRewards: validatorInfo.WithdrawableRewards.Uint64(),
		active:              validatorInfo.Active,
	})

	return nil
}
