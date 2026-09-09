package keeper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/staking/types"
)

// These ceilings apply before activation and to every subsequent store write.
// Completed withdrawals retain their history. New admission stops at the key
// ceiling; completing an existing withdrawal requires no new key.
const (
	MaxAccountingStoreEntries = 50_000
	MaxAccountingRecordBytes  = 256 * 1024
	MaxAccountingKeyBytes     = 1024
)

func accountingError(format string, args ...any) error {
	return types.ErrAccountingSafety.Wrapf(format, args...)
}

// SDK cache iterators report these two sentinels on normal exhaustion. Every
// other terminal error remains a failed audit, including backend I/O errors.
func accountingIteratorError(iter interface {
	Valid() bool
	Error() error
}) error {
	err := iter.Error()
	if err != nil && !iter.Valid() && (err.Error() == "invalid cacheMergeIterator" || err.Error() == "invalid memIterator") {
		return nil
	}
	return err
}

func (k Keeper) closeAccountingIterator(ctx sdk.Context, iter storetypes.Iterator) {
	err := errors.Join(accountingIteratorError(iter), iter.Close())
	if k.AccountingSafetyEnabled(ctx) && err != nil {
		panic(accountingError("store iteration failed: %v", err))
	}
}

// AccountingSafetyEnabled reports the committed custody boundary. A malformed
// marker must never select historical, unsafe execution as a fallback.
func (k Keeper) AccountingSafetyEnabled(ctx sdk.Context) bool {
	bz := k.getStore(ctx).Get(types.AccountingSafetyKey)
	if bz == nil {
		return false
	}
	if !bytes.Equal(bz, []byte{1}) {
		panic(accountingError("invalid accounting marker"))
	}
	return true
}

// EnableAccountingSafety validates the existing claimant ledger without
// repairing, repricing, moving, or assigning any claim or bank balance.
func (k Keeper) EnableAccountingSafety(ctx sdk.Context) error {
	cache, write := ctx.CacheContext()
	if err := k.ValidateAccountingSafety(cache); err != nil {
		return err
	}
	k.getStore(cache).Set(types.AccountingSafetyKey, []byte{1})
	if err := k.accountingCapacity(cache); err != nil {
		return err
	}
	write()
	return nil
}

func canonicalAccountingAddress(address string) error {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil || addr.String() != address {
		return accountingError("noncanonical account address %q", address)
	}
	return nil
}

func accountingAmount(value string, positive bool) (*big.Int, error) {
	if len(value) == 0 || len(value) > 78 || (len(value) > 1 && value[0] == '0') {
		return nil, accountingError("noncanonical claim amount")
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return nil, accountingError("invalid claim amount")
		}
	}
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.BitLen() > 256 || (positive && n.Sign() == 0) {
		return nil, accountingError("invalid claim amount")
	}
	return n, nil
}

func validateAccountingParams(p *types.Params) error {
	if p == nil || len(p.TierConfigs) != 4 {
		return accountingError("four tier configurations required")
	}
	seen := map[types.ValidatorTier]bool{}
	for _, tc := range p.TierConfigs {
		if tc == nil || tc.Tier < types.TierApprentice || tc.Tier > types.TierGuardian || seen[tc.Tier] {
			return accountingError("invalid or duplicate tier configuration")
		}
		seen[tc.Tier] = true
	}
	if err := p.Validate(); err != nil {
		return accountingError("invalid params: %v", err)
	}
	return nil
}

// Walk the token stream first because encoding/json otherwise accepts duplicate
// object keys, which cannot establish one unambiguous historical claim.
func uniqueJSONValue(d *json.Decoder, typ reflect.Type, depth int, tokens *int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting ceiling exceeded")
	}
	*tokens++
	if *tokens > 32768 {
		return fmt.Errorf("JSON token ceiling exceeded")
	}
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		allowed := map[string]reflect.Type{}
		if typ == nil || typ.Kind() != reflect.Struct {
			return fmt.Errorf("unexpected object")
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if f.IsExported() && name != "" && name != "-" {
				allowed[name] = f.Type
			}
		}
		seen := map[string]bool{}
		for d.More() {
			*tokens++
			if *tokens > 32768 {
				return fmt.Errorf("JSON token ceiling exceeded")
			}
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			field, known := allowed[name]
			if !ok || seen[name] || !known {
				return fmt.Errorf("duplicate, unknown or aliased JSON key")
			}
			seen[name] = true
			if err := uniqueJSONValue(d, field, depth+1, tokens); err != nil {
				return err
			}
		}
	case '[':
		if typ == nil || (typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array) {
			return fmt.Errorf("unexpected array")
		}
		for d.More() {
			if err := uniqueJSONValue(d, typ.Elem(), depth+1, tokens); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	*tokens++
	if *tokens > 32768 {
		return fmt.Errorf("JSON token ceiling exceeded")
	}
	_, err = d.Token()
	return err
}

