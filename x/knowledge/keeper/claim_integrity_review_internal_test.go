package keeper

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// sameActor (plan Task 14): one key is one actor in every bech32 encoding;
// anything else matches only an identical non-empty string.
func TestSameActor(t *testing.T) {
	encode := func(hrp string, key []byte) string {
		s, err := bech32.ConvertAndEncode(hrp, key)
		require.NoError(t, err)
		return s
	}
	key := bytes.Repeat([]byte{7}, 20)
	other := bytes.Repeat([]byte{8}, 20)
	account := encode("zrn", key)
	valoper := encode("zrnvaloper", key)
	mixed := strings.ToUpper(account[:5]) + account[5:]

	for _, tc := range []struct {
		name string
		a, b string
		same bool
	}{
		{"identical account", account, account, true},
		{"operator encoding", account, valoper, true},
		{"operator encoding reversed", valoper, account, true},
		{"upper case", account, strings.ToUpper(account), true},
		{"other prefix", account, encode("cosmos", key), true},
		{"longer key with the same prefix bytes", account, encode("zrn", append(bytes.Clone(key), 0)), false},
		{"different key", account, encode("zrn", other), false},
		{"different key, operator encoding", account, encode("zrnvaloper", other), false},
		{"mixed case is not bech32", account, mixed, false},
		{"identical non-bech32 string", "zrn1testvalidator", "zrn1testvalidator", true},
		{"different non-bech32 strings", "zrn1validator1", "zrn1validator2", false},
		{"non-bech32 against bech32", "zrn1testvalidator", account, false},
		{"empty strings", "", "", false},
		{"empty against account", "", account, false},
		{"account against empty", account, "", false},
		{"empty data under two prefixes", encode("zrn", nil), encode("cosmos", nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.same, sameActor(tc.a, tc.b), "%q vs %q", tc.a, tc.b)
		})
	}
}

// disputedFactID (plan Task 14) names the one fact a claim disputes: its
// ProvisionalFactId, or else the target of a CHALLENGE_DEPOSIT claim's single
// CONTRADICTS relation. Only MsgSubmitContradiction creates such a claim, so
// the shapes below that no message creates pin the rule's edges.
func TestDisputedFactID(t *testing.T) {
	relation := func(kind types.RelationType, target string) *types.ClaimRelation {
		return &types.ClaimRelation{Relation: kind, TargetFactId: target}
	}
	contradicts := relation(types.RelationType_RELATION_TYPE_CONTRADICTS, "fact-f")
	deposit := &types.ClaimFundingTerms{Kind: types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT}
	fee := &types.ClaimFundingTerms{Kind: types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE}

	for _, tc := range []struct {
		name  string
		claim *types.Claim
		want  string
	}{
		{"challenge", &types.Claim{ProvisionalFactId: "fact-p", FundingTerms: deposit}, "fact-p"},
		{"challenge before fund settlement", &types.Claim{ProvisionalFactId: "fact-p"}, "fact-p"},
		{"provisional fact wins over a relation", &types.Claim{ProvisionalFactId: "fact-p", FundingTerms: deposit, Relations: []*types.ClaimRelation{contradicts}}, "fact-p"},
		{"contradiction", &types.Claim{FundingTerms: deposit, Relations: []*types.ClaimRelation{contradicts}}, "fact-f"},
		{"contradiction before fund settlement", &types.Claim{Relations: []*types.ClaimRelation{contradicts}}, ""},
		{"submitted claim", &types.Claim{FundingTerms: fee, Relations: []*types.ClaimRelation{contradicts}}, ""},
		{"deposit with another relation type", &types.Claim{FundingTerms: deposit, Relations: []*types.ClaimRelation{relation(types.RelationType_RELATION_TYPE_SUPPORTS, "fact-f")}}, ""},
		{"deposit with two relations", &types.Claim{FundingTerms: deposit, Relations: []*types.ClaimRelation{contradicts, relation(types.RelationType_RELATION_TYPE_CONTRADICTS, "fact-g")}}, ""},
		{"deposit with a nil relation", &types.Claim{FundingTerms: deposit, Relations: []*types.ClaimRelation{nil}}, ""},
		{"deposit without relations", &types.Claim{FundingTerms: deposit}, ""},
		{"plain claim", &types.Claim{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, disputedFactID(tc.claim))
		})
	}
}
