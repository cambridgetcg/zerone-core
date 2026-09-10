package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func (k Keeper) validateDeclaredMethod(ctx context.Context, method string) error {
	if err := types.ValidateRecordText("method_id", method, types.MaxMethodIDBytes, false); err != nil {
		return err
	}
	if method == "" {
		return nil
	}
	// This special method belongs to authority-authored doctrine, not an ordinary
	// contributor's declaration. It exempts records from lifecycle processing.
	if method == types.DoctrineMethodId {
		return fmt.Errorf("doctrine authorship cannot be declared by an ordinary contribution")
	}
	if _, found := k.GetMethodology(ctx, method); !found {
		return fmt.Errorf("unknown methodology %q", method)
	}
	return nil
}

func (k Keeper) validateChallengeRecordInput(ctx context.Context, reason string, evidence []string, method string) error {
	if err := types.ValidateRecordText("challenge reason", reason, types.MaxReviewReasonBytes, true); err != nil {
		return err
	}
	if err := types.ValidateEvidenceReferences(evidence); err != nil {
		return err
	}
	return k.validateDeclaredMethod(ctx, method)
}

func (k Keeper) validateClaimRecordInput(ctx context.Context, msg *types.MsgSubmitClaim) error {
	if len(msg.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unsupported claim fields")
	}
	if msg.ClaimType == types.ClaimType_CLAIM_TYPE_CONJECTURE {
		return fmt.Errorf("use PostConjecture with an explicit falsification predicate")
	}
	if msg.ClaimType < types.ClaimType_CLAIM_TYPE_UNSPECIFIED || msg.ClaimType > types.ClaimType_CLAIM_TYPE_COMPUTATIONAL {
		return fmt.Errorf("unknown claim type")
	}
	if !types.ValidAxiomCategories[msg.Category] {
		return fmt.Errorf("unknown epistemic category %q", msg.Category)
	}
	if err := k.validateDeclaredMethod(ctx, msg.MethodId); err != nil {
		return err
	}
	if err := types.ValidateRecordText("claim reasoning", msg.ReasoningTrace, types.MaxClaimReasoningBytes, false); err != nil {
		return err
	}
	if err := types.ValidateRecordText("canonical form", msg.CanonicalForm, types.MaxClaimReasoningBytes, false); err != nil {
		return err
	}
	if len(msg.References) > types.MaxClaimReferences {
		return fmt.Errorf("too many claim references")
	}
	for _, ref := range msg.References {
		if err := types.ValidateRecordText("claim reference", ref, types.MaxEvidenceReferenceBytes, true); err != nil {
			return err
		}
		if _, found := k.GetFact(ctx, ref); !found {
			return fmt.Errorf("claim reference %q does not exist", ref)
		}
	}
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	for _, rel := range msg.Relations {
		if rel == nil || len(rel.ProtoReflect().GetUnknown()) != 0 {
			return fmt.Errorf("invalid claim relation")
		}
		if rel.Relation < types.RelationType_RELATION_TYPE_SUPPORTS || rel.Relation > types.RelationType_RELATION_TYPE_REFORMULATES {
			return fmt.Errorf("unknown relation type")
		}
		if rel.Inference < types.InferenceType_INFERENCE_TYPE_UNSPECIFIED || rel.Inference > types.InferenceType_INFERENCE_TYPE_CITATION {
			return fmt.Errorf("unknown inference type")
		}
		if rel.InferenceStrengthBps > types.BPS {
			return fmt.Errorf("inference strength exceeds BPS scale")
		}
		if err := types.ValidateRecordText("relation target", rel.TargetFactId, types.MaxEvidenceReferenceBytes, true); err != nil {
			return err
		}
		if err := k.validateDeclaredMethod(ctx, rel.MethodId); err != nil {
			return err
		}
	}
	if s := msg.Structure; s != nil {
		if len(s.ProtoReflect().GetUnknown()) != 0 {
			return fmt.Errorf("unsupported claim structure fields")
		}
		for _, field := range []struct {
			name, value string
			limit       int
		}{
			{"subject", s.Subject, 1024}, {"predicate", s.Predicate, 4096}, {"object", s.Object, 2048},
			{"scope", s.Scope, 2048}, {"temporal scope", s.TemporalScope, 256},
		} {
			if err := types.ValidateRecordText("claim structure "+field.name, field.value, field.limit, false); err != nil {
				return err
			}
		}
		if msg.CanonicalForm != "" && types.NormalizeCanonicalForm(msg.CanonicalForm) != types.BuildCanonicalForm(msg.ClaimType, s, msg.Domain) {
			return fmt.Errorf("canonical form does not match declared claim structure")
		}
	}
	return nil
}
