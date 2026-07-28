package types

import "cosmossdk.io/errors"

// K-alpha sentinel errors. Kept in a separate file so the K-alpha slice does
// not contend with concurrent edits to errors.go; fold into errors.go when
// the karma work lands. Codes continue the errors.go sequence (last used: 81).
var (
	// ErrRoundSeatsFull rejects a tx-path commitment once a round already
	// holds CommitSeatHardCap commitments. This is the K-alpha backstop
	// bound, not the design §2.6 tight seat cap — see the CommitSeatHardCap
	// comment in validate.go for why the tight cap waits for C-2 seat bonds.
	ErrRoundSeatsFull = errors.Register(ModuleName, 82, "verification round commit seats are full")
)
