package plgbft

import (
	"github.com/plingatech/go-plgchain/command/sidechain/registration"
	"github.com/plingatech/go-plgchain/command/sidechain/staking"
	"github.com/plingatech/go-plgchain/command/sidechain/unstaking"
	"github.com/plingatech/go-plgchain/command/sidechain/validators"

	"github.com/plingatech/go-plgchain/command/sidechain/whitelist"
	"github.com/plingatech/go-plgchain/command/sidechain/withdraw"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	plgbftCmd := &cobra.Command{
		Use:   "plgbft",
		Short: "Plgbft command",
	}

	plgbftCmd.AddCommand(
		staking.GetCommand(),
		unstaking.GetCommand(),
		withdraw.GetCommand(),
		validators.GetCommand(),
		whitelist.GetCommand(),
		registration.GetCommand(),
	)

	return plgbftCmd
}
