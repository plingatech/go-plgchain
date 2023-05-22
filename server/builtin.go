package server

import (
	"github.com/plingatech/go-plgchain/chain"
	"github.com/plingatech/go-plgchain/consensus"
	consensusDev "github.com/plingatech/go-plgchain/consensus/dev"
	consensusDummy "github.com/plingatech/go-plgchain/consensus/dummy"
	consensusIBFT "github.com/plingatech/go-plgchain/consensus/ibft"
	consensusPlgBFT "github.com/plingatech/go-plgchain/consensus/plgbft"
	"github.com/plingatech/go-plgchain/secrets"
	"github.com/plingatech/go-plgchain/secrets/awsssm"
	"github.com/plingatech/go-plgchain/secrets/gcpssm"
	"github.com/plingatech/go-plgchain/secrets/hashicorpvault"
	"github.com/plingatech/go-plgchain/secrets/local"
	"github.com/plingatech/go-plgchain/state"
)

type GenesisFactoryHook func(config *chain.Chain, engineName string) func(*state.Transition) error

type ConsensusType string

const (
	DevConsensus    ConsensusType = "dev"
	IBFTConsensus   ConsensusType = "ibft"
	PlgBFTConsensus ConsensusType = "plgbft"
	DummyConsensus  ConsensusType = "dummy"
)

var consensusBackends = map[ConsensusType]consensus.Factory{
	DevConsensus:    consensusDev.Factory,
	IBFTConsensus:   consensusIBFT.Factory,
	PlgBFTConsensus: consensusPlgBFT.Factory,
	DummyConsensus:  consensusDummy.Factory,
}

// secretsManagerBackends defines the SecretManager factories for different
// secret management solutions
var secretsManagerBackends = map[secrets.SecretsManagerType]secrets.SecretsManagerFactory{
	secrets.Local:          local.SecretsManagerFactory,
	secrets.HashicorpVault: hashicorpvault.SecretsManagerFactory,
	secrets.AWSSSM:         awsssm.SecretsManagerFactory,
	secrets.GCPSSM:         gcpssm.SecretsManagerFactory,
}

var genesisCreationFactory = map[ConsensusType]GenesisFactoryHook{
	PlgBFTConsensus: consensusPlgBFT.GenesisPostHookFactory,
}

func ConsensusSupported(value string) bool {
	_, ok := consensusBackends[ConsensusType(value)]

	return ok
}
