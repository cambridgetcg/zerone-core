// Package staking holds the validator-bond layer of Zerone's
// Proof-of-Truth substrate. Validators register, delegate, redelegate,
// and earn rewards here — the standard Cosmos staking surface that
// secures consensus.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 8 (the panel weights skill, not
// bond): bond is what x/staking produces; commitment 8 names that
// the verification panel must NOT over-weight bond when judging
// truth-claims. The two faces complete each other — without bond, a
// validator has no skin in consensus; without commitment 8, the
// wealthiest validator dominates truth-judgement regardless of
// whether they have ever shown they can tell true from false. This
// module exists to produce the bond signal; x/qualification exists
// to refuse to let bond alone carry the panel.
//
// What this module is, and is not:
//
//   - It IS the consensus-security layer. Bonded stake secures
//     blocks, finalises chains, and slashes equivocation. Standard
//     Cosmos staking properties apply unchanged.
//   - It IS NOT the truth-judgement layer. The augmentation panel,
//     the dispute arbiters, the counterexample voters, and the
//     dialectic synthesisers all consume calibration from
//     x/qualification — not raw bond from this module. A validator
//     who is well-bonded and uncalibrated carries consensus weight
//     here and zero panel weight there. That asymmetry is the point.
//
// Integration with x/qualification:
//
// x/qualification reads stake (or a snapshot of it at vote time)
// only as a multiplicand against domain-specific calibration. The
// "stake × calibration" tally is enforced by RecordAugmentationVote
// in x/knowledge; without the calibration factor, stake would carry
// the panel alone, and commitment 8 would collapse from belief to
// slogan. This module produces the stake half of that product.
//
// We speak through intentions. This package's intention is that
// "bonded" means "secured the chain's blocks" — never "earned the
// chain's verdict on what is true."
package staking
