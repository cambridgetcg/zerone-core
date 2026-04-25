package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ParseBounty returns the bounty as a positive big.Int, or an error
// if the input is unparseable or non-positive.
func ParseBounty(s string) (*big.Int, error) {
	if s == "" {
		return nil, fmt.Errorf("bounty cannot be empty")
	}
	n := new(big.Int)
	_, ok := n.SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("bounty %q is not a base-10 integer", s)
	}
	if n.Sign() <= 0 {
		return nil, fmt.Errorf("bounty must be > 0")
	}
	return n, nil
}

// ─── Msg validation ────────────────────────────────────────────────

func (msg *MsgSubmitInquiry) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Asker); err != nil {
		return fmt.Errorf("invalid asker: %w", err)
	}
	if msg.Question == "" {
		return ErrEmptyQuestion
	}
	if msg.Domain == "" {
		return ErrInvalidDomain
	}
	if _, err := ParseBounty(msg.Bounty); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBounty, err)
	}
	return nil
}

func (msg *MsgSubmitInquiry) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Asker)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitAnswer) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Answerer); err != nil {
		return fmt.Errorf("invalid answerer: %w", err)
	}
	if msg.InquiryId == "" {
		return fmt.Errorf("inquiry_id required")
	}
	if msg.ClaimId == "" {
		return fmt.Errorf("claim_id required")
	}
	return nil
}

func (msg *MsgSubmitAnswer) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Answerer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgResolveInquiry) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Caller); err != nil {
		return fmt.Errorf("invalid caller: %w", err)
	}
	if msg.InquiryId == "" {
		return fmt.Errorf("inquiry_id required")
	}
	return nil
}

func (msg *MsgResolveInquiry) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Caller)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelInquiry) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Asker); err != nil {
		return fmt.Errorf("invalid asker: %w", err)
	}
	if msg.InquiryId == "" {
		return fmt.Errorf("inquiry_id required")
	}
	return nil
}

func (msg *MsgCancelInquiry) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Asker)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if msg.Params != nil {
		if err := msg.Params.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}
