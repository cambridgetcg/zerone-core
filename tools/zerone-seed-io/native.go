package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	mathsdk "cosmossdk.io/math"
	feegrant "cosmossdk.io/x/feegrant"
	storekeyring "github.com/99designs/keyring"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	gogoproto "github.com/cosmos/gogoproto/proto"
	curve "github.com/decred/dcrd/dcrec/secp256k1/v4"
	pottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

func marshalNative(v any) []byte {
	var b []byte
	var e error
	if p, ok := v.(proto.Message); ok {
		b, e = (proto.MarshalOptions{Deterministic: true}).Marshal(p)
	} else {
		b, e = gogoproto.Marshal(v.(gogoproto.Message))
	}
	require(e == nil && len(b) <= maxProto, "invalid_response")
	return b
}
func unmarshalNative(b []byte, v any) error {
	if p, ok := v.(proto.Message); ok {
		return proto.Unmarshal(b, p)
	}
	return gogoproto.Unmarshal(b, v.(gogoproto.Message))
}
func anyNative(url string, v any) *codectypes.Any {
	return &codectypes.Any{TypeUrl: url, Value: marshalNative(v)}
}
func coin(amount string) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin("uzrn", mathsdk.NewIntFromBigInt(decimal(amount, 256, true))))
}
func buildUnsigned(c object) (body, auth, doc, simulation []byte) {
	chain := str(c, "chain_id")
	claimant := rawAccount(str(c, "claimant_account"), chain)
	sponsor := rawAccount(str(c, "sponsor_account"), chain)
	msg := &pottypes.MsgClaim{Claimant: claimant, PotId: str(c, "pot_id")}
	body = marshalNative(&txtypes.TxBody{Messages: []*codectypes.Any{anyNative(claimURL, msg)}, TimeoutHeight: uint64s(str(c, "timeout_height"), true)})
	key := &secp256k1.PubKey{Key: unb64(str(c, "signer_public_key_b64u"))}
	info := &txtypes.SignerInfo{PublicKey: anyNative("/cosmos.crypto.secp256k1.PubKey", key), ModeInfo: &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}}, Sequence: uint64s(str(c, "sequence"), false)}
	auth = marshalNative(&txtypes.AuthInfo{SignerInfos: []*txtypes.SignerInfo{info}, Fee: &txtypes.Fee{Amount: coin(str(c, "fee_amount_uzrn")), GasLimit: uint64s(str(c, "gas_limit"), true), Granter: sponsor}})
	doc = marshalNative(&txtypes.SignDoc{BodyBytes: body, AuthInfoBytes: auth, ChainId: str(c, "chain_reference"), AccountNumber: uint64s(str(c, "account_number"), false)})
	simulation = marshalNative(&txtypes.TxRaw{BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{{}}})
	return
}
func validatePlan(p, pol, plan object) {
	exact(plan, "protocol commitment commitment_hash observation_hash body_bytes_b64u body_bytes_hash auth_info_bytes_b64u auth_info_bytes_hash sign_doc_bytes_b64u sign_doc_bytes_hash simulation_tx_bytes_b64u simulation_tx_bytes_hash plan_id")
	require(str(plan, "protocol") == "agent-wallet-zerone.seed-plan/0.1", "plan_mismatch")
	c := obj(plan["commitment"])
	exact(c, "protocol profile_id source_digest genesis_hash chain_id chain_reference policy_hash capability_record_id intent_record_id claimant_account sponsor_account pot_id signer_key_id signer_public_key_b64u account_number sequence fee_amount_uzrn gas_limit timeout_height expires_at grant_spend_limit_uzrn grant_expires_at")
	require(str(c, "protocol") == "agent-wallet-zerone.seed-commitment/0.1", "plan_mismatch")
	for _, k := range []string{"profile_id", "source_digest", "genesis_hash", "chain_id", "chain_reference"} {
		require(str(c, k) == str(p, k), "plan_mismatch")
	}
	for _, k := range []string{"policy_hash", "claimant_account", "sponsor_account", "pot_id", "timeout_height", "grant_spend_limit_uzrn", "grant_expires_at"} {
		require(str(c, k) == str(pol, k), "plan_mismatch")
	}
	expiry := timestamp(str(c, "expires_at"))
	require(expiry.After(timestamp(str(pol, "not_before"))) && !expiry.After(timestamp(str(pol, "expires_at"))), "plan_mismatch")
	for _, k := range []string{"capability_record_id", "intent_record_id", "signer_key_id"} {
		require(isHash(str(c, k)), "plan_mismatch")
	}
	pub := unb64(str(c, "signer_public_key_b64u"))
	require(len(pub) == 33 && (pub[0] == 2 || pub[0] == 3) && digest(pub) == str(c, "signer_key_id"), "key_mismatch")
	_, parseErr := curve.ParsePubKey(pub)
	require(parseErr == nil, "key_mismatch")
	key := &secp256k1.PubKey{Key: pub}
	require(bytes.Equal(key.Address(), address(rawAccount(str(c, "claimant_account"), str(p, "chain_id")))), "key_mismatch")
	uint64s(str(c, "account_number"), false)
	uint64s(str(c, "sequence"), false)
	gas := uint64s(str(c, "gas_limit"), true)
	fee := decimal(str(c, "fee_amount_uzrn"), 256, true)
	require(gas >= 22222 && gas <= uint64s(str(pol, "max_gas"), true) && fee.Cmp(new(big.Int).SetUint64(gas)) >= 0 && fee.Cmp(decimal(str(pol, "max_fee_uzrn"), 256, true)) <= 0, "plan_mismatch")
	require(hashObject(c, "") == str(plan, "commitment_hash") && isHash(str(plan, "observation_hash")) && hashObject(plan, "plan_id") == str(plan, "plan_id"), "plan_mismatch")
	body, auth, doc, simulation := buildUnsigned(c)
	for k, b := range map[string][]byte{"body": body, "auth_info": auth, "sign_doc": doc, "simulation_tx": simulation} {
		require(bytes.Equal(unb64(str(plan, k+"_bytes_b64u")), b) && str(plan, k+"_bytes_hash") == digest(b), "plan_mismatch")
	}
}
func signedSummary(plan object, b []byte) object {
	c := obj(plan["commitment"])
	require(len(b) <= maxProto, "signature_invalid")
	var tx txtypes.TxRaw
	require(tx.Unmarshal(b) == nil && len(tx.Signatures) == 1 && len(tx.Signatures[0]) == 64, "signature_invalid")
	body, auth, doc, _ := buildUnsigned(c)
	require(bytes.Equal(tx.BodyBytes, body) && bytes.Equal(tx.AuthInfoBytes, auth), "signature_invalid")
	sig := tx.Signatures[0]
	order, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	half := new(big.Int).Rsh(new(big.Int).Set(order), 1)
	require(r.Sign() > 0 && r.Cmp(order) < 0 && s.Sign() > 0 && s.Cmp(half) <= 0, "signature_invalid")
	pub := &secp256k1.PubKey{Key: unb64(str(c, "signer_public_key_b64u"))}
	require(pub.VerifySignature(doc, sig), "signature_invalid")
	// Rebuilding, not re-marshalling the untrusted object, rejects unknown/repeated
	// fields and redundant defaults anywhere inside the exact committed components.
	expected := marshalNative(&txtypes.TxRaw{BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{sig}})
	require(bytes.Equal(expected, b), "signature_invalid")
	hash := digest(b)
	return object{"plan_id": plan["plan_id"], "commitment_hash": plan["commitment_hash"], "signer_key_id": c["signer_key_id"], "sign_doc_bytes_hash": plan["sign_doc_bytes_hash"], "signed_tx_bytes_hash": hash, "tx_hash": strings.ToUpper(strings.TrimPrefix(hash, "sha256:"))}
}
func operatorMessage(r object, now time.Time) object {
	command := str(r, "command")
	var v any
	var url string
	if command == "operator-admit" {
		authority := str(r, "authority")
		a := str(r, "address")
		address(authority)
		address(a)
		v = &pottypes.MsgAddBootstrapEntry{Authority: authority, Addresses: []string{a}}
		url = "/zerone.claiming_pot.v1.MsgAddBootstrapEntry"
	} else {
		granter, grantee := str(r, "granter"), str(r, "grantee")
		address(granter)
		address(grantee)
		require(granter != grantee, "invalid_request")
		if command == "operator-revoke" {
			v = &feegrant.MsgRevokeAllowance{Granter: granter, Grantee: grantee}
			url = "/cosmos.feegrant.v1beta1.MsgRevokeAllowance"
		} else {
			exp := timestamp(str(r, "expires_at"))
			supplied := timestamp(str(r, "now"))
			require(exp.After(supplied) && exp.After(now), "invalid_request")
			basic := &feegrant.BasicAllowance{SpendLimit: coin(str(r, "spend_limit_uzrn")), Expiration: &exp}
			allowed := &feegrant.AllowedMsgAllowance{Allowance: anyNative("/cosmos.feegrant.v1beta1.BasicAllowance", basic), AllowedMessages: []string{claimURL}}
			v = &feegrant.MsgGrantAllowance{Granter: granter, Grantee: grantee, Allowance: anyNative("/cosmos.feegrant.v1beta1.AllowedMsgAllowance", allowed)}
			url = "/cosmos.feegrant.v1beta1.MsgGrantAllowance"
		}
	}
	b := marshalNative(v)
	return object{"type_url": url, "value_b64u": b64(b), "value_hash": digest(b)}
}
func keyCodec() *codec.ProtoCodec {
	reg := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(reg)
	return codec.NewProtoCodec(reg)
}

