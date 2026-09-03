package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

const (
	// ApplicationMempoolMinTxs rejects the SDK's MaxTx=0 sentinel because it
	// means unbounded storage rather than a disabled pool.
	ApplicationMempoolMinTxs = 1
	// ApplicationMempoolMaxTxs is the immutable process-local ceiling. Keep it
	// aligned with the smaller/equal production CometBFT transaction-count cap.
	ApplicationMempoolMaxTxs = 5_000
)

// messageSignerExtractionAdapter derives signer identity from the transaction
// messages and sequence from AuthInfo. Cosmos permits a signed transaction to
// omit SignerInfo.public_key once the account already stores that key; the SDK's
// default mempool extractor dereferences that optional field and can panic.
// Pairing these two canonical surfaces keeps CheckTx, proposal selection, and
// FinalizeBlock removal safe for that valid wire shape.
type messageSignerExtractionAdapter struct{}

func newMessageSignerExtractionAdapter() messageSignerExtractionAdapter {
	return messageSignerExtractionAdapter{}
}

func (messageSignerExtractionAdapter) GetSigners(tx sdk.Tx) ([]sdkmempool.SignerData, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, fmt.Errorf("tx of type %T does not implement SigVerifiableTx", tx)
	}
	signatures, err := sigTx.GetSignaturesV2()
	if err != nil {
		return nil, fmt.Errorf("read transaction signatures: %w", err)
	}
	signerBytes, err := sigTx.GetSigners()
	if err != nil {
		return nil, fmt.Errorf("derive transaction message signers: %w", err)
	}
	if len(signatures) != len(signerBytes) {
		return nil, fmt.Errorf(
			"signature count %d does not match message signer count %d",
			len(signatures),
			len(signerBytes),
		)
	}
	signers := make([]sdkmempool.SignerData, len(signatures))
	for i, signer := range signerBytes {
		if err := sdk.VerifyAddressFormat(signer); err != nil {
			return nil, fmt.Errorf("invalid signer %d: %w", i, err)
		}
		signers[i] = sdkmempool.NewSignerData(
			sdk.AccAddress(append([]byte(nil), signer...)),
			signatures[i].Sequence,
		)
	}
	return signers, nil
}

// NewApplicationMempool constructs Zerone's bounded application-side pool.
// PriorityNonce preserves each sender's nonce order and accepts the safe signer
// adapter above. The network-facing CometBFT pool remains responsible for
// gossip; this pool is a proposer-selection index, never consensus authority.
func NewApplicationMempool(maxTx int) sdkmempool.Mempool {
	if err := ValidateApplicationMempoolMaxTx(maxTx); err != nil {
		panic(err)
	}
	config := sdkmempool.DefaultPriorityNonceMempoolConfig()
	config.MaxTx = maxTx
	config.SignerExtractor = newMessageSignerExtractionAdapter()
	return sdkmempool.NewPriorityMempool(config)
}

// ValidateApplicationMempoolMaxTx rejects both SDK sentinel values and local
// overrides above Zerone's process-level ceiling. Production startup must not
// turn a bounded safety mechanism into a NoOp or unbounded pool by config.
func ValidateApplicationMempoolMaxTx(maxTx int) error {
	if maxTx < ApplicationMempoolMinTxs || maxTx > ApplicationMempoolMaxTxs {
		return fmt.Errorf(
			"application mempool max-txs must be between %d and %d, got %d",
			ApplicationMempoolMinTxs,
			ApplicationMempoolMaxTxs,
			maxTx,
		)
	}
	return nil
}
