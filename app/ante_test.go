package app

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ---------- lookupMsgGas Tests ----------

func TestLookupMsgGas_KnownTypes(t *testing.T) {
	tests := []struct {
		msgTypeURL string
		wantGas    uint64
	}{
		{"/cosmos.bank.v1beta1.MsgSend", TransactionGasCosts["transfer"]},
		{"/cosmos.bank.v1beta1.MsgMultiSend", TransactionGasCosts["transfer"] * 2},
		{"/zerone.knowledge.v1.MsgSubmitClaim", TransactionGasCosts["claim_submit"]},
		{"/zerone.staking.v1.MsgRegisterValidator", TransactionGasCosts["register_validator"]},
		{"/zerone.emergency.v1.MsgProposeHalt", TransactionGasCosts["propose_halt"]},
		{"/zerone.alignment.v1.MsgActivate", TransactionGasCosts["activate_alignment"]},
		{"/zerone.liquiditypool.v1.MsgCreatePool", TransactionGasCosts["create_pool"]},
	}

	for _, tc := range tests {
		got := lookupMsgGas(tc.msgTypeURL)
		if got != tc.wantGas {
			t.Errorf("lookupMsgGas(%q) = %d, want %d", tc.msgTypeURL, got, tc.wantGas)
		}
	}
}

func TestLookupMsgGas_UnknownTypeFallsBackToMinGas(t *testing.T) {
	unknownTypes := []string{
		"/cosmos.unknown.v1.MsgDoSomething",
		"/zerone.future.v1.MsgNewFeature",
		"",
		"not-a-valid-type-url",
	}

	for _, msgType := range unknownTypes {
		got := lookupMsgGas(msgType)
		if got != MinGasLimit {
			t.Errorf("lookupMsgGas(%q) = %d, want MinGasLimit %d", msgType, got, MinGasLimit)
		}
	}
}

// ---------- Message Type Classification Tests ----------

func TestIsTransferMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/ibc.applications.transfer.v1.MsgTransfer",
	}
	for _, tc := range trueCases {
		if !isTransferMsg(tc) {
			t.Errorf("isTransferMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/cosmos.gov.v1.MsgVote",
		"",
	}
	for _, tc := range falseCases {
		if isTransferMsg(tc) {
			t.Errorf("isTransferMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsStakeMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/zerone.staking.v1.MsgRegisterValidator",
	}
	for _, tc := range trueCases {
		if !isStakeMsg(tc) {
			t.Errorf("isStakeMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"",
	}
	for _, tc := range falseCases {
		if isStakeMsg(tc) {
			t.Errorf("isStakeMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsClaimMsg(t *testing.T) {
	trueCases := []string{
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/zerone.knowledge.v1.MsgSubmitCommitment",
		"/zerone.knowledge.v1.MsgSubmitReveal",
		"/zerone.knowledge.v1.MsgChallengeFact",
	}
	for _, tc := range trueCases {
		if !isClaimMsg(tc) {
			t.Errorf("isClaimMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.gov.v1.MsgVote",
		"",
	}
	for _, tc := range falseCases {
		if isClaimMsg(tc) {
			t.Errorf("isClaimMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsVoteMsg(t *testing.T) {
	trueCases := []string{
		"/cosmos.gov.v1.MsgVote",
		"/zerone.gov.v1.MsgCastVote",
		"/zerone.emergency.v1.MsgVoteHalt",
		"/zerone.emergency.v1.MsgVoteRevert",
		"/zerone.emergency.v1.MsgVoteResume",
	}
	for _, tc := range trueCases {
		if !isVoteMsg(tc) {
			t.Errorf("isVoteMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"",
	}
	for _, tc := range falseCases {
		if isVoteMsg(tc) {
			t.Errorf("isVoteMsg(%q) = true, want false", tc)
		}
	}
}


func TestIsClaimSubmissionMsg(t *testing.T) {
	trueCases := []string{
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/zerone.knowledge.v1.MsgSubmitCommitment",
		"/zerone.knowledge.v1.MsgSubmitReveal",
	}
	for _, tc := range trueCases {
		if !isClaimSubmissionMsg(tc) {
			t.Errorf("isClaimSubmissionMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/zerone.knowledge.v1.MsgChallengeFact",
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.gov.v1.MsgVote",
		"",
	}
	for _, tc := range falseCases {
		if isClaimSubmissionMsg(tc) {
			t.Errorf("isClaimSubmissionMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsChallengeMsg(t *testing.T) {
	if !isChallengeMsg("/zerone.knowledge.v1.MsgChallengeFact") {
		t.Error("isChallengeMsg(MsgChallengeFact) = false, want true")
	}

	falseCases := []string{
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/zerone.knowledge.v1.MsgSubmitCommitment",
		"/cosmos.bank.v1beta1.MsgSend",
		"",
	}
	for _, tc := range falseCases {
		if isChallengeMsg(tc) {
			t.Errorf("isChallengeMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsClaimMsg_BackwardCompatible(t *testing.T) {
	// isClaimMsg must still return true for all 4 original messages
	allClaims := []string{
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/zerone.knowledge.v1.MsgSubmitCommitment",
		"/zerone.knowledge.v1.MsgSubmitReveal",
		"/zerone.knowledge.v1.MsgChallengeFact",
	}
	for _, tc := range allClaims {
		if !isClaimMsg(tc) {
			t.Errorf("isClaimMsg(%q) = false, want true (backward compat)", tc)
		}
	}
}

func TestIsAuthManagementMsg(t *testing.T) {
	trueCases := []string{
		"/zerone.auth.v1.MsgRegisterAccount",
		"/zerone.auth.v1.MsgRotateKey",
		"/zerone.auth.v1.MsgFreezeAccount",
		"/zerone.auth.v1.MsgUnfreezeAccount",
	}
	for _, tc := range trueCases {
		if !isAuthManagementMsg(tc) {
			t.Errorf("isAuthManagementMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"",
	}
	for _, tc := range falseCases {
		if isAuthManagementMsg(tc) {
			t.Errorf("isAuthManagementMsg(%q) = true, want false", tc)
		}
	}
}

func TestIsZeroneSpecificMsg(t *testing.T) {
	trueCases := []string{
		"/zerone.knowledge.v1.MsgSubmitClaim",
		"/zerone.knowledge.v1.MsgChallengeFact",
	}
	for _, tc := range trueCases {
		if !isZeroneSpecificMsg(tc) {
			t.Errorf("isZeroneSpecificMsg(%q) = false, want true", tc)
		}
	}

	falseCases := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.gov.v1.MsgVote",
		"/zerone.auth.v1.MsgRegisterAccount",
		"",
	}
	for _, tc := range falseCases {
		if isZeroneSpecificMsg(tc) {
			t.Errorf("isZeroneSpecificMsg(%q) = true, want false", tc)
		}
	}
}

// ---------- Gas Constants Validation ----------

func TestGasConstantsInvariant(t *testing.T) {
	// MinGasLimit < TxGasLimit < BlockGasLimit
	if MinGasLimit >= TxGasLimit {
		t.Errorf("MinGasLimit (%d) should be < TxGasLimit (%d)", MinGasLimit, TxGasLimit)
	}
	if TxGasLimit >= BlockGasLimit {
		t.Errorf("TxGasLimit (%d) should be < BlockGasLimit (%d)", TxGasLimit, BlockGasLimit)
	}
}

func TestAllGasCostsBelowTxLimit(t *testing.T) {
	for txType, gas := range TransactionGasCosts {
		if gas > TxGasLimit {
			t.Errorf("TransactionGasCosts[%q] = %d exceeds TxGasLimit %d", txType, gas, TxGasLimit)
		}
	}
}

func TestFeeRoutingBPSSumTo1000000(t *testing.T) {
	if ResearchContributionBPS+ValidatorFeeBPS != 1000000 {
		t.Errorf("ResearchContributionBPS (%d) + ValidatorFeeBPS (%d) = %d, want 1000000",
			ResearchContributionBPS, ValidatorFeeBPS, ResearchContributionBPS+ValidatorFeeBPS)
	}
}

// ---------- EstimateTransactionGas Tests ----------

func TestEstimateTransactionGas_KnownType(t *testing.T) {
	got := EstimateTransactionGas("transfer")
	if got != 21_000 {
		t.Errorf("EstimateTransactionGas(transfer) = %d, want 21000", got)
	}
}

func TestEstimateTransactionGas_UnknownType(t *testing.T) {
	got := EstimateTransactionGas("nonexistent_type")
	if got != MinGasLimit {
		t.Errorf("EstimateTransactionGas(nonexistent_type) = %d, want %d", got, MinGasLimit)
	}
}

func TestIsSystemTransaction(t *testing.T) {
	sysTxs := []string{
		"verification_reward", "slash_validator", "epoch_transition",
		"emergency_halt", "emergency_revert", "emergency_resume", "block_reward",
	}
	for _, tx := range sysTxs {
		if !IsSystemTransaction(tx) {
			t.Errorf("IsSystemTransaction(%q) = false, want true", tx)
		}
	}

	nonSysTxs := []string{"transfer", "stake", "claim_submit", ""}
	for _, tx := range nonSysTxs {
		if IsSystemTransaction(tx) {
			t.Errorf("IsSystemTransaction(%q) = true, want false", tx)
		}
	}
}

// ---------- GetMinimumFee Tests ----------

// mockMsg implements sdk.Msg for testing gas calculations.
type mockMsg struct {
	typeURL string
}

func (m mockMsg) ProtoMessage()        {}
func (m mockMsg) Reset()               {}
func (m mockMsg) String() string       { return m.typeURL }
func (m mockMsg) ValidateBasic() error { return nil }

func (m mockMsg) XXX_MessageName() string { return m.typeURL[1:] } // strip leading /

func TestGetMinimumFee_SingleMessage(t *testing.T) {
	// MsgSend -> transfer -> 21,000 gas -> clamped to MinGasLimit 22,222
	msgs := []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}
	fee := GetMinimumFee(msgs)

	expectedGas := uint64(22_222) // MinGasLimit since 21,000 < 22,222
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for MsgSend = %s, want %s uzrn", fee, expectedFee)
	}
}

func TestGetMinimumFee_MultipleMessages(t *testing.T) {
	// 3 transfers: 21,000 * 3 = 63,000 gas
	msgs := []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	}
	fee := GetMinimumFee(msgs)

	expectedGas := uint64(63_000) // 21,000 * 3 > MinGasLimit
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for 3x MsgSend = %s, want %s uzrn", fee, expectedFee)
	}
}

func TestGetMinimumFee_ExpensiveMessage(t *testing.T) {
	// register_validator = 100,000 gas
	msgs := []sdk.Msg{mockMsg{typeURL: "/zerone.staking.v1.MsgRegisterValidator"}}
	fee := GetMinimumFee(msgs)

	expectedGas := TransactionGasCosts["register_validator"]
	expectedFee := math.NewIntFromUint64(expectedGas * MinGasPrice)

	if !fee.AmountOf(BondDenom).Equal(expectedFee) {
		t.Errorf("GetMinimumFee for MsgRegisterValidator = %s, want %s uzrn", fee, expectedFee)
	}
}

// ---------- Proto URL Coverage Tests ----------

func TestMsgTypeURLToGas_AllEntriesReferenceValidCosts(t *testing.T) {
	for url, gas := range msgTypeURLToGas {
		if gas == 0 {
			t.Errorf("msgTypeURLToGas[%q] = 0, all mapped messages should have non-zero gas", url)
		}
	}
}


// ---------- ZRNGasDecorator Tests ----------

// mockFeeTx implements sdk.FeeTx for testing.
type mockFeeTx struct {
	sdk.Tx
	gas     uint64
	fee     sdk.Coins
	msgs    []sdk.Msg
	granter []byte // non-nil simulates a feegranted tx (granter pays the declared fee)
}

func (m mockFeeTx) GetGas() uint64     { return m.gas }
func (m mockFeeTx) GetFee() sdk.Coins  { return m.fee }
func (m mockFeeTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockFeeTx) FeeGranter() []byte { return m.granter }
func (m mockFeeTx) FeePayer() []byte   { return nil }

// passThroughHandler is a no-op next handler for testing decorators.
func passThroughHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

func TestZRNGasDecorator_SimulateSkips(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}
	tx := mockFeeTx{gas: 0} // Would fail if not simulating

	_, err := decorator.AnteHandle(ctx, tx, true, passThroughHandler)
	if err != nil {
		t.Errorf("simulate mode should skip validation, got error: %v", err)
	}
}

func TestZRNGasDecorator_ExceedsBlockGasLimit(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}.WithBlockHeight(1)
	tx := mockFeeTx{
		gas:  BlockGasLimit + 1,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for gas exceeding block limit")
	}
}

func TestZRNGasDecorator_ExceedsTxGasLimit(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}.WithBlockHeight(1)
	tx := mockFeeTx{
		gas:  TxGasLimit + 1,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for gas exceeding tx limit")
	}
}

func TestZRNGasDecorator_InsufficientGas(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}.WithBlockHeight(1)
	// register_validator requires 100,000 gas, provide only 1000
	tx := mockFeeTx{
		gas:  1000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(999999999))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/zerone.staking.v1.MsgRegisterValidator"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for insufficient gas")
	}
}

func TestZRNGasDecorator_SufficientGas(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}.WithBlockHeight(1)
	// transfer = 21,000 but MinGasLimit = 22,222; provide 30,000
	tx := mockFeeTx{
		gas:  30_000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(30_000))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Errorf("expected no error for sufficient gas, got: %v", err)
	}
}

func TestZRNGasDecorator_InsufficientFee(t *testing.T) {
	decorator := NewZRNGasDecorator()
	ctx := sdk.Context{}.WithBlockHeight(1)
	// Gas = 100,000, MinGasPrice = 1, so min fee = 100,000 uzrn. Provide only 1.
	tx := mockFeeTx{
		gas:  100_000,
		fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(1))),
		msgs: []sdk.Msg{mockMsg{typeURL: "/zerone.knowledge.v1.MsgSubmitClaim"}},
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Error("expected error for insufficient fee")
	}
}

// TestZRNGasDecorator_MinFeeEnforcement verifies the zero-fee consensus
// bypass is closed (design doc 2026-07-07 §10): the minimum-fee check
// applies to ALL txs at height >= 1, including zero-fee ones. Feegranted
// txs pass because the granter pays a NON-zero declared fee. The sole
// exemption is height 0 (InitChain gentx delivery).
func TestZRNGasDecorator_MinFeeEnforcement(t *testing.T) {
	sendMsg := []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}
	claimMsg := []sdk.Msg{mockMsg{typeURL: "/zerone.knowledge.v1.MsgSubmitClaim"}}
	granter := []byte("granter_address_____")

	tests := []struct {
		name    string
		height  int64
		tx      mockFeeTx
		wantErr bool
	}{
		{
			name:    "zero-fee bank send rejected at height 1",
			height:  1,
			tx:      mockFeeTx{gas: 30_000, fee: sdk.Coins{}, msgs: sendMsg},
			wantErr: true,
		},
		{
			name:   "properly-fee'd bank send passes",
			height: 1,
			tx: mockFeeTx{
				gas:  30_000,
				fee:  sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(30_000))),
				msgs: sendMsg,
			},
			wantErr: false,
		},
		{
			name:   "feegranted tx with adequate fee passes (granter pays)",
			height: 1,
			tx: mockFeeTx{
				gas:     30_000,
				fee:     sdk.NewCoins(sdk.NewCoin(BondDenom, math.NewInt(30_000))),
				msgs:    sendMsg,
				granter: granter,
			},
			wantErr: false,
		},
		{
			name:   "feegranted tx cannot smuggle a zero fee",
			height: 1,
			tx: mockFeeTx{
				gas:     30_000,
				fee:     sdk.Coins{},
				msgs:    sendMsg,
				granter: granter,
			},
			wantErr: true,
		},
		{
			name:    "BootstrapGasFreeTypes msg NOT fee-exempt at height 1 (window = 0)",
			height:  1,
			tx:      mockFeeTx{gas: 100_000, fee: sdk.Coins{}, msgs: claimMsg},
			wantErr: true,
		},
		{
			name:    "zero-fee gentx allowed at height 0 (InitChain genesis delivery)",
			height:  0,
			tx:      mockFeeTx{gas: 200_000, fee: sdk.Coins{}, msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.staking.v1beta1.MsgCreateValidator"}}},
			wantErr: false,
		},
	}

	decorator := NewZRNGasDecorator()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}.WithBlockHeight(tc.height)
			_, err := decorator.AnteHandle(ctx, tc.tx, false, passThroughHandler)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected ErrInsufficientFee, got nil")
				}
				if !sdkerrors.ErrInsufficientFee.Is(err) {
					t.Fatalf("expected ErrInsufficientFee, got: %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestBootstrapWindowRetired pins the mainnet-genesis decision: the
// 14-day gas-free window is zeroed in code (design doc 2026-07-07,
// "NO zero-fee bootstrap epoch"). If a future upgrade deliberately
// re-activates a window, update this test alongside the gov decision.
func TestBootstrapWindowRetired(t *testing.T) {
	if BootstrapEndBlock != 0 {
		t.Fatalf("BootstrapEndBlock = %d, mainnet genesis requires 0", BootstrapEndBlock)
	}
}

