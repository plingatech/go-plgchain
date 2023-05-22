package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/plingatech/go-plgchain/command/backup"
	"github.com/plingatech/go-plgchain/command/bridge"
	"github.com/plingatech/go-plgchain/command/genesis"
	"github.com/plingatech/go-plgchain/command/helper"
	"github.com/plingatech/go-plgchain/command/ibft"
	"github.com/plingatech/go-plgchain/command/license"
	"github.com/plingatech/go-plgchain/command/monitor"
	"github.com/plingatech/go-plgchain/command/peers"
	"github.com/plingatech/go-plgchain/command/plgbft"
	"github.com/plingatech/go-plgchain/command/plgbftmanifest"
	"github.com/plingatech/go-plgchain/command/plgbftsecrets"
	"github.com/plingatech/go-plgchain/command/regenesis"
	"github.com/plingatech/go-plgchain/command/rootchain"
	"github.com/plingatech/go-plgchain/command/secrets"
	"github.com/plingatech/go-plgchain/command/server"
	"github.com/plingatech/go-plgchain/command/status"
	"github.com/plingatech/go-plgchain/command/txpool"
	"github.com/plingatech/go-plgchain/command/version"
	"github.com/plingatech/go-plgchain/command/whitelist"
)

type RootCommand struct {
	baseCmd *cobra.Command
}

func NewRootCommand() *RootCommand {
	rootCommand := &RootCommand{
		baseCmd: &cobra.Command{
			Short: "Go Plgchain is a framework for building Ethereum-compatible Blockchain networks",
		},
	}

	helper.RegisterJSONOutputFlag(rootCommand.baseCmd)

	rootCommand.registerSubCommands()

	return rootCommand
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		version.GetCommand(),
		txpool.GetCommand(),
		status.GetCommand(),
		secrets.GetCommand(),
		peers.GetCommand(),
		rootchain.GetCommand(),
		monitor.GetCommand(),
		ibft.GetCommand(),
		backup.GetCommand(),
		genesis.GetCommand(),
		server.GetCommand(),
		whitelist.GetCommand(),
		license.GetCommand(),
		plgbftsecrets.GetCommand(),
		plgbft.GetCommand(),
		plgbftmanifest.GetCommand(),
		bridge.GetCommand(),
		regenesis.GetCommand(),
	)
}

func (rc *RootCommand) Execute() {
	if err := rc.baseCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
