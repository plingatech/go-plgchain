package plgbft

import (
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/plingatech/go-plgchain/consensus"
	"github.com/plingatech/go-plgchain/consensus/ibft/signer"
	bls "github.com/plingatech/go-plgchain/consensus/plgbft/signer"
	"github.com/plingatech/go-plgchain/consensus/plgbft/wallet"
	"github.com/plingatech/go-plgchain/helper/progress"
	"github.com/plingatech/go-plgchain/txpool"
	"github.com/plingatech/go-plgchain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// the test initializes plgbft and chain mock (map of headers) after which a new header is verified
// firstly, two invalid situation of header verifications are triggered (missing Committed field and invalid validators for ParentCommitted)
// afterwards, valid inclusion into the block chain is checked
// and at the end there is a situation when header is already a part of blockchain
func TestPlgbft_VerifyHeader(t *testing.T) {
	t.Parallel()

	const (
		allValidatorsSize = 6 // overall there are 6 validators
		validatorSetSize  = 5 // only 5 validators are active at the time
		fixedEpochSize    = uint64(10)
	)

	updateHeaderExtra := func(header *types.Header,
		validators *ValidatorSetDelta,
		parentSignature *Signature,
		checkpointData *CheckpointData,
		committedAccounts []*wallet.Account) *Signature {
		extra := &Extra{
			Validators: validators,
			Parent:     parentSignature,
			Checkpoint: checkpointData,
			Committed:  &Signature{},
			Seal:       []byte{},
		}

		if extra.Checkpoint == nil {
			extra.Checkpoint = &CheckpointData{}
		}

		header.ExtraData = append(make([]byte, ExtraVanity), extra.MarshalRLPTo(nil)...)
		header.ComputeHash()

		if len(committedAccounts) > 0 {
			checkpointHash, err := extra.Checkpoint.Hash(0, header.Number, header.Hash)
			require.NoError(t, err)

			extra.Committed = createSignature(t, committedAccounts, checkpointHash, bls.DomainCheckpointManager)
			header.ExtraData = append(make([]byte, signer.IstanbulExtraVanity), extra.MarshalRLPTo(nil)...)
		}

		return extra.Committed
	}

	// create all validators
	validators := newTestValidators(t, allValidatorsSize)

	// create configuration
	plgBftConfig := PlgBFTConfig{
		InitialValidatorSet: validators.getParamValidators(),
		EpochSize:           fixedEpochSize,
		SprintSize:          5,
	}

	validatorSet := validators.getPublicIdentities()
	accounts := validators.getPrivateIdentities()

	// calculate validators before and after the end of the first epoch
	validatorSetParent, validatorSetCurrent := validatorSet[:len(validatorSet)-1], validatorSet[1:]
	accountSetParent, accountSetCurrent := accounts[:len(accounts)-1], accounts[1:]

	// create header map to simulate blockchain
	headersMap := &testHeadersMap{}

	// create genesis header
	genesisDelta, err := createValidatorSetDelta(nil, validatorSetParent)
	require.NoError(t, err)

	genesisHeader := &types.Header{Number: 0}
	updateHeaderExtra(genesisHeader, genesisDelta, nil, nil, nil)

	// add genesis header to map
	headersMap.addHeader(genesisHeader)

	// create headers from 1 to 9
	for i := uint64(1); i < plgBftConfig.EpochSize; i++ {
		delta, err := createValidatorSetDelta(validatorSetParent, validatorSetParent)
		require.NoError(t, err)

		header := &types.Header{Number: i}
		updateHeaderExtra(header, delta, nil, &CheckpointData{EpochNumber: 1}, nil)

		// add headers from 1 to 9 to map (blockchain imitation)
		headersMap.addHeader(header)
	}

	// mock blockchain
	blockchainMock := new(blockchainMock)
	blockchainMock.On("GetHeaderByNumber", mock.Anything).Return(headersMap.getHeader)
	blockchainMock.On("GetHeaderByHash", mock.Anything).Return(headersMap.getHeaderByHash)

	// create plgbft with appropriate mocks
	plgbft := &Plgbft{
		closeCh:         make(chan struct{}),
		logger:          hclog.NewNullLogger(),
		consensusConfig: &plgBftConfig,
		blockchain:      blockchainMock,
		validatorsCache: newValidatorsSnapshotCache(
			hclog.NewNullLogger(),
			newTestState(t),
			blockchainMock,
		),
	}

	// create parent header (block 10)
	parentDelta, err := createValidatorSetDelta(validatorSetParent, validatorSetCurrent)
	require.NoError(t, err)

	parentHeader := &types.Header{
		Number:    plgBftConfig.EpochSize,
		Timestamp: uint64(time.Now().UTC().UnixMilli()),
	}
	parentCommitment := updateHeaderExtra(parentHeader, parentDelta, nil, &CheckpointData{EpochNumber: 1}, accountSetParent)

	// add parent header to map
	headersMap.addHeader(parentHeader)

	// create current header (block 11) with all appropriate fields required for validation
	currentDelta, err := createValidatorSetDelta(validatorSetCurrent, validatorSetCurrent)
	require.NoError(t, err)

	currentHeader := &types.Header{
		Number:     plgBftConfig.EpochSize + 1,
		ParentHash: parentHeader.Hash,
		Timestamp:  parentHeader.Timestamp + 1,
		MixHash:    PlgBFTMixDigest,
		Difficulty: 1,
	}
	updateHeaderExtra(currentHeader, currentDelta, nil,
		&CheckpointData{
			EpochNumber:           2,
			CurrentValidatorsHash: types.StringToHash("Foo"),
			NextValidatorsHash:    types.StringToHash("Bar"),
		}, nil)

	currentHeader.Hash[0] = currentHeader.Hash[0] + 1
	assert.ErrorContains(t, plgbft.VerifyHeader(currentHeader), "invalid header hash")

	// omit Parent field (parent signature) intentionally
	updateHeaderExtra(currentHeader, currentDelta, nil,
		&CheckpointData{
			EpochNumber:           1,
			CurrentValidatorsHash: types.StringToHash("Foo"),
			NextValidatorsHash:    types.StringToHash("Bar")},
		accountSetCurrent)

	// since parent signature is intentionally disregarded the following error is expected
	assert.ErrorContains(t, plgbft.VerifyHeader(currentHeader), "failed to verify signatures for parent of block")

	updateHeaderExtra(currentHeader, currentDelta, parentCommitment,
		&CheckpointData{
			EpochNumber:           1,
			CurrentValidatorsHash: types.StringToHash("Foo"),
			NextValidatorsHash:    types.StringToHash("Bar")},
		accountSetCurrent)

	assert.NoError(t, plgbft.VerifyHeader(currentHeader))

	// clean validator snapshot cache (re-instantiate it), submit invalid validator set for parent signature and expect the following error
	plgbft.validatorsCache = newValidatorsSnapshotCache(hclog.NewNullLogger(), newTestState(t), blockchainMock)
	assert.NoError(t, plgbft.validatorsCache.storeSnapshot(&validatorSnapshot{Epoch: 0, Snapshot: validatorSetCurrent})) // invalid validator set is submitted
	assert.NoError(t, plgbft.validatorsCache.storeSnapshot(&validatorSnapshot{Epoch: 1, Snapshot: validatorSetCurrent}))
	assert.ErrorContains(t, plgbft.VerifyHeader(currentHeader), "failed to verify signatures for parent of block")

	// clean validators cache again and set valid snapshots
	plgbft.validatorsCache = newValidatorsSnapshotCache(hclog.NewNullLogger(), newTestState(t), blockchainMock)
	assert.NoError(t, plgbft.validatorsCache.storeSnapshot(&validatorSnapshot{Epoch: 0, Snapshot: validatorSetParent}))
	assert.NoError(t, plgbft.validatorsCache.storeSnapshot(&validatorSnapshot{Epoch: 1, Snapshot: validatorSetCurrent}))
	assert.NoError(t, plgbft.VerifyHeader(currentHeader))

	// add current header to the blockchain (headersMap) and try validating again
	headersMap.addHeader(currentHeader)
	assert.NoError(t, plgbft.VerifyHeader(currentHeader))
}

func TestPlgbft_Close(t *testing.T) {
	t.Parallel()

	syncer := &syncerMock{}
	syncer.On("Close", mock.Anything).Return(error(nil)).Once()

	plgbft := Plgbft{
		closeCh: make(chan struct{}),
		syncer:  syncer,
		runtime: &consensusRuntime{stateSyncManager: &dummyStateSyncManager{}},
	}

	assert.NoError(t, plgbft.Close())

	<-plgbft.closeCh

	syncer.AssertExpectations(t)

	errExpected := errors.New("something")
	syncer.On("Close", mock.Anything).Return(errExpected).Once()

	plgbft.closeCh = make(chan struct{})

	assert.Error(t, errExpected, plgbft.Close())

	select {
	case <-plgbft.closeCh:
		assert.Fail(t, "channel closing is invoked")
	case <-time.After(time.Millisecond * 100):
	}

	syncer.AssertExpectations(t)
}

func TestPlgbft_GetSyncProgression(t *testing.T) {
	t.Parallel()

	result := &progress.Progression{}

	syncer := &syncerMock{}
	syncer.On("GetSyncProgression", mock.Anything).Return(result).Once()

	plgbft := Plgbft{
		syncer: syncer,
	}

	assert.Equal(t, result, plgbft.GetSyncProgression())
}

func Test_Factory(t *testing.T) {
	t.Parallel()

	const epochSize = uint64(141)

	txPool := &txpool.TxPool{}

	params := &consensus.Params{
		TxPool: txPool,
		Logger: hclog.Default(),
		Config: &consensus.Config{
			Config: map[string]interface{}{
				"EpochSize": epochSize,
			},
		},
	}

	r, err := Factory(params)

	require.NoError(t, err)
	require.NotNil(t, r)

	plgbft, ok := r.(*Plgbft)
	require.True(t, ok)

	assert.Equal(t, txPool, plgbft.txPool)
	assert.Equal(t, epochSize, plgbft.consensusConfig.EpochSize)
	assert.Equal(t, params, plgbft.config)
}