// SDK Key() invokes a legacy migration internally. The injected storage seam
// denies every write, delete, list, metadata read and non-selected key, so a
// legacy record cannot silently migrate or emit the SDK's success diagnostic.
// Local record custody remains inside the native SDK; this is not a key export.
type selectedReadOnlyKeyring struct {
	source   storekeyring.Keyring
	selected string
}

var errReadOnlyKeyring = errors.New("selected keyring is read-only")

func (k selectedReadOnlyKeyring) Get(key string) (storekeyring.Item, error) {
	if key != k.selected {
		return storekeyring.Item{}, errReadOnlyKeyring
	}
	item, err := k.source.Get(key)
	if err != nil || len(item.Data) > maxProto {
		return storekeyring.Item{}, errReadOnlyKeyring
	}
	return item, nil
}
func (k selectedReadOnlyKeyring) GetMetadata(string) (storekeyring.Metadata, error) {
	return storekeyring.Metadata{}, errReadOnlyKeyring
}
func (k selectedReadOnlyKeyring) Set(storekeyring.Item) error { return errReadOnlyKeyring }
func (k selectedReadOnlyKeyring) Remove(string) error         { return errReadOnlyKeyring }
func (k selectedReadOnlyKeyring) Keys() ([]string, error)     { return nil, errReadOnlyKeyring }

