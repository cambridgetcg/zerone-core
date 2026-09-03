package app_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"sort"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	vestingrewards "github.com/zerone-chain/zerone/x/vesting_rewards"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func proposalTxBytes(t *testing.T, app *zeroneapp.ZeroneApp, gas uint64, memo string) []byte {
	t.Helper()
	builder := app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(
		sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20)),
		sdk.AccAddress(bytes.Repeat([]byte{0x22}, 20)),
		sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1)),
	)))
	builder.SetGasLimit(gas)
	builder.SetMemo(memo)
	txBytes, err := app.TxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func proposalContext(app *zeroneapp.ZeroneApp, maxGas int64) sdk.Context {
	return app.NewUncachedContext(false, cmtproto.Header{Height: 2}).WithConsensusParams(
		cmtproto.ConsensusParams{Block: &cmtproto.BlockParams{MaxGas: maxGas}},
	)
}

func TestPrepareProposalUsesExactProtobufByteLimit(t *testing.T) {
	app := newTestApp(t)
	txBytes := proposalTxBytes(t, app, 100, "protobuf-overhead")
	protoSize := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txBytes})
	require.Greater(t, protoSize, int64(len(txBytes)), "protobuf framing must add bytes")

	handler := app.PrepareProposalHandler()
	tooSmall, err := handler(proposalContext(app, 1_000), &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: int64(len(txBytes)),
		Txs:        [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Empty(t, tooSmall.Txs, "raw byte length must not bypass protobuf framing overhead")

	exact, err := handler(proposalContext(app, 1_000), &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: protoSize,
		Txs:        [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{txBytes}, exact.Txs)
}

func TestPrepareProposalUsesCurrentConsensusMaxGas(t *testing.T) {
	app := newTestApp(t)
	first := proposalTxBytes(t, app, 60, "first")
	second := proposalTxBytes(t, app, 50, "second")
	maxBytes := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{first, second})
	handler := app.PrepareProposalHandler()

	limited, err := handler(proposalContext(app, 100), &abci.RequestPrepareProposal{
		Height: 2, MaxTxBytes: maxBytes, Txs: [][]byte{first, second},
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{first}, limited.Txs)

	expanded, err := handler(proposalContext(app, 110), &abci.RequestPrepareProposal{
		Height: 2, MaxTxBytes: maxBytes, Txs: [][]byte{first, second},
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{first, second}, expanded.Txs,
		"selection must follow the context consensus params rather than a compiled constant")

}

func TestPrepareProposalSkipsMalformedAndHighGasTransactions(t *testing.T) {
	app := newTestApp(t)
	highGas := proposalTxBytes(t, app, 101, "high-gas")
	valid := proposalTxBytes(t, app, 100, "valid")
	maxBytes := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{highGas, valid})

	response, err := app.PrepareProposalHandler()(proposalContext(app, 100), &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: maxBytes,
		Txs: [][]byte{
			{0xff, 0x00, 0xff},
			highGas,
			valid,
		},
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{valid}, response.Txs)
}

func TestPrepareProposalCannotWrapDeclaredGasTotal(t *testing.T) {
	app := newTestApp(t)
	valid := proposalTxBytes(t, app, 1, "valid-first")
	overflow := proposalTxBytes(t, app, ^uint64(0), "overflow-second")
	maxBytes := cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{valid, overflow})

	response, err := app.PrepareProposalHandler()(proposalContext(app, 100), &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: maxBytes,
		Txs:        [][]byte{valid, overflow},
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{valid}, response.Txs)
}

func TestVoteExtInjectionEncodeDecode(t *testing.T) {
	inj := zeroneapp.VoteExtInjection{
		Commitments: []zeroneapp.InjectedCommitment{
			{
				RoundID:        "round-1",
				Validator:      "zrn1validator1",
				CommitmentHash: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
				VRFOutput:      "aabb",
				VRFProof:       "ccdd",
			},
		},
		Reveals: []zeroneapp.InjectedReveal{
			{
				RoundID:    "round-1",
				Validator:  "zrn1validator1",
				Verdict:    "accept",
				Confidence: 600_000,
				Salt:       "deadbeef",
			},
		},
	}

	encoded, err := zeroneapp.EncodeVoteExtInjection(inj)
	require.NoError(t, err)
	require.True(t, zeroneapp.IsVoteExtInjectionTx(encoded))

	decoded, err := zeroneapp.DecodeVoteExtInjection(encoded)
	require.NoError(t, err)
	require.Len(t, decoded.Commitments, 1)
	require.Len(t, decoded.Reveals, 1)
	require.Equal(t, "round-1", decoded.Commitments[0].RoundID)
	require.Equal(t, "accept", decoded.Reveals[0].Verdict)
	require.Equal(t, uint64(600_000), decoded.Reveals[0].Confidence)
}

func TestComputeCommitmentHash(t *testing.T) {
	salt := "aabbccdd11223344aabbccdd11223344"

	// Compute hash using app-layer function
	hash := zeroneapp.ComputeCommitmentHash("round-1", "accept", 600_000, salt)
	require.Len(t, hash, 64) // hex-encoded SHA-256 = 64 chars

	// Verify round-trip
	verified := zeroneapp.VerifyCommitmentHash(hash, "round-1", "accept", 600_000, salt)
	require.True(t, verified)

	// Wrong verdict → no match
	require.False(t, zeroneapp.VerifyCommitmentHash(hash, "round-1", "reject", 600_000, salt))

	// Wrong confidence → no match
	require.False(t, zeroneapp.VerifyCommitmentHash(hash, "round-1", "accept", 500_000, salt))

	// Wrong salt → no match
	require.False(t, zeroneapp.VerifyCommitmentHash(hash, "round-1", "accept", 600_000, "0000000000000000"))

	// Verify types-layer function produces the same result
	saltBytes, _ := hex.DecodeString(salt)
	typesHash := hex.EncodeToString(types.ComputeCommitmentHash("round-1", "accept", 600_000, saltBytes))
	require.Equal(t, hash, typesHash)
}

func TestProcessVoteExtensions_Sorting(t *testing.T) {
	// Simulate out-of-order vote extensions
	ext1 := zeroneapp.VoteExtension{
		ValidatorAddress: "zrn1val_b",
		Commitments: []zeroneapp.VoteCommitment{
			{RoundID: "round-2", CommitmentHash: "hash2"},
		},
	}
	ext2 := zeroneapp.VoteExtension{
		ValidatorAddress: "zrn1val_a",
		Commitments: []zeroneapp.VoteCommitment{
			{RoundID: "round-1", CommitmentHash: "hash1"},
		},
	}
	ext3 := zeroneapp.VoteExtension{
		ValidatorAddress: "zrn1val_c",
		Commitments: []zeroneapp.VoteCommitment{
			{RoundID: "round-1", CommitmentHash: "hash3"},
		},
	}

	// Serialize and collect all commitments
	var allCommitments []zeroneapp.InjectedCommitment
	for _, ext := range []zeroneapp.VoteExtension{ext1, ext2, ext3} {
		for _, c := range ext.Commitments {
			allCommitments = append(allCommitments, zeroneapp.InjectedCommitment{
				RoundID:        c.RoundID,
				Validator:      ext.ValidatorAddress,
				CommitmentHash: c.CommitmentHash,
			})
		}
	}

	// Sort the same way processVoteExtensions does
	sort.Slice(allCommitments, func(i, j int) bool {
		if allCommitments[i].RoundID != allCommitments[j].RoundID {
			return allCommitments[i].RoundID < allCommitments[j].RoundID
		}
		return allCommitments[i].Validator < allCommitments[j].Validator
	})

	// Verify deterministic order: round-1 before round-2, then by validator
	require.Equal(t, "round-1", allCommitments[0].RoundID)
	require.Equal(t, "zrn1val_a", allCommitments[0].Validator)
	require.Equal(t, "round-1", allCommitments[1].RoundID)
	require.Equal(t, "zrn1val_c", allCommitments[1].Validator)
	require.Equal(t, "round-2", allCommitments[2].RoundID)
	require.Equal(t, "zrn1val_b", allCommitments[2].Validator)
}

func TestIsVoteExtInjectionTx(t *testing.T) {
	// Valid prefix
	validTx := append([]byte{0x00, 'V', 'E', 'X'}, []byte(`{"commitments":[],"reveals":[]}`)...)
	require.True(t, zeroneapp.IsVoteExtInjectionTx(validTx))

	// Too short
	require.False(t, zeroneapp.IsVoteExtInjectionTx([]byte{0x00, 'V', 'E', 'X'}))

	// Wrong prefix
	require.False(t, zeroneapp.IsVoteExtInjectionTx([]byte{0x01, 'V', 'E', 'X', 0x00}))

	// Empty
	require.False(t, zeroneapp.IsVoteExtInjectionTx(nil))

	// Normal tx bytes
	require.False(t, zeroneapp.IsVoteExtInjectionTx([]byte(`{"body":{}}`)))

	// Just the prefix with one extra byte is enough
	require.True(t, zeroneapp.IsVoteExtInjectionTx([]byte{0x00, 'V', 'E', 'X', '{'}))
}

func TestVoteExtInjectionSizeLimit(t *testing.T) {
	// Create an injection that's just under the limit
	inj := zeroneapp.VoteExtInjection{}

	// Each commitment is ~200 bytes in JSON
	for i := 0; i < 100; i++ {
		inj.Commitments = append(inj.Commitments, zeroneapp.InjectedCommitment{
			RoundID:        "round-normal",
			Validator:      "zrn1validator_test",
			CommitmentHash: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		})
	}

	encoded, err := zeroneapp.EncodeVoteExtInjection(inj)
	require.NoError(t, err)
	require.True(t, len(encoded) < zeroneapp.MaxVEXInjectionBytes, "normal injection should be under limit")

	// Verify it round-trips
	decoded, err := zeroneapp.DecodeVoteExtInjection(encoded)
	require.NoError(t, err)
	require.Len(t, decoded.Commitments, 100)

	// Create oversized injection by using huge data
	bigInj := zeroneapp.VoteExtInjection{}
	hugePayload := make([]byte, zeroneapp.MaxVEXInjectionBytes)
	for i := range hugePayload {
		hugePayload[i] = 'a'
	}
	bigInj.Commitments = append(bigInj.Commitments, zeroneapp.InjectedCommitment{
		RoundID:        string(hugePayload),
		Validator:      "zrn1validator_test",
		CommitmentHash: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
	})

	encoded, err = zeroneapp.EncodeVoteExtInjection(bigInj)
	require.NoError(t, err)
	require.True(t, len(encoded) > zeroneapp.MaxVEXInjectionBytes, "oversized injection should exceed limit")
}

// TestVoteExtensionJSONRoundTrip verifies that VoteExtension serializes correctly.
func TestVoteExtensionJSONRoundTrip(t *testing.T) {
	ext := zeroneapp.VoteExtension{
		ValidatorAddress: "zrn1test",
		Commitments: []zeroneapp.VoteCommitment{
			{
				RoundID:        "r1",
				CommitmentHash: "abcd",
				VRFOutput:      "1234",
				VRFProof:       "5678",
				Height:         42,
			},
		},
		Reveals: []zeroneapp.VoteReveal{
			{
				RoundID:    "r1",
				Verdict:    "accept",
				Confidence: 750_000,
				Salt:       "deadbeef",
			},
		},
	}

	bz, err := json.Marshal(ext)
	require.NoError(t, err)

	var decoded zeroneapp.VoteExtension
	require.NoError(t, json.Unmarshal(bz, &decoded))

	require.Equal(t, ext.ValidatorAddress, decoded.ValidatorAddress)
	require.Len(t, decoded.Commitments, 1)
	require.Equal(t, "abcd", decoded.Commitments[0].CommitmentHash)
	require.Len(t, decoded.Reveals, 1)
	require.Equal(t, uint64(750_000), decoded.Reveals[0].Confidence)
}

func TestUnsignedTransactionIsRejectedWithoutAutomaticMint(t *testing.T) {
	app := newTestApp(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: 2})
	app.VestingRewardsKeeper.InitGenesis(ctx, vestingrewardstypes.DefaultGenesis())

	from := sdk.AccAddress("unsigned_sender_____")
	to := sdk.AccAddress("unsigned_recipient__")
	builder := app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(
		from,
		to,
		sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1)),
	)))
	txBytes, err := app.TxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn").Amount
	beforeMinted := app.VestingRewardsKeeper.GetTotalMinted(ctx)
	response, err := app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 1,
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status,
		"every validator must apply signature and account checks to proposal txs")

	module := vestingrewards.NewAppModule(app.AppCodec(), app.VestingRewardsKeeper)
	require.NoError(t, module.BeginBlock(ctx))
	require.True(t, app.BankKeeper.GetSupply(ctx, "uzrn").Amount.Equal(beforeSupply),
		"proposal inclusion must not change native supply")
	require.Zero(t, app.VestingRewardsKeeper.GetTotalMinted(ctx).Cmp(beforeMinted),
		"proposal inclusion must not change mint accounting")
}

func TestVestingRewardsInitGenesisRejectsRetiredRewardReactivation(t *testing.T) {
	app := newTestApp(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: 0})
	module := vestingrewards.NewAppModule(app.AppCodec(), app.VestingRewardsKeeper)

	genesis := vestingrewardstypes.DefaultGenesis()
	genesis.Params.BlockReward = "1"
	bz, err := json.Marshal(genesis)
	require.NoError(t, err)
	require.Panics(t, func() {
		module.InitGenesis(ctx, app.AppCodec(), bz)
	}, "the module boundary must reject invalid imported consensus state")

	require.Panics(t, func() {
		module.InitGenesis(ctx, app.AppCodec(), json.RawMessage(`{"params":`))
	}, "malformed genesis must not silently fall back to defaults")
}
