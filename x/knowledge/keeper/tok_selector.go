package keeper

import (
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ToK chain-side caps. Adjustable via params in a future wave.
const (
	ToKMaxDepthCap   uint32 = 32
	ToKMaxPathsCap   uint32 = 256
	ToKFrontierLimit uint32 = 1024
	ToKFrontierCap   uint32 = 8192
)

// ValidateToKSelector returns an error iff the selector is malformed.
// TC5 (extraction is open) requires that the only refusal classes are
// syntax error, snapshot-out-of-range, and rate-limit. This is the
// syntax-error gate.
func ValidateToKSelector(s *types.ToKSelector) error {
	if s == nil || s.Variant == nil {
		return fmt.Errorf("selector variant required (TC5: extraction is open — but selector must be well-formed)")
	}
	switch v := s.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		if v.RootedSubtree == nil || v.RootedSubtree.RootFactId == "" {
			return fmt.Errorf("rooted_subtree.root_fact_id required")
		}
	case *types.ToKSelector_AncestorCone:
		if v.AncestorCone == nil || v.AncestorCone.LeafFactId == "" {
			return fmt.Errorf("ancestor_cone.leaf_fact_id required")
		}
	case *types.ToKSelector_Frontier:
		if v.Frontier == nil || v.Frontier.Domain == "" {
			return fmt.Errorf("frontier.domain required")
		}
	default:
		return fmt.Errorf("selector variant not recognised by this chain version")
	}
	return nil
}

// ValidateAndCapToKSelector validates and applies chain-side caps.
// Returns the capped selector. Caller should pass the capped value
// downstream so caps are applied uniformly.
func ValidateAndCapToKSelector(s *types.ToKSelector) (*types.ToKSelector, error) {
	if err := ValidateToKSelector(s); err != nil {
		return nil, err
	}
	out := &types.ToKSelector{Variant: s.Variant}
	switch v := out.Variant.(type) {
	case *types.ToKSelector_RootedSubtree:
		if v.RootedSubtree.MaxDepth == 0 || v.RootedSubtree.MaxDepth > ToKMaxDepthCap {
			v.RootedSubtree.MaxDepth = ToKMaxDepthCap
		}
	case *types.ToKSelector_AncestorCone:
		if v.AncestorCone.MaxDepth == 0 || v.AncestorCone.MaxDepth > ToKMaxDepthCap {
			v.AncestorCone.MaxDepth = ToKMaxDepthCap
		}
		if v.AncestorCone.MaxPaths == 0 || v.AncestorCone.MaxPaths > ToKMaxPathsCap {
			v.AncestorCone.MaxPaths = ToKMaxPathsCap
		}
	case *types.ToKSelector_Frontier:
		if v.Frontier.Limit == 0 {
			v.Frontier.Limit = ToKFrontierLimit
		}
		if v.Frontier.Limit > ToKFrontierCap {
			v.Frontier.Limit = ToKFrontierCap
		}
	}
	return out, nil
}