func decodeAccountingJSON(bz []byte, target any) error {
	if len(bz) == 0 || len(bz) > MaxAccountingRecordBytes || !utf8.Valid(bz) || bytes.Equal(bytes.TrimSpace(bz), []byte("null")) {
		return accountingError("invalid record size or null record")
	}
	d := json.NewDecoder(bytes.NewReader(bz))
	d.UseNumber()
	tokens := 0
	if err := uniqueJSONValue(d, reflect.TypeOf(target), 0, &tokens); err != nil {
		return accountingError("ambiguous JSON: %v", err)
	}
	if _, err := d.Token(); err != io.EOF {
		return accountingError("trailing JSON data")
	}
	d = json.NewDecoder(bytes.NewReader(bz))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return accountingError("invalid record: %v", err)
	}
	return nil
}

func validateAccountingValidator(v *types.Validator) error {
	if err := canonicalAccountingAddress(v.OperatorAddress); err != nil {
		return err
	}
	// The legacy consensus-key description is not an authenticated Comet key.
	// Bounds preserve future update headroom without inventing a key binding.
	if len(v.Moniker) > 70 || len(v.Did) > 128 || len(v.Website) > 140 || len(v.Details) > 2000 || v.CommissionBps > 10_000 {
		return accountingError("invalid validator metadata bounds")
	}
	if len(v.ConsensusPubkey) > 4096 {
		return accountingError("legacy consensus-key description too large")
	}
	self, err := accountingAmount(v.SelfDelegation, false)
	if err != nil {
		return err
	}
	delegated, err := accountingAmount(v.DelegatedStake, false)
	if err != nil {
		return err
	}
	total, err := accountingAmount(v.TotalStake, false)
	if err != nil {
		return err
	}
	if new(big.Int).Add(self, delegated).Cmp(total) != 0 {
		return accountingError("validator total disagrees with self plus delegated")
	}
	return nil
}

func validateAccountingDelegation(d *types.Delegation) (*big.Int, error) {
	if err := canonicalAccountingAddress(d.DelegatorAddress); err != nil {
		return nil, err
	}
	if err := canonicalAccountingAddress(d.ValidatorAddress); err != nil {
		return nil, err
	}
	return accountingAmount(d.Amount, true)
}

func validateAccountingUnbonding(u *types.UnbondingEntry) (*big.Int, uint64, error) {
	if err := canonicalAccountingAddress(u.DelegatorAddress); err != nil {
		return nil, 0, err
	}
	if err := canonicalAccountingAddress(u.ValidatorAddress); err != nil {
		return nil, 0, err
	}
	amount, err := accountingAmount(u.Amount, true)
	if err != nil {
		return nil, 0, err
	}
	if u.Status != "pending" && u.Status != "completed" {
		return nil, 0, accountingError("unknown unbonding status")
	}
	if u.CompletesAtHeight <= u.CreatedAtHeight || u.CompletesAtHeight > math.MaxInt64 {
		return nil, 0, accountingError("invalid unbonding heights")
	}
	prefix := u.DelegatorAddress + "_" + u.ValidatorAddress + "_" + strconv.FormatUint(u.CreatedAtHeight, 10) + "_"
	if !strings.HasPrefix(u.Id, prefix) {
		return nil, 0, accountingError("unbonding ID disagrees with claim")
	}
	seqText := strings.TrimPrefix(u.Id, prefix)
	seq, err := strconv.ParseUint(seqText, 10, 64)
	if err != nil || seq == 0 || strconv.FormatUint(seq, 10) != seqText {
		return nil, 0, accountingError("invalid unbonding sequence")
	}
	return amount, seq, nil
}

