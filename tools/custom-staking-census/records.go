package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	customstakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

const (
	customStakingStore = customstakingtypes.StoreKey
	bankStore          = banktypes.StoreKey
	sdkStakingStore    = sdkstakingtypes.StoreKey
	claimDenom         = "uzrn"
)

var (
	bankBalanceKeyCodec = collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey)
	customModuleAddress = []byte(authtypes.NewModuleAddress(customstakingtypes.ModuleName))
)

type parsedValidator struct {
	key          string
	value        *customstakingtypes.Validator
	addressBytes []byte
	amounts      validatorAmounts
}

type validatorAmounts struct {
	self      *big.Int
	delegated *big.Int
	total     *big.Int
}

type parsedDelegation struct {
	key             string
	value           *customstakingtypes.Delegation
	delegatorBytes  []byte
	validatorBytes  []byte
	amount          *big.Int
	identityIsValid bool
}

type parsedUnbonding struct {
	key             string
	value           *customstakingtypes.UnbondingEntry
	delegatorBytes  []byte
	validatorBytes  []byte
	amount          *big.Int
	sequence        uint64
	identityIsValid bool
}

type parsedTierConfig struct {
	key   string
	value *customstakingtypes.TierConfig
}

type parsedDIDIndex struct {
	key      string
	did      string
	operator string
}

type parsedReverseIndex struct {
	key       string
	validator string
	delegator string
}

type parsedCooldown struct {
	key       string
	delegator string
	height    uint64
}

type parsedSDKValidator struct {
	key          string
	operator     string
	status       string
	jailed       bool
	tokens       string
	addressBytes []byte
}

type leafCommitment struct {
	key       []byte
	digest    [sha256.Size]byte
	inputSize uint64
}

func newLeafCommitment(key, value []byte) leafCommitment {
	h := sha256.New()
	writeHashField(h, []byte("zerone/custom-staking-census/leaf/v1"))
	writeHashField(h, key)
	writeHashField(h, value)
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))
	return leafCommitment{
		key:       bytes.Clone(key),
		digest:    digest,
		inputSize: uint64(len(key)) + uint64(len(value)),
	}
}

func writeHashField(w io.Writer, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = w.Write(size[:])
	_, _ = w.Write(value)
}

func displayKey(store string, key []byte) string {
	const maxDisplayedKeyBytes = 256
	if len(key) > maxDisplayedKeyBytes {
		digest := sha256.Sum256(key)
		return fmt.Sprintf(
			"%s/sha256:%s/bytes:%d/prefix:%s",
			store,
			hex.EncodeToString(digest[:]),
			len(key),
			hex.EncodeToString(key[:64]),
		)
	}
	return store + "/" + hex.EncodeToString(key)
}

