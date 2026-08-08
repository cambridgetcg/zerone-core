// Package staking currently implements Zerone's legacy custom Proof-of-Truth
// validator, delegation, tier, unbonding, and application-slash state.
//
// It is separate from Cosmos SDK x/staking. It does not create CometBFT
// validators, emit validator updates, own consensus power, or replace SDK
// distribution, evidence, and slashing. Callers must not describe a custom
// validator or custom bond as consensus staking.
//
// The accepted target architecture in docs/AUTHORITATIVE-STATE.md makes SDK
// x/staking the sole long-lived stake/delegation authority. This package's
// custodial state is to be frozen, reconciled into explicit legacy claims, and
// retired; non-monetary history moves to a verifier-profile module. Until that
// named migration is implemented and activated, this package remains current
// consensus-committed application state and its value-changing paths must be
// treated as legacy custody, not as the accepted destination.
//
// Truth-seeking position: bonded wealth secures block consensus. It does not
// buy policy voice or truth-judgement weight. Truth faults may affect
// qualification, profile reputation, and an explicitly posted work bond, but
// must not slash passive SDK delegators.
package staking
