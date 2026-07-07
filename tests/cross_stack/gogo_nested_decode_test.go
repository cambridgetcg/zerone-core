package cross_stack_test

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	zeronegov "github.com/zerone-chain/zerone/x/gov/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// allNestedMsgFullNames is the complete set of protobuf full-names for every
// message that appears as a NESTED CONCRETE (non-Any, non-enum) field of a
// service Msg across every module. cosmos-sdk v0.50's tx decoder resolves each
// such field's type via gogoproto.MessageType (codec/unknownproto/
// unknown_fields.go), i.e. the *gogo* registry. protoc-gen-go types are absent
// from that registry unless a per-module gogo_register.go explicitly bridges
// them, so a tx carrying a Msg with any of these nested fields would otherwise
// fail to decode with "failed to retrieve the message of type ...".
//
// This list mirrors, one-for-one, the registrations in every
// x/*/types/gogo_register.go (explicit-map modules) plus the messages covered
// by substrate_bridge's package-wide bridge. Keep it in sync when a new nested
// message is added to any Msg.
var allNestedMsgFullNames = []string{
	// tokens (original reference fix)
	"zerone.tokens.v1.TokenFeatures",
	// liquiditypool
	"zerone.liquiditypool.v1.Params",
	// claiming_pot
	"zerone.claiming_pot.v1.Params",
	"zerone.claiming_pot.v1.EligibilityCriteria",
	"zerone.claiming_pot.v1.VestingSchedule",
	// substrate_bridge (module that first surfaced the bug)
	"zerone.substrate_bridge.v1.AdapterRegistration",
	"zerone.substrate_bridge.v1.SubstrateLink",
	// alignment
	"zerone.alignment.v1.Params",
	// auth
	"zerone.auth.v1.Params",
	// capture_challenge
	"zerone.capture_challenge.v1.Params",
	// capture_defense
	"zerone.capture_defense.v1.Params",
	// counterexamples
	"zerone.counterexamples.v1.Params",
	// creed
	"zerone.creed.v1.Params",
	"zerone.creed.v1.PinnedCreed",
	"zerone.creed.v1.CreedCouncilMember",
	"zerone.creed.v1.CommitmentEntry",
	// emergency
	"zerone.emergency.v1.Params",
	// gov
	"zerone.gov.v1.Params",
	"zerone.gov.v1.ParamChange",
	"zerone.gov.v1.ResearchFundVoters",
	"zerone.gov.v1.CategoryConfig",
	// home
	"zerone.home.v1.Params",
	"zerone.home.v1.HomeGuardian",
	"zerone.home.v1.DeadmanConfig",
	// ibcratelimit
	"zerone.ibcratelimit.v1.Params",
	// knowledge
	"zerone.knowledge.v1.Params",
	"zerone.knowledge.v1.TokenizerSpec",
	"zerone.knowledge.v1.TraceSchema",
	"zerone.knowledge.v1.ClaimStructure",
	"zerone.knowledge.v1.ClaimRelation",
	"zerone.knowledge.v1.CorpusSelector",
	"zerone.knowledge.v1.DemandReport",
	// ontology
	"zerone.ontology.v1.Params",
	"zerone.ontology.v1.LogicZoneProperties",
	// qualification
	"zerone.qualification.v1.Params",
	// staking
	"zerone.staking.v1.Params",
	"zerone.staking.v1.TierConfig",
	// vesting_rewards (+ its transitive common types)
	"zerone.vesting_rewards.v1.Params",
	"zerone.vesting_rewards.v1.CategoryRewardConfig",
	"zerone.common.v1.RevenueSplit",
	"zerone.common.v1.ProtocolSubSplit",
}

// TestAllNestedMsgTypesGogoResolvable asserts that every nested concrete
// message carried by a service Msg across ALL modules is resolvable through the
// gogo registry — the exact lookup the tx decoder performs. A nil result here
// is precisely the condition that produced "failed to retrieve the message of
// type" at decode time.
func TestAllNestedMsgTypesGogoResolvable(t *testing.T) {
	for _, name := range allNestedMsgFullNames {
		require.NotNilf(t, gogoproto.MessageType(name),
			"nested message %q is not registered in the gogo registry; a tx "+
				"carrying a Msg with this field would fail to decode with "+
				"\"failed to retrieve the message of type %s\"", name, name)
	}
}

// TestNestedMsgTxRoundTripDecode drives the SAME production path that failed:
// it builds real txs carrying Msgs whose bodies embed a nested concrete
// message, encodes them with the app's TxConfig, then decodes them with the
// app's TxDecoder (which runs unknownproto.RejectUnknownFields). Before the
// gogo_register.go bridges existed, the decode returned "failed to retrieve the
// message of type ...". It must now succeed.
func TestNestedMsgTxRoundTripDecode(t *testing.T) {
	h := NewTestHarness(t)
	txConfig := h.App.TxConfig()
	authority := "zerone1updateparamsauthority00000000001"

	cases := []struct {
		name string
		msg  gogoproto.Message
	}{
		{
			// substrate_bridge: the module that first surfaced the bug.
			// MsgRegisterAdapter embeds AdapterRegistration, which itself
			// embeds AxisBounds and SlashGradient — exercising the transitive
			// nested closure.
			name: "substrate_bridge/MsgRegisterAdapter",
			msg: &substratebridgetypes.MsgRegisterAdapter{
				Authority: authority,
				Adapter: &substratebridgetypes.AdapterRegistration{
					AdapterId:   "wikipedia-en-v1",
					SourceType:  "wikipedia",
					Version:     "1.0.0",
					AxisBounds:  &substratebridgetypes.AxisBounds{},
					SlashGradient: &substratebridgetypes.SlashGradient{},
				},
			},
		},
		{
			// gov MsgUpdateParams: representative of the nested-Params carrier
			// pattern shared by every module's MsgUpdateParams.
			name: "gov/MsgUpdateParams",
			msg: &zeronegov.MsgUpdateParams{
				Authority: authority,
				Params:    &zeronegov.Params{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builder := txConfig.NewTxBuilder()
			require.NoError(t, builder.SetMsgs(tc.msg))

			bz, err := txConfig.TxEncoder()(builder.GetTx())
			require.NoError(t, err, "encode should succeed")

			_, err = txConfig.TxDecoder()(bz)
			if err != nil {
				require.NotContains(t, err.Error(), "failed to retrieve the message of type",
					"decode hit the unregistered-nested-message path for %s", tc.name)
			}
			require.NoError(t, err, "decode of %s must succeed", tc.name)
		})
	}
}