func decodeStrictJSON(raw []byte, destination any) error {
	if len(raw) == 0 {
		return errors.New("empty JSON value")
	}
	if err := preflightJSON(raw); err != nil {
		return fmt.Errorf("preflight JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	canonical, err := json.Marshal(destination)
	if err != nil {
		return fmt.Errorf("marshal canonical JSON: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return errors.New("value is not the canonical encoding/json representation")
	}
	return nil
}

func decodeCanonicalAddress(value, expectedHRP string) ([]byte, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return nil, errors.New("address is empty or has surrounding whitespace")
	}
	hrp, decoded, err := bech32.DecodeAndConvert(value)
	if err != nil {
		return nil, err
	}
	if hrp != expectedHRP {
		return nil, fmt.Errorf("address HRP is %q, expected %q", hrp, expectedHRP)
	}
	if err := sdk.VerifyAddressFormat(decoded); err != nil {
		return nil, err
	}
	canonical, err := bech32.ConvertAndEncode(expectedHRP, decoded)
	if err != nil {
		return nil, err
	}
	if value != canonical {
		return nil, errors.New("address is not canonical lowercase bech32")
	}
	return decoded, nil
}

func parseCanonicalAmount(value string, positive bool) (*big.Int, error) {
	if value == "" {
		return nil, errors.New("amount is empty")
	}
	// SDK bank amounts are bounded to 256 bits. Reject oversized text before
	// parsing so a corrupt copied DB cannot turn the census into an unbounded
	// big.Int allocation.
	if len(value) > 78 {
		return nil, fmt.Errorf("amount exceeds the %d-bit SDK integer limit", sdkmath.MaxBitLen)
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok || amount.Sign() < 0 {
		return nil, errors.New("amount is not a non-negative base-10 integer")
	}
	if amount.String() != value {
		return nil, errors.New("amount is not canonical base-10")
	}
	if amount.BitLen() > sdkmath.MaxBitLen {
		return nil, fmt.Errorf("amount exceeds the %d-bit SDK integer limit", sdkmath.MaxBitLen)
	}
	if positive && amount.Sign() == 0 {
		return nil, errors.New("amount must be positive")
	}
	return amount, nil
}

func splitAddressPair(raw []byte) (string, string, error) {
	if !utf8.Valid(raw) {
		return "", "", errors.New("key suffix is not UTF-8")
	}
	if bytes.Count(raw, []byte{0}) != 1 {
		return "", "", errors.New("key suffix must contain exactly one NUL separator")
	}
	parts := bytes.SplitN(raw, []byte{0}, 2)
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return "", "", errors.New("address component is empty")
	}
	return string(parts[0]), string(parts[1]), nil
}

func parseBankBalanceKey(key []byte) (sdk.AccAddress, string, error) {
	prefix := banktypes.BalancesPrefix.Bytes()
	if !bytes.HasPrefix(key, prefix) {
		return nil, "", errors.New("not a bank balance key")
	}
	read, pair, err := bankBalanceKeyCodec.Decode(key[len(prefix):])
	if err != nil {
		return nil, "", err
	}
	if read != len(key)-len(prefix) {
		return nil, "", errors.New("bank balance key has trailing bytes")
	}
	address := pair.K1()
	if err := sdk.VerifyAddressFormat(address); err != nil {
		return nil, "", fmt.Errorf("invalid bank address bytes: %w", err)
	}
	denom := pair.K2()
	if err := sdk.ValidateDenom(denom); err != nil {
		return nil, "", fmt.Errorf("invalid bank denomination: %w", err)
	}
	return address, denom, nil
}

func parseBankBalanceValue(denom string, raw []byte) (*big.Int, error) {
	amount, intErr := sdk.IntValue.Decode(raw)
	if intErr == nil {
		canonical, err := sdk.IntValue.Encode(amount)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(raw, canonical) {
			return nil, errors.New("bank integer value is not canonical")
		}
		if !amount.IsPositive() {
			return nil, errors.New("bank balance must be positive")
		}
		return new(big.Int).Set(amount.BigInt()), nil
	}

	var coin sdk.Coin
	if err := coin.Unmarshal(raw); err != nil {
		return nil, fmt.Errorf("decode bank balance as integer or legacy coin: %w", intErr)
	}
	canonical, err := coin.Marshal()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(raw, canonical) {
		return nil, errors.New("legacy bank coin value is not canonical protobuf")
	}
	if err := coin.Validate(); err != nil {
		return nil, fmt.Errorf("invalid legacy bank coin: %w", err)
	}
	if coin.Denom != denom {
		return nil, fmt.Errorf("legacy coin denomination %q does not match key denomination %q", coin.Denom, denom)
	}
	if !coin.Amount.IsPositive() {
		return nil, errors.New("bank balance must be positive")
	}
	return new(big.Int).Set(coin.Amount.BigInt()), nil
}

func parseSDKValidator(key, raw []byte) (*parsedSDKValidator, error) {
	if len(key) < 3 || key[0] != sdkstakingtypes.ValidatorsKey[0] {
		return nil, errors.New("invalid SDK validator primary key")
	}
	addressLength := int(key[1])
	if addressLength == 0 || len(key) != addressLength+2 {
		return nil, errors.New("SDK validator key has invalid address length prefix")
	}
	keyAddress := key[2:]
	if err := sdk.VerifyAddressFormat(keyAddress); err != nil {
		return nil, fmt.Errorf("invalid SDK validator key address: %w", err)
	}
	if err := preflightSDKValidatorProto(raw); err != nil {
		return nil, fmt.Errorf("preflight SDK validator protobuf: %w", err)
	}

	var validator sdkstakingtypes.Validator
	if err := validator.Unmarshal(raw); err != nil {
		return nil, fmt.Errorf("decode SDK validator protobuf: %w", err)
	}
	canonical, err := validator.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal SDK validator protobuf: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return nil, errors.New("SDK validator value is not canonical protobuf")
	}
	payloadAddress, err := decodeCanonicalAddress(validator.OperatorAddress, "zrnvaloper")
	if err != nil {
		return nil, fmt.Errorf("invalid SDK operator address: %w", err)
	}
	if !bytes.Equal(keyAddress, payloadAddress) {
		return nil, errors.New("SDK validator key address does not match payload operator address")
	}
	if validator.Status < sdkstakingtypes.Unbonded || validator.Status > sdkstakingtypes.Bonded {
		return nil, fmt.Errorf("SDK validator has invalid bond status %d", validator.Status)
	}
	if validator.Tokens.IsNil() || validator.Tokens.IsNegative() {
		return nil, errors.New("SDK validator tokens are nil or negative")
	}
	if validator.DelegatorShares.IsNil() || validator.DelegatorShares.IsNegative() {
		return nil, errors.New("SDK validator shares are nil or negative")
	}
	return &parsedSDKValidator{
		key:          displayKey(sdkStakingStore, key),
		operator:     validator.OperatorAddress,
		status:       validator.Status.String(),
		jailed:       validator.Jailed,
		tokens:       validator.Tokens.String(),
		addressBytes: bytes.Clone(payloadAddress),
	}, nil
}

func parseUnbondingSequence(entry *customstakingtypes.UnbondingEntry) (uint64, error) {
	expectedPrefix := entry.DelegatorAddress + "_" + entry.ValidatorAddress + "_"
	if !strings.HasPrefix(entry.Id, expectedPrefix) {
		return 0, errors.New("unbonding ID does not begin with its delegator and validator")
	}
	remainder := strings.TrimPrefix(entry.Id, expectedPrefix)
	createdText, sequenceText, found := strings.Cut(remainder, "_")
	if !found || strings.Contains(sequenceText, "_") {
		return 0, errors.New("unbonding ID must end in created-height and sequence")
	}
	created, ok := new(big.Int).SetString(createdText, 10)
	if !ok || created.Sign() < 0 || created.BitLen() > 64 || created.String() != createdText || created.Uint64() != entry.CreatedAtHeight {
		return 0, errors.New("unbonding ID created-height is invalid or mismatched")
	}
	sequence, ok := new(big.Int).SetString(sequenceText, 10)
	if !ok || sequence.Sign() <= 0 || sequence.BitLen() > 64 || sequence.String() != sequenceText {
		return 0, errors.New("unbonding ID sequence is not a canonical positive uint64")
	}
	return sequence.Uint64(), nil
}
