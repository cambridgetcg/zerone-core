package types_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func claimRelations(n int) []*types.ClaimRelation {
	rels := make([]*types.ClaimRelation, n)
	for i := range rels {
		rels[i] = &types.ClaimRelation{
			TargetFactId: fmt.Sprintf("fact-%d", i),
			Relation:     types.RelationType_RELATION_TYPE_SUPPORTS,
		}
	}
	return rels
}

func TestMsgSubmitClaim_ValidateBasic_RelationsAtCap(t *testing.T) {
	msg := &types.MsgSubmitClaim{
		Submitter:   "zrn1submitter",
		FactContent: "water boils at 100C at sea level",
		Relations:   claimRelations(types.MaxRelationsPerClaim),
	}
	require.NoError(t, msg.ValidateBasic())
}

func TestMsgSubmitClaim_ValidateBasic_RelationsOverCap(t *testing.T) {
	msg := &types.MsgSubmitClaim{
		Submitter:   "zrn1submitter",
		FactContent: "water boils at 100C at sea level",
		Relations:   claimRelations(types.MaxRelationsPerClaim + 1),
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.True(t, errors.IsOf(err, types.ErrInvalidClaim))
}

func TestMsgSubmitClaim_ValidateBasic_NoRelations(t *testing.T) {
	msg := &types.MsgSubmitClaim{
		Submitter:   "zrn1submitter",
		FactContent: "water boils at 100C at sea level",
	}
	require.NoError(t, msg.ValidateBasic())
}

// MsgSubmitClaim is the only knowledge Msg with a repeated ClaimRelation
// field (verified against tx.pb.go at feat/karma-alpha base 2e37c4c), so the
// cap has exactly one enforcement site. This test pins that assumption: if a
// future Msg grows a Relations field, its ValidateBasic must enforce
// MaxRelationsPerClaim too.
func TestMaxRelationsPerClaim_Value(t *testing.T) {
	require.Equal(t, 16, types.MaxRelationsPerClaim,
		"K-alpha DoR A-3 fixes the cap at 16; K-beta paramifies it")
}
