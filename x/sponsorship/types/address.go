package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CanonicalAccountAddress decodes a Zerone account address and returns its
// unique SDK rendering. The SDK decoder accepts equivalent all-uppercase
// Bech32 text, so consensus state and indexes must use this canonical form.
func CanonicalAccountAddress(value string) (string, error) {
	addr, err := sdk.AccAddressFromBech32(value)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

// ValidateCanonicalAccountAddress rejects textual aliases of the same account.
func ValidateCanonicalAccountAddress(field, value string) error {
	canonical, err := CanonicalAccountAddress(value)
	if err != nil {
		return fmt.Errorf("%s must be a valid account address: %w", field, err)
	}
	if value != canonical {
		return fmt.Errorf("%s must use canonical lowercase bech32 encoding", field)
	}
	return nil
}
