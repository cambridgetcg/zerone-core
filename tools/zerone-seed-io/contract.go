package main

// This is the external custody/transport boundary, NOT Wallet admission authority.
// The caller must persist and authorize its possible-effect boundary before invocation.
import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type object = map[string]any

const protocol = "zerone-seed-io/0.1"
const claimURL = "/zerone.claiming_pot.v1.MsgClaim"
const maxJSON = 262144
const maxProto = 16384
const stamp = "2006-01-02T15:04:05.000Z"

type failure string

func (e failure) Error() string { return string(e) }
func require(ok bool, code string) {
	if !ok {
		panic(failure(code))
	}
}
func obj(v any) object              { o, ok := v.(map[string]any); require(ok, "invalid_request"); return o }
func str(o object, k string) string { s, ok := o[k].(string); require(ok, "invalid_request"); return s }
func num(o object, k string) int64 {
	n, ok := o[k].(json.Number)
	require(ok, "invalid_request")
	i, e := strconv.ParseInt(string(n), 10, 64)
	require(e == nil, "invalid_request")
	return i
}
func exact(o object, fields string) {
	keys := strings.Fields(fields)
	require(len(o) == len(keys), "invalid_request")
	for _, k := range keys {
		_, ok := o[k]
		require(ok, "invalid_request")
	}
}
func isHash(s string) bool   { return regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(s) }
func digest(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func hashObject(o object, omit string) string {
	c := object{}
	for k, v := range o {
		if k != omit {
			c[k] = v
		}
	}
	return digest(canonical(c))
}
func canonical(v any) []byte {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	require(e.Encode(v) == nil, "invalid_request")
	return bytes.TrimSuffix(b.Bytes(), []byte{'\n'})
}
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
func unb64(s string) []byte {
	b, e := base64.RawURLEncoding.DecodeString(s)
	require(e == nil && b64(b) == s && len(b) <= maxProto, "invalid_request")
	return b
}
func decimal(s string, bits int, positive bool) *big.Int {
	require(regexp.MustCompile(`^(0|[1-9][0-9]{0,77})$`).MatchString(s), "invalid_request")
	n, ok := new(big.Int).SetString(s, 10)
	require(ok && n.BitLen() <= bits && (!positive || n.Sign() > 0), "invalid_request")
	return n
}
func uint64s(s string, positive bool) uint64 { return decimal(s, 64, positive).Uint64() }
func timestamp(s string) time.Time {
	t, e := time.Parse(stamp, s)
	require(e == nil && t.Format(stamp) == s, "invalid_request")
	return t
}
func nowStamp(t time.Time) string { return t.UTC().Truncate(time.Millisecond).Format(stamp) }
func ascii(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
func address(s string) sdk.AccAddress {
	require(ascii(s) && strings.ToLower(s) == s, "invalid_request")
	a, e := sdk.AccAddressFromBech32(s)
	require(e == nil && len(a) == 20 && a.String() == s, "invalid_request")
	return a
}
func rawAccount(s, chain string) string {
	prefix := chain + ":"
	require(strings.HasPrefix(s, prefix), "invalid_request")
	a := strings.TrimPrefix(s, prefix)
	address(a)
	return a
}

// Token parsing rejects duplicate members before map construction. The canonical
// byte comparison also rejects escaped surrogate aliases, whitespace and -0.
func parseJSON(b []byte) object {
	require(len(b) <= maxJSON+1 && utf8.Valid(b), "limit_exceeded")
	if bytes.HasSuffix(b, []byte{'\n'}) {
		b = b[:len(b)-1]
	}
	require(len(b) <= maxJSON, "limit_exceeded")
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	count := 0
	var read func(int) any
	read = func(depth int) any {
		count++
		require(depth <= 32 && count <= 4096, "limit_exceeded")
		tok, e := d.Token()
		require(e == nil, "invalid_request")
		switch x := tok.(type) {
		case json.Delim:
			if x == '{' {
				o := object{}
				for d.More() {
					k, e := d.Token()
					require(e == nil, "invalid_request")
					key, ok := k.(string)
					require(ok && ascii(key) && len(key) <= 4096, "invalid_request")
					_, dup := o[key]
					require(!dup, "invalid_request")
					o[key] = read(depth + 1)
				}
				t, e := d.Token()
				require(e == nil && t == json.Delim('}'), "invalid_request")
				return o
			}
			require(x == '[', "invalid_request")
			a := []any{}
			for d.More() {
				a = append(a, read(depth+1))
			}
			t, e := d.Token()
			require(e == nil && t == json.Delim(']'), "invalid_request")
			return a
		case string:
			require(len(x) <= 4096 && !strings.ContainsRune(x, 0), "limit_exceeded")
			return x
		case json.Number:
			i, e := strconv.ParseInt(string(x), 10, 64)
			require(e == nil && i >= -9007199254740991 && i <= 9007199254740991 && string(x) != "-0", "invalid_request")
			return x
		case bool, nil:
			return x
		default:
			panic(failure("invalid_request"))
		}
	}
	o := obj(read(0))
	_, e := d.Token()
	require(e == io.EOF, "invalid_request")
	require(bytes.Equal(canonical(o), b), "invalid_request")
	var checkStrings func(any, string)
	checkStrings = func(v any, k string) {
		switch x := v.(type) {
		case string:
			if k != "home" && k != "signed_tx_path" && k != "ca_file" && k != "genesis_file" && k != "runtime_file" && k != "source_manifest_file" {
				require(ascii(x), "invalid_request")
			}
		case map[string]any:
			for key, v := range x {
				checkStrings(v, key)
			}
		case []any:
			for _, v := range x {
				checkStrings(v, k)
			}
		}
	}
	checkStrings(o, "")
	return o
}

func validateProfile(p object) {
	exact(p, "protocol chain_reference chain_id native_asset_id claiming_pot_account genesis_hash source_digest zerone_core_commit cosmos_sdk_version runtime_sha256 helper_sha256 native_denom bech32_prefix seed_amount_uzrn claim_type_url claim_gas_floor tx_gas_cap min_gas_price_uzrn confirmation_depth profile_id")
	require(str(p, "protocol") == "agent-wallet-zerone.seed-profile/0.1" && str(p, "cosmos_sdk_version") == "v0.53.8", "profile_mismatch")
	ref := str(p, "chain_reference")
	require(regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`).MatchString(ref), "profile_mismatch")
	chain := "cosmos:" + ref
	require(str(p, "chain_id") == chain && str(p, "native_asset_id") == chain+"/denom:uzrn" && str(p, "claiming_pot_account") == chain+":"+authtypes.NewModuleAddress("claiming_pot").String(), "profile_mismatch")
	for k, want := range map[string]string{"native_denom": "uzrn", "bech32_prefix": "zrn", "seed_amount_uzrn": "222000", "claim_type_url": claimURL, "claim_gas_floor": "22222", "tx_gas_cap": "11111111", "min_gas_price_uzrn": "1"} {
		require(str(p, k) == want, "profile_mismatch")
	}
	require(num(p, "confirmation_depth") == 1, "profile_mismatch")
	for _, k := range []string{"genesis_hash", "source_digest", "runtime_sha256", "helper_sha256", "profile_id"} {
		require(isHash(str(p, k)), "profile_mismatch")
	}
	require(regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(str(p, "zerone_core_commit")), "profile_mismatch")
	require(hashObject(p, "profile_id") == str(p, "profile_id"), "profile_mismatch")
}
func validatePolicy(p, pol object) {
	exact(pol, "protocol profile_id node_trust_id claimant_account sponsor_account pot_id max_intents seed_amount_uzrn max_fee_uzrn max_gas grant_spend_limit_uzrn grant_expires_at setup_fee_budget_uzrn not_before expires_at timeout_height max_observation_age_seconds max_height_lag policy_hash")
	require(str(pol, "protocol") == "agent-wallet-zerone.seed-policy/0.1" && str(pol, "profile_id") == str(p, "profile_id") && str(pol, "seed_amount_uzrn") == "222000" && num(pol, "max_intents") == 1, "policy_mismatch")
	claimant := rawAccount(str(pol, "claimant_account"), str(p, "chain_id"))
	sponsor := rawAccount(str(pol, "sponsor_account"), str(p, "chain_id"))
	require(claimant != sponsor && str(pol, "pot_id") == "bootstrap-"+claimant, "policy_mismatch")
	require(isHash(str(pol, "node_trust_id")), "policy_mismatch")
	maxFee := decimal(str(pol, "max_fee_uzrn"), 256, true)
	grant := decimal(str(pol, "grant_spend_limit_uzrn"), 256, true)
	require(maxFee.Cmp(grant) <= 0 && maxFee.Cmp(big.NewInt(22222)) >= 0, "policy_mismatch")
	gas := uint64s(str(pol, "max_gas"), true)
	require(gas >= 22222 && gas <= 11111111, "policy_mismatch")
	setup := decimal(str(pol, "setup_fee_budget_uzrn"), 256, false)
	require(new(big.Int).Add(new(big.Int).Set(grant), setup).BitLen() <= 256, "policy_mismatch")
	uint64s(str(pol, "timeout_height"), true)
	start := timestamp(str(pol, "not_before"))
	end := timestamp(str(pol, "expires_at"))
	exp := timestamp(str(pol, "grant_expires_at"))
	require(start.Before(end) && !end.After(exp), "policy_mismatch")
	require(num(pol, "max_observation_age_seconds") >= 1 && num(pol, "max_observation_age_seconds") <= 300 && num(pol, "max_height_lag") >= 0 && num(pol, "max_height_lag") <= 100, "policy_mismatch")
	require(hashObject(pol, "policy_hash") == str(pol, "policy_hash"), "policy_mismatch")
}
func validateRequest(r object, command string) {
	require(str(r, "protocol") == protocol && str(r, "command") == command, "invalid_request")
	require(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`).MatchString(str(r, "request_id")), "invalid_request")
	require(num(r, "timeout_ms") >= 1 && num(r, "timeout_ms") <= 30000, "invalid_request")
	base := "protocol request_id timeout_ms command profile "
	fields := map[string]string{
		"inspect": "policy node height", "simulate": "policy plan node height", "sign": "policy plan keyring signed_tx_path", "verify": "policy plan signed_tx_path", "submit": "policy plan node signed_tx_path expected_tx_hash", "lookup": "policy plan node tx_hash", "operator-grant": "granter grantee spend_limit_uzrn expires_at now", "operator-revoke": "granter grantee", "operator-admit": "authority address",
	}
	f, ok := fields[command]
	require(ok, "unsupported_command")
	exact(r, base+f)
	p := obj(r["profile"])
	validateProfile(p)
	if v, ok := r["policy"]; ok {
		validatePolicy(p, obj(v))
	}
	if v, ok := r["height"]; ok && v != nil {
		uint64s(str(r, "height"), true)
	}
	if command == "simulate" {
		require(r["height"] != nil, "invalid_request")
	}
	for _, k := range []string{"tx_hash", "expected_tx_hash"} {
		if _, ok := r[k]; ok {
			require(regexp.MustCompile(`^[0-9A-F]{64}$`).MatchString(str(r, k)), "invalid_request")
		}
	}
	if _, ok := r["plan"]; ok {
		validatePlan(p, obj(r["policy"]), obj(r["plan"]))
	}
	if v, ok := r["keyring"]; ok {
		k := obj(v)
		exact(k, "backend home key_name")
		require(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`).MatchString(str(k, "key_name")), "invalid_request")
	}
}
