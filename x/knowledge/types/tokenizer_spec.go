package types

// TokenizerSpecV1 is the bootstrap tokenizer contract. Seeded at genesis;
// governance-amendable via a version bump. Every training pipeline pins to
// a specific version so reproducibility is preserved across amendments.
//
// The structural tokens are chosen to be short, LLM-friendly, and
// unambiguous under common tokenisation schemes (BPE / SentencePiece).
// Angle-bracket syntax so they survive as single tokens rather than being
// split on whitespace.
func TokenizerSpecV1() *TokenizerSpec {
	return &TokenizerSpec{
		Version:                      1,
		RatifiedAtBlock:              0,
		MethodTokenPrefix:            "<method:",
		InferenceTokenPrefix:         "<inference:",
		RelationTokenPrefix:          "<relation:",
		FactStatusTokenPrefix:        "<status:",
		TierTokenPrefix:              "<tier:",
		FactBeginToken:               "<fact>",
		FactEndToken:                 "</fact>",
		ReasoningBeginToken:          "<reasoning>",
		ReasoningEndToken:            "</reasoning>",
		SupportBeginToken:            "<support>",
		SupportEndToken:              "</support>",
		DisproofMarkerToken:          "<disproved/>",
		CanonicalSerialisationVersion: 1,
	}
}
