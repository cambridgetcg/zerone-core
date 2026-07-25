package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"sort"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// ComputeLinkHash returns the deterministic canonical sha256 of a
// SubstrateLink. Length-prefixed everywhere; sorted children for
// determinism. Same input → same hash (M2 re-derivability anchor).
func ComputeLinkHash(l *types.SubstrateLink) []byte {
	h := sha256.New()
	writeLen(h, []byte(l.AdapterId))

	if l.Source != nil {
		writeLen(h, []byte(l.Source.SourceId))
		writeLen(h, l.Source.ContentHash)
		writeUint64(h, l.Source.FetchedAtBlock)
	}

	cf := append([]*types.FactCitation{}, l.CitedFacts...)
	sort.Slice(cf, func(i, j int) bool { return cf[i].FactId < cf[j].FactId })
	for _, c := range cf {
		writeLen(h, []byte(c.FactId))
		writeUint32(h, uint32(c.CitationType))
	}

	pc := append([]*types.PendingClaim{}, l.PendingClaims...)
	sort.Slice(pc, func(i, j int) bool {
		return types.PendingClaimCanonicalHash(pc[i]) < types.PendingClaimCanonicalHash(pc[j])
	})
	for _, c := range pc {
		writeLen(h, []byte(c.ClaimContent))
		writeLen(h, []byte(c.Domain))
		writeLen(h, []byte(c.MethodologyId))
	}

	if l.RecursionWeight != nil {
		writeUint64(h, l.RecursionWeight.AxisSubstrate)
		writeUint64(h, l.RecursionWeight.AxisVerification)
		writeUint64(h, l.RecursionWeight.AxisClassification)
		writeUint64(h, l.RecursionWeight.AxisAttribution)
		writeUint64(h, l.RecursionWeight.AxisTooling)
		writeUint64(h, l.RecursionWeight.AxisInterface)
	}

	return h.Sum(nil)
}

func writeLen(h interface{ Write([]byte) (int, error) }, data []byte) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(data)))
	h.Write(buf[:])
	h.Write(data)
}

func writeUint32(h interface{ Write([]byte) (int, error) }, v uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	h.Write(buf[:])
}

func writeUint64(h interface{ Write([]byte) (int, error) }, v uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	h.Write(buf[:])
}

func (k Keeper) ValidateLink(ctx context.Context, l *types.SubstrateLink, p types.Params) error {
	if l == nil {
		return types.ErrAdapterNotFound
	}
	adapter, found := k.GetAdapter(ctx, l.AdapterId)
	if !found {
		return types.ErrAdapterNotFound
	}
	if adapter.Status != types.AdapterStatus_ADAPTER_STATUS_ACTIVE {
		return types.ErrAdapterNotActive
	}
	if uint32(len(l.PendingClaims)) > p.MaxPendingClaimsPerAttestation {
		return types.ErrTooManyPendingClaims
	}
	if l.RecursionWeight != nil {
		// PROTOCOL CEILING — unconditional. The six axis projections are
		// self-declared uint64s supplied by the party being paid, and
		// computeReward sums them straight into the mint. Applying the
		// ceiling only when the adapter happened to declare AxisBounds made
		// the whole bound opt-in, and mainnet's only registered adapter
		// (agenttool-invocation-v1) ships axis_bounds:null — so the sum was
		// unbounded on the one adapter that exists. A projection is a
		// fraction of one axis; MaxAxisProjectionBps is 1.0 on this module's
		// 10,000-BPS scale.
		w := l.RecursionWeight
		for _, v := range []uint64{
			w.AxisSubstrate, w.AxisVerification, w.AxisClassification,
			w.AxisAttribution, w.AxisTooling, w.AxisInterface,
		} {
			if v > types.MaxAxisProjectionBps {
				return types.ErrAxisOverflow.Wrapf(
					"axis projection %d exceeds protocol ceiling %d", v, types.MaxAxisProjectionBps)
			}
		}
		// Adapter bounds may only tighten the protocol ceiling, never widen it.
		if adapter.AxisBounds != nil {
			if w.AxisSubstrate > adapter.AxisBounds.AxisSubstrateMax ||
				w.AxisVerification > adapter.AxisBounds.AxisVerificationMax ||
				w.AxisClassification > adapter.AxisBounds.AxisClassificationMax ||
				w.AxisAttribution > adapter.AxisBounds.AxisAttributionMax ||
				w.AxisTooling > adapter.AxisBounds.AxisToolingMax ||
				w.AxisInterface > adapter.AxisBounds.AxisInterfaceMax {
				return types.ErrAxisOverflow
			}
		}
	}
	// Cited facts must exist in x/knowledge. Only called when knowledgeKeeper is non-nil.
	if k.knowledgeKeeper != nil {
		for _, c := range l.CitedFacts {
			if _, found := k.knowledgeKeeper.GetFact(ctx, c.FactId); !found {
				return types.ErrCitedFactNotFound
			}
		}
	}
	return nil
}
