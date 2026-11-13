package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"

	"github.com/cometbft/cometbft/crypto/ed25519"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
)

const oneOperatorLabel = "ONE_OPERATOR"

const oneOperatorWords = "One operator's validator key signed these headers. " +
	"This shows what that operator's chain recorded. " +
	"It is not independent confirmation and not a promise of finality."

type pinnedValidator struct {
	PublicKey string // standard base64 of a 32-byte Ed25519 key
	Power     int64
}

// A policy pins a chain's complete validator set. Changing it is a code
// change that needs review, never a runtime choice.
type policy struct {
	ChainID    string
	Validators []pinnedValidator
	Label      string
	Words      string
	Source     string
}

// compiledPolicies is the only trust table main uses. Both entries reuse pins
// this repository already holds. The x/bank and x/auth key layouts the state
// command reads are identical in cosmos-sdk v0.50.15 and v0.53.8, so the
// entries do not name an SDK line.
func compiledPolicies() map[string]policy {
	return map[string]policy{
		"zerone-dev-1": {
			ChainID:    "zerone-dev-1",
			Validators: []pinnedValidator{{PublicKey: "C7jBQ2qhJ6uORoLnFoshmLIeKOeOtDnDlCsWdpgsJBg=", Power: 1000000}},
			Label:      oneOperatorLabel,
			Words:      oneOperatorWords,
			Source:     "tools/research-checkpoint/verify.go (public key); dashboard/public/research/checkpoints/2026-09-14-demimetric-v1/chain/validators.json (voting power at height 183706)",
		},
		"zerone-1": {
			ChainID:    "zerone-1",
			Validators: []pinnedValidator{{PublicKey: "LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80=", Power: 11111}},
			Label:      oneOperatorLabel,
			Words:      oneOperatorWords,
			Source:     "deploy/legacy-observer/checkpoint/main.go (lines 32-34)",
		},
	}
}

func (p policy) validatorSet() (*cmttypes.ValidatorSet, error) {
	if p.ChainID == "" || len(p.Validators) == 0 || p.Label == "" || p.Words == "" || p.Source == "" {
		return nil, errors.New("incomplete trust policy")
	}
	var total int64
	seen := map[string]bool{}
	vals := make([]*cmttypes.Validator, 0, len(p.Validators))
	for _, v := range p.Validators {
		key, err := base64.StdEncoding.DecodeString(v.PublicKey)
		if err != nil || len(key) != ed25519.PubKeySize || seen[string(key)] || v.Power <= 0 || v.Power > cmttypes.MaxTotalVotingPower-total {
			return nil, errors.New("invalid pinned validator")
		}
		seen[string(key)] = true
		total += v.Power
		vals = append(vals, cmttypes.NewValidator(ed25519.PubKey(key), v.Power))
	}
	return cmttypes.NewValidatorSet(vals), nil
}

type attestedValidator struct {
	Address     string `json:"address"`
	PublicKey   string `json:"public_key_base64"`
	VotingPower string `json:"voting_power"`
}

type attestation struct {
	Label      string              `json:"label"`
	Words      string              `json:"words"`
	Validators []attestedValidator `json:"validators"`
	PinSource  string              `json:"pin_source"`
}

// attestation states whose signatures every proof in a report rests on.
func (p policy) attestation() attestation {
	a := attestation{Label: p.Label, Words: p.Words, PinSource: p.Source, Validators: []attestedValidator{}}
	if set, err := p.validatorSet(); err == nil {
		for _, v := range set.Validators {
			a.Validators = append(a.Validators, attestedValidator{Address: upperHex(v.Address), PublicKey: base64.StdEncoding.EncodeToString(v.PubKey.Bytes()), VotingPower: strconv.FormatInt(v.VotingPower, 10)})
		}
	}
	return a
}

// trustedHeader returns the header at height only if its validator response
// is exactly the pinned set, that set hashes to the header's validators hash,
// the header hashes to the commit's block ID, and the commit carries valid
// signatures from more than two thirds of the pinned power.
func trustedHeader(p policy, commitRaw, validatorsRaw []byte, height int64) (*cmttypes.SignedHeader, error) {
	pinned, err := p.validatorSet()
	if err != nil {
		return nil, err
	}
	var c rpctypes.ResultCommit
	var v rpctypes.ResultValidators
	if err := rpcResult(commitRaw, &c); err != nil {
		return nil, fmt.Errorf("commit response: %w", err)
	}
	if err := rpcResult(validatorsRaw, &v); err != nil {
		return nil, fmt.Errorf("validators response: %w", err)
	}
	sh := c.SignedHeader
	if err := sh.ValidateBasic(p.ChainID); err != nil {
		return nil, err
	}
	// ValidateBasic leaves the commit's own block ID unchecked, and VerifyCommit
	// panics while canonicalising a malformed one.
	if err := sh.Commit.BlockID.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("commit block ID: %w", err)
	}
	if sh.Height != height {
		return nil, fmt.Errorf("signed header is for height %d, not %d", sh.Height, height)
	}
	if v.BlockHeight != height {
		return nil, fmt.Errorf("validator response is for height %d, not %d", v.BlockHeight, height)
	}
	if len(v.Validators) != pinned.Size() || v.Count != pinned.Size() || v.Total != pinned.Size() {
		return nil, errors.New("validator response is not exactly the pinned set")
	}
	seen := map[string]bool{}
	for _, x := range v.Validators {
		if x == nil || x.PubKey == nil {
			return nil, errors.New("validator response has an empty entry")
		}
		_, want := pinned.GetByAddress(x.Address)
		if want == nil || seen[string(x.Address)] || !x.PubKey.Equals(want.PubKey) || !bytes.Equal(x.PubKey.Address(), x.Address) || x.VotingPower != want.VotingPower {
			return nil, fmt.Errorf("validator %X in the response is not in the pinned set with its pinned power", x.Address)
		}
		seen[string(x.Address)] = true
	}
	if !bytes.Equal(pinned.Hash(), sh.ValidatorsHash) {
		return nil, errors.New("header's validators hash is not the hash of the pinned set")
	}
	if !pinned.HasAddress(sh.ProposerAddress) {
		return nil, errors.New("header's proposer is not in the pinned set")
	}
	// No ValidateBasic checks a commit signature's timestamp, and VerifyCommit
	// panics on one that a protobuf timestamp cannot carry (years 1 to 9999).
	for i, cs := range sh.Commit.Signatures {
		if y := cs.Timestamp.UTC().Year(); y < 1 || y > 9999 {
			return nil, fmt.Errorf("commit signature %d timestamp is outside years 1 to 9999", i)
		}
	}
	if err := verifyCommit(pinned, p.ChainID, height, sh.Commit); err != nil {
		return nil, fmt.Errorf("commit signatures: %w", err)
	}
	return &sh, nil
}

// verifyCommit turns a panic inside CometBFT's commit verification into a
// refusal. Evidence is untrusted input: it may be refused, and must never
// crash the command.
func verifyCommit(set *cmttypes.ValidatorSet, chainID string, height int64, c *cmttypes.Commit) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("commit could not be verified: %v", r)
		}
	}()
	return set.VerifyCommit(chainID, c.BlockID, height, c)
}