// Only explicit existing file-backed SDK keyrings are supported in this first
// reference. os/pass are refused: their SDK implementations discover ambient
// stores or may silently choose another backend. This is not hardware custody.
func signExisting(ctx context.Context, r object, opts options) object {
	checkNewPrivatePath(str(r, "signed_tx_path"))
	k := obj(r["keyring"])
	backend := str(k, "backend")
	home := str(k, "home")
	name := str(k, "key_name")
	require(backend == "file" || backend == "test", "invalid_request")
	if backend == "test" {
		require(opts.disposable && strings.HasPrefix(str(obj(r["profile"]), "chain_reference"), "seed-local-"), "profile_mismatch")
	}
	dir := filepath.Join(home, "keyring-"+backend)
	d := openPrivateDir(dir)
	d.Close()
	// Existing named record is checked before opening the SDK backend; no create,
	// import, export, migrate, rename or delete API is exposed by this helper.
	record := filepath.Join(dir, name+".info")
	f := openPrivateFile(record)
	st, statErr := f.Stat()
	f.Close()
	require(statErr == nil && st.Size() <= maxJSON, "limit_exceeded")
	var password func(string) (string, error)
	if backend == "test" {
		password = func(string) (string, error) { return "test", nil }
	} else {
		require(opts.unlockTerminal != "", "invalid_request")
		kh := readPrivate(ctx, filepath.Join(dir, "keyhash"), 4096)
		tty, err := os.OpenFile(opts.unlockTerminal, os.O_RDWR, 0)
		require(err == nil, "unsafe_path")
		defer tty.Close()
		st, err := tty.Stat()
		require(err == nil && st.Mode()&os.ModeCharDevice != 0 && term.IsTerminal(int(tty.Fd())), "unsafe_path")
		// No prompt on stdout/stderr; the separately attached terminal is explicit.
		_, err = io.WriteString(tty, "Unlock existing seed keyring: ")
		require(err == nil, "key_mismatch")
		secret, err := unlockPassword(ctx, tty)
		require(err == nil, "key_mismatch")
		defer clear(secret)
		require(bcrypt.CompareHashAndPassword(kh, secret) == nil, "key_mismatch")
		password = func(string) (string, error) { return string(secret), nil }
	}
	db, err := storekeyring.Open(storekeyring.Config{AllowedBackends: []storekeyring.BackendType{storekeyring.FileBackend}, ServiceName: sdk.KeyringServiceName(), FileDir: dir, FilePasswordFunc: password})
	require(err == nil, "key_mismatch")
	kr := keyring.NewInMemoryWithKeyring(selectedReadOnlyKeyring{source: db, selected: name + ".info"}, keyCodec())
	rec, err := kr.Key(name)
	require(err == nil && rec.Name == name && rec.GetLocal() != nil, "key_mismatch")
	pub, err := rec.GetPubKey()
	require(err == nil && pub.Type() == "secp256k1", "key_mismatch")
	plan := obj(r["plan"])
	c := obj(plan["commitment"])
	require(bytes.Equal(pub.Bytes(), unb64(str(c, "signer_public_key_b64u"))), "key_mismatch")
	require(budgetError(ctx) == nil, "signing_unknown")
	require(time.Now().Before(timestamp(str(c, "expires_at"))), "plan_mismatch")
	_, _, doc, _ := buildUnsigned(c)
	// From here the runtime's previously persisted possible-signing state is sticky.
	defer func() {
		if recover() != nil {
			panic(failure("signing_unknown"))
		}
	}()
	signature, returned, err := kr.Sign(name, doc, signing.SignMode_SIGN_MODE_DIRECT)
	require(err == nil && returned != nil && bytes.Equal(returned.Bytes(), pub.Bytes()), "signing_unknown")
	body, auth, _, _ := buildUnsigned(c)
	b := marshalNative(&txtypes.TxRaw{BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{signature}})
	summary := signedSummary(plan, b)
	require(budgetError(ctx) == nil, "signing_unknown")
	writePrivateExclusive(str(r, "signed_tx_path"), b)
	require(budgetError(ctx) == nil, "signing_unknown")
	return summary
}
func unlockPassword(ctx context.Context, tty *os.File) ([]byte, error) {
	type result struct {
		secret []byte
		err    error
	}
	ch := make(chan result)
	go func() {
		secret, err := term.ReadPassword(int(tty.Fd()))
		select {
		case ch <- result{secret, err}:
		case <-ctx.Done():
			clear(secret)
		}
	}()
	select {
	case result := <-ch:
		return result.secret, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func txHashBytes(s string) []byte {
	b, e := hex.DecodeString(s)
	require(e == nil && len(b) == 32, "invalid_request")
	return b
}
