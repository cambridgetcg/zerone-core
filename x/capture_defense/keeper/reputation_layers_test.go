package keeper_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func TestReputationLayersThroughKnowledgeAdapter(t *testing.T) {
	// Pins captured from 62c5cbad560062f6bda613ebabbfdb6ecc291c6c with this fixture.
	// These cover deterministic genesis-export bytes and gas, not a raw-store/AppHash proof.
	for _, scope := range []struct {
		domain, stratum                    string
		rejectedSnapshot, approvedSnapshot string
	}{
		{"physics", "empirical",
			"e9d28898071412c7a29a7bb21408af003f49652445bc7edeb97af8686aa9f5c5 gas=42388",
			"5f80164dab40e0544c33d1637d89343f736a3585d29d236f2de1219beaf9c18a gas=42427",
		},
		{"physics", "",
			"a0c58370b341c89db3c6464d003c038f3bbd52f1717eb7ac3a0700c2915bb366 gas=35560",
			"d2cde3969a7b01a8036aec7498c6fc5a3fc4f03e42058acec995a54360ac9294 gas=35599",
		},
		{"", "empirical",
			"25a9f6d70e20f957512671e5e160d0da47ba23ae2b711ae9b0c497e9d57879e7 gas=35764",
			"eaea7b17ae1a345e9b88543dfd29fedd3cd87b42c48195288b53c6bfb26553a5 gas=35764",
		},
		{"", "",
			"08c909abb324f26b160d000cf35e579b1f2d310f3df872b10fbefe194c0c1600 gas=28936",
			"0eab327f3b4bf731b32bd47cc40fda63e256f1ab66ba320ca4cf5e54ef6f7ed5 gas=28936",
		},
	} {
		for _, approved := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/approved=%t", scope.domain, scope.stratum, approved), func(t *testing.T) {
				k, ctx := setupKeeper(t)
				validator := testAddr(1)
				k.SetGlobalReputation(ctx, &types.GlobalReputation{Validator: validator, Score: 500000, TotalVerifications: 7, LastUpdatedBlock: 4})
				k.SetStratumReputation(ctx, &types.StratumReputation{Validator: validator, Stratum: "empirical", Score: 999999, Verifications: 4, LastUpdatedBlock: 4})
				k.SetDomainReputation(ctx, &types.DomainReputation{Validator: validator, Domain: "physics", Score: 1001, Verifications: 9, LastUpdatedBlock: 4})

				keeper.NewKnowledgeCaptureDefenseAdapter(k).UpdateReputation(sdk.WrapSDKContext(ctx), validator, scope.domain, scope.stratum, approved)

				wantGlobal, wantStratum, wantDomain := uint64(490000), uint64(980000), uint64(1)
				if approved {
					wantGlobal, wantStratum, wantDomain = 505000, 1000000, 10990
				}
				global, found := k.GetGlobalReputation(ctx, validator)
				require.True(t, found)
				require.Equal(t, wantGlobal, global.Score)
				require.Equal(t, uint64(8), global.TotalVerifications)
				require.Equal(t, uint64(100), global.LastUpdatedBlock)
				stratum, found := k.GetStratumReputation(ctx, "empirical", validator)
				require.True(t, found)
				if scope.stratum == "" {
					require.Equal(t, uint64(999999), stratum.Score)
					require.Equal(t, uint64(4), stratum.Verifications)
					require.Equal(t, uint64(4), stratum.LastUpdatedBlock)
				} else {
					require.Equal(t, wantStratum, stratum.Score)
					require.Equal(t, uint64(5), stratum.Verifications)
					require.Equal(t, uint64(100), stratum.LastUpdatedBlock)
				}
				domain, found := k.GetDomainReputation(ctx, "physics", validator)
				require.True(t, found)
				if scope.domain == "" {
					require.Equal(t, uint64(1001), domain.Score)
					require.Equal(t, uint64(9), domain.Verifications)
					require.Equal(t, uint64(4), domain.LastUpdatedBlock)
				} else {
					require.Equal(t, wantDomain, domain.Score)
					require.Equal(t, uint64(10), domain.Verifications)
					require.Equal(t, uint64(100), domain.LastUpdatedBlock)
				}
				data, err := proto.MarshalOptions{Deterministic: true}.Marshal(k.ExportGenesis(ctx))
				require.NoError(t, err)
				wantSnapshot := scope.rejectedSnapshot
				if approved {
					wantSnapshot = scope.approvedSnapshot
				}
				require.Equal(t, wantSnapshot, fmt.Sprintf("%x gas=%d", sha256.Sum256(data), ctx.GasMeter().GasConsumed()))
			})
		}
	}
}