// ValidateAccountingSafety performs one bounded, strict primary-store scan.
// It does not use legacy iterators, which intentionally skip malformed records.
// Both directions of each identity and delegation index must be exact.
func (k Keeper) ValidateAccountingSafety(ctx sdk.Context) (err error) {
	store := k.getStore(ctx)
	iter := store.Iterator(nil, nil)
	defer func() { err = errors.Join(err, accountingIteratorError(iter), iter.Close()) }()
	validators := map[string]*types.Validator{}
	selfClaims, delegatedClaims := map[string]*big.Int{}, map[string]*big.Int{}
	expectedDIDs, actualDIDs := map[string]string{}, map[string]string{}
	expectedReverse, actualReverse := map[string]bool{}, map[string]bool{}
	tiers := map[types.ValidatorTier]*types.TierConfig{}
	unbondings := []*types.UnbondingEntry{}
	sequences := map[uint64]bool{}
	liabilities := new(big.Int)
	var params *types.Params
	var storedSeq, maxSeq uint64
	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
		key, value := iter.Key(), iter.Value()
		if count > MaxAccountingStoreEntries || len(key) == 0 || len(key) > MaxAccountingKeyBytes || len(value) > MaxAccountingRecordBytes {
			return accountingError("accounting resource ceiling exceeded")
		}
		if bytes.Equal(key, []byte("_iavl_init")) {
			if !bytes.Equal(value, []byte{1}) {
				return accountingError("invalid IAVL initialization sentinel")
			}
			continue
		}
		switch key[0] {
		case types.ValidatorKeyPrefix[0]:
			var v types.Validator
			if err := decodeAccountingJSON(value, &v); err != nil {
				return err
			}
			if err := validateAccountingValidator(&v); err != nil {
				return err
			}
			if !bytes.Equal(key, types.ValidatorKey(v.OperatorAddress)) {
				return accountingError("validator key disagrees with identity")
			}
			validators[v.OperatorAddress] = &v
			if v.Did != "" {
				if _, exists := expectedDIDs[v.Did]; exists {
					return accountingError("duplicate validator DID")
				}
				expectedDIDs[v.Did] = v.OperatorAddress
			}
		case types.DelegationKeyPrefix[0]:
			var d types.Delegation
			if err := decodeAccountingJSON(value, &d); err != nil {
				return err
			}
			amount, err := validateAccountingDelegation(&d)
			if err != nil {
				return err
			}
			if !bytes.Equal(key, types.DelegationKey(d.DelegatorAddress, d.ValidatorAddress)) {
				return accountingError("delegation key disagrees with claimant")
			}
			liabilities.Add(liabilities, amount)
			dest := delegatedClaims
			if d.DelegatorAddress == d.ValidatorAddress {
				dest = selfClaims
			}
			if dest[d.ValidatorAddress] == nil {
				dest[d.ValidatorAddress] = new(big.Int)
			}
			dest[d.ValidatorAddress].Add(dest[d.ValidatorAddress], amount)
			expectedReverse[string(types.ValidatorDelegationIndexKey(d.ValidatorAddress, d.DelegatorAddress))] = true
		case types.UnbondingKeyPrefix[0]:
			var u types.UnbondingEntry
			if err := decodeAccountingJSON(value, &u); err != nil {
				return err
			}
			amount, seq, err := validateAccountingUnbonding(&u)
			if err != nil {
				return err
			}
			if !bytes.Equal(key, types.UnbondingKey(u.Id)) || sequences[seq] {
				return accountingError("unbonding key or global sequence disagrees")
			}
			sequences[seq] = true
			if seq > maxSeq {
				maxSeq = seq
			}
			if u.Status == "pending" {
				liabilities.Add(liabilities, amount)
			}
			unbondings = append(unbondings, &u)
		case types.TierConfigKeyPrefix[0]:
			var tc types.TierConfig
			if err := decodeAccountingJSON(value, &tc); err != nil {
				return err
			}
			if !bytes.Equal(key, types.TierConfigKey(tc.Tier)) {
				return accountingError("tier key disagrees with value")
			}
			tiers[tc.Tier] = &tc
		case types.ParamsKey[0]:
			if !bytes.Equal(key, types.ParamsKey) {
				return accountingError("invalid params singleton key")
			}
			var p types.Params
			if err := decodeAccountingJSON(value, &p); err != nil {
				return err
			}
			if err := validateAccountingParams(&p); err != nil {
				return err
			}
			params = &p
		case types.ValidatorByDIDPrefix[0]:
			actualDIDs[string(key[1:])] = string(value)
		case types.UnbondingSeqKey[0]:
			if !bytes.Equal(key, types.UnbondingSeqKey) || len(value) != 8 {
				return accountingError("invalid sequence singleton")
			}
			storedSeq = binary.BigEndian.Uint64(value)
			if storedSeq == 0 {
				return accountingError("stored sequence is zero")
			}
		case types.RedelegationCooldownPrefix[0]:
			if err := canonicalAccountingAddress(string(key[1:])); err != nil {
				return err
			}
			if len(value) != 8 || binary.BigEndian.Uint64(value) == 0 || binary.BigEndian.Uint64(value) > math.MaxInt64 {
				return accountingError("invalid redelegation cooldown")
			}
		case types.ValidatorDelegationIndexPrefix[0]:
			if !bytes.Equal(value, []byte{1}) {
				return accountingError("invalid delegation reverse-index value")
			}
			actualReverse[string(key)] = true
		case types.AccountingSafetyKey[0]:
			if !bytes.Equal(key, types.AccountingSafetyKey) || !bytes.Equal(value, []byte{1}) {
				return accountingError("invalid accounting marker")
			}
		default:
			return accountingError("unrecognized custom staking key prefix %x", key[0])
		}
	}
	if params == nil || len(tiers) != len(params.TierConfigs) {
		return accountingError("missing params or incomplete tier keyspace")
	}
	for _, tc := range params.TierConfigs {
		if tc == nil || !reflect.DeepEqual(tc, tiers[tc.Tier]) {
			return accountingError("tier storage disagrees with params")
		}
	}
	if !reflect.DeepEqual(expectedDIDs, actualDIDs) || !reflect.DeepEqual(expectedReverse, actualReverse) {
		return accountingError("incomplete or orphaned custody index")
	}
	for addr, v := range validators {
		self, delegated := selfClaims[addr], delegatedClaims[addr]
		if self == nil {
			self = new(big.Int)
		}
		if delegated == nil {
			delegated = new(big.Int)
		}
		if self.String() != v.SelfDelegation || delegated.String() != v.DelegatedStake {
			return accountingError("validator aggregates disagree with individual claims: %s", addr)
		}
	}
	for addr := range selfClaims {
		if validators[addr] == nil {
			return accountingError("self claim has no validator")
		}
	}
	for addr := range delegatedClaims {
		if validators[addr] == nil {
			return accountingError("delegation has no validator")
		}
	}
	for _, u := range unbondings {
		if validators[u.ValidatorAddress] == nil {
			return accountingError("unbonding has no validator")
		}
	}
	if storedSeq < maxSeq {
		return accountingError("unbonding sequence behind historical claims")
	}
	if k.bankKeeper == nil {
		return accountingError("bank keeper unavailable")
	}
	balance := k.bankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(types.ModuleName))
	for _, coin := range balance {
		if coin.Denom != "uzrn" {
			return accountingError("unresolved non-uzrn module balance")
		}
	}
	if balance.AmountOf("uzrn").BigInt().Cmp(liabilities) != 0 {
		return accountingError("bank balance does not equal delegation plus pending-unbonding claims")
	}
	return nil
}

