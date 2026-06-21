package types

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Expected keeper interfaces
//
// The contagion module depends on bank (to custody + disburse the reserve) and
// on the tokens module (to move ZO balances inside MsgSneeze, and to validate
// the ZO token). It exposes a ContagionHook interface that the tokens module
// can hold as a nil-safe optional pointer so non-ZO tokens pay zero contagion
// cost.
// ─────────────────────────────────────────────────────────────────────────────

// BankKeeper is the subset of the bank keeper the contagion module needs.
// Mirrors x/tokens/types/expected_keepers.go BankKeeper so the same bank
// keeper can be injected.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// TokensKeeper is the subset of the x/tokens keeper the contagion module
// needs to perform a ZO transfer inside MsgSneeze (the wrapper path) and to
// look up the ZO TokenDefinition for validation.
type TokensKeeper interface {
	// GetToken returns the ZRN-20 token definition by id, or nil if not found.
	GetToken(ctx sdk.Context, tokenId string) *TokenDefinitionShim
	// GetBalance returns an owner's balance for a token in base units.
	GetBalance(ctx sdk.Context, tokenId, ownerAddr string) *big.Int
	// SetBalance sets an owner's balance for a token in base units.
	SetBalance(ctx sdk.Context, tokenId, ownerAddr string, balance *big.Int)
	// Transfer performs a ZRN-20 transfer of `amount` base units from `from` to
	// `to`. Returns an error on insufficient balance or paused token. This is
	// the same move used by x/tokens MsgTransferToken — contagion reuses it
	// rather than reimplementing transfer.
	Transfer(ctx sdk.Context, tokenId, from, to string, amount *big.Int) error
}

// TokenDefinitionShim is a local mirror of x/tokens TokenDefinition so the
// contagion module does not import x/tokens (avoids an import cycle: tokens
// holds a ContagionHook pointer, contagion holds a TokensKeeper). The shim is
// populated by the app wiring layer from the real TokenDefinition.
type TokenDefinitionShim struct {
	Id          string
	Creator     string
	Name        string
	Symbol      string
	Decimals    uint32
	TotalSupply string
	MaxSupply   string
	Paused      bool
}

// ContagionHook is the interface the x/tokens module optionally holds. When
// non-nil, tokens.TransferToken calls OnTokenTransfer after a successful ZO
// move. The contagion keeper implements this. A nil hook means no contagion
// logic runs (e.g. for non-ZO tokens or before contagion is wired).
//
// This is the recommended integration path — see DESIGN.md Question 2.
type ContagionHook interface {
	// OnTokenTransfer is called by x/tokens after a successful transfer of
	// `amount` base units of `tokenId` from `from` to `to`. It MUST be
	// idempotent and MUST NOT revert the transfer on error (contagion is
	// best-effort: a depleted reserve or already-infected recipient simply
	// means no sneeze). Returns the SneezeEvent if a sneeze fired, else nil.
	OnTokenTransfer(ctx sdk.Context, tokenId, from, to string, amount *big.Int) *SneezeEvent
}