// accountingCapacity counts keys only, and is called when an admission can
// add a key. Payout and normal history updates never add keys.
func (k Keeper) accountingCapacity(ctx sdk.Context) (err error) {
	iter := k.getStore(ctx).Iterator(nil, nil)
	defer func() { err = errors.Join(err, accountingIteratorError(iter), iter.Close()) }()
	count := 0
	// App InitChainer adds this exact sentinel after module initialization.
	if !k.getStore(ctx).Has([]byte("_iavl_init")) {
		count++
	}
	for ; iter.Valid(); iter.Next() {
		count++
		if count > MaxAccountingStoreEntries {
			return accountingError("accounting key capacity reached")
		}
	}
	return nil
}

// accountingWrite guards all keeper writes once enabled, including metadata
// and reputation changes outside the message server.
func (k Keeper) accountingWrite(ctx sdk.Context, key, value []byte) {
	store := k.getStore(ctx)
	if !k.AccountingSafetyEnabled(ctx) {
		store.Set(key, value)
		return
	}
	if len(key) > MaxAccountingKeyBytes || len(value) > MaxAccountingRecordBytes {
		panic(accountingError("accounting record ceiling exceeded"))
	}
	var record any
	switch key[0] {
	case types.ValidatorKeyPrefix[0]:
		record = &types.Validator{}
	case types.DelegationKeyPrefix[0]:
		record = &types.Delegation{}
	case types.UnbondingKeyPrefix[0]:
		record = &types.UnbondingEntry{}
	case types.TierConfigKeyPrefix[0]:
		record = &types.TierConfig{}
	case types.ParamsKey[0]:
		record = &types.Params{}
	}
	if record != nil {
		if err := decodeAccountingJSON(value, record); err != nil {
			panic(err)
		}
	}
	if !store.Has(key) {
		iter := store.Iterator(nil, nil)
		count := 0
		if !store.Has([]byte("_iavl_init")) {
			count++
		}
		for ; iter.Valid(); iter.Next() {
			count++
			if count >= MaxAccountingStoreEntries {
				iter.Close()
				panic(accountingError("accounting key capacity reached"))
			}
		}
		if err := errors.Join(accountingIteratorError(iter), iter.Close()); err != nil {
			panic(accountingError("capacity scan failed: %v", err))
		}
	}
	store.Set(key, value)
}

// checkedAccountingValidator verifies the touched validator against its
// previously authenticated reverse index. Full activation and payout audits
// additionally prove no primary claim is missing from that index.
func (k Keeper) checkedAccountingValidator(ctx sdk.Context, addr string) (result *types.Validator, err error) {
	if err := canonicalAccountingAddress(addr); err != nil {
		return nil, err
	}
	bz := k.getStore(ctx).Get(types.ValidatorKey(addr))
	if bz == nil {
		return nil, types.ErrValidatorNotFound
	}
	var v types.Validator
	if err := decodeAccountingJSON(bz, &v); err != nil {
		return nil, err
	}
	if v.OperatorAddress != addr {
		return nil, accountingError("validator identity mismatch")
	}
	if err := validateAccountingValidator(&v); err != nil {
		return nil, err
	}
	self, delegated := new(big.Int), new(big.Int)
	iter := storetypes.KVStorePrefixIterator(k.getStore(ctx), types.DelegationsByValidatorPrefix(addr))
	defer func() { err = errors.Join(err, accountingIteratorError(iter), iter.Close()) }()
	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
		if count > MaxAccountingStoreEntries {
			return nil, accountingError("delegation index capacity exceeded")
		}
		if !bytes.Equal(iter.Value(), []byte{1}) {
			return nil, accountingError("invalid reverse index")
		}
		delegator := string(iter.Key()[len(types.DelegationsByValidatorPrefix(addr)):])
		d, err := k.checkedAccountingDelegation(ctx, delegator, addr)
		if err != nil {
			return nil, err
		}
		n, _ := accountingAmount(d.Amount, true)
		if delegator == addr {
			self.Add(self, n)
		} else {
			delegated.Add(delegated, n)
		}
	}
	if self.String() != v.SelfDelegation || delegated.String() != v.DelegatedStake {
		return nil, accountingError("validator aggregates disagree with touched claims")
	}
	return &v, nil
}

func (k Keeper) checkedAccountingDelegation(ctx sdk.Context, delegator, validator string) (*types.Delegation, error) {
	bz := k.getStore(ctx).Get(types.DelegationKey(delegator, validator))
	if bz == nil {
		return nil, types.ErrDelegationNotFound
	}
	var d types.Delegation
	if err := decodeAccountingJSON(bz, &d); err != nil {
		return nil, err
	}
	if d.DelegatorAddress != delegator || d.ValidatorAddress != validator {
		return nil, accountingError("delegation identity mismatch")
	}
	if _, err := validateAccountingDelegation(&d); err != nil {
		return nil, err
	}
	if !bytes.Equal(k.getStore(ctx).Get(types.ValidatorDelegationIndexKey(validator, delegator)), []byte{1}) {
		return nil, accountingError("delegation reverse index missing")
	}
	return &d, nil
}
