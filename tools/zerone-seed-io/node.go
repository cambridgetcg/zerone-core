package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	feegrant "cosmossdk.io/x/feegrant"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	pottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	vesttypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nodeFailure string

func known(ok bool, reason string) {
	if !ok {
		panic(nodeFailure(reason))
	}
}

type rpcFailure struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}
type nodeClient struct {
	ctx                   context.Context
	client                *http.Client
	url                   string
	profile, policy, node object
	now                   func() time.Time
	sequence              int
}

func newNode(ctx context.Context, r object, now func() time.Time) *nodeClient {
	n := obj(r["node"])
	mode := str(n, "mode")
	fields := "node_trust_id rpc_url grpc_address mode"
	if mode == "tls" {
		fields += " tls_server_name ca_file"
	}
	exact(n, fields)
	require(str(n, "node_trust_id") == str(obj(r["policy"]), "node_trust_id"), "policy_mismatch")
	u, e := url.Parse(str(n, "rpc_url"))
	require(e == nil && u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/") && u.Opaque == "", "invalid_request")
	host, port, e := net.SplitHostPort(str(n, "grpc_address"))
	require(e == nil && port != "" && !strings.ContainsAny(host, "/@?#"), "invalid_request")
	_, e = strconv.ParseUint(port, 10, 16)
	require(e == nil && port != "0", "invalid_request")
	transport := &http.Transport{Proxy: nil, MaxConnsPerHost: 1, MaxIdleConns: 1, DisableCompression: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second}
	if mode == "local" {
		rpcIP, grpcIP := net.ParseIP(u.Hostname()), net.ParseIP(host)
		require(u.Scheme == "http" && rpcIP != nil && grpcIP != nil && rpcIP.IsLoopback() && grpcIP.IsLoopback(), "invalid_request")
	} else {
		require(mode == "tls" && u.Scheme == "https" && str(n, "tls_server_name") != "", "invalid_request")
		pool := x509.NewCertPool()
		require(pool.AppendCertsFromPEM(readPublic(ctx, str(n, "ca_file"), maxJSON)), "invalid_request")
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: str(n, "tls_server_name")}
	}
	// All native state queries use typed protobuf over this same approved Comet
	// ABCI transport; grpc_address is explicit trust configuration, never a fallback.
	return &nodeClient{ctx: ctx, client: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, url: u.String(), profile: obj(r["profile"]), policy: obj(r["policy"]), node: n, now: now}
}
func (n *nodeClient) close() { n.client.CloseIdleConnections() }

// Native Comet JSON is not canonical Wallet JSON, but duplicate members,
// excessive depth/count and oversized strings are still invalid responses.
func validateNodeJSON(data []byte) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	count := 0
	var read func(int)
	read = func(depth int) {
		count++
		known(depth <= 32 && count <= 4096, "response_limit")
		tok, err := d.Token()
		known(err == nil, "invalid_response")
		switch x := tok.(type) {
		case json.Delim:
			if x == '{' {
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					known(err == nil, "invalid_response")
					s, ok := key.(string)
					known(ok && len(s) <= 4096 && !seen[s], "invalid_response")
					seen[s] = true
					read(depth + 1)
				}
				end, err := d.Token()
				known(err == nil && end == json.Delim('}'), "invalid_response")
			} else {
				known(x == '[', "invalid_response")
				for d.More() {
					read(depth + 1)
				}
				end, err := d.Token()
				known(err == nil && end == json.Delim(']'), "invalid_response")
			}
		case string:
			known(len(x) <= 4096, "response_limit")
		}
	}
	read(0)
	_, err := d.Token()
	known(err == io.EOF, "invalid_response")
}

func (n *nodeClient) rpc(method string, params object, out any) *rpcFailure {
	allowed := map[string]bool{"status": true, "block": true, "genesis": true, "abci_query": true, "broadcast_tx_sync": true, "tx": true, "num_unconfirmed_txs": true, "block_results": true}
	require(allowed[method], "unsupported_command")
	n.sequence++
	id := n.sequence
	b := canonical(object{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	req, e := http.NewRequestWithContext(n.ctx, http.MethodPost, n.url, bytes.NewReader(b))
	known(e == nil, "unavailable")
	req.Header.Set("Content-Type", "application/json")
	res, e := n.client.Do(req)
	known(e == nil, "unavailable")
	defer res.Body.Close()
	known(res.StatusCode == 200, "unavailable")
	data, e := io.ReadAll(io.LimitReader(res.Body, maxJSON+1))
	known(e == nil, "unavailable")
	known(len(data) <= maxJSON, "response_limit")
	validateNodeJSON(data)
	var env struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *rpcFailure     `json:"error"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	known(decoder.Decode(&env) == nil && env.JSONRPC == "2.0" && env.ID == id, "invalid_response")
	if env.Error != nil {
		known(len(env.Result) == 0 || string(env.Result) == "null", "invalid_response")
		return env.Error
	}
	known(len(env.Result) > 0 && string(env.Result) != "null" && cmtjson.Unmarshal(env.Result, out) == nil, "invalid_response")
	return nil
}
func (n *nodeClient) status() *rpctypes.ResultStatus {
	var s rpctypes.ResultStatus
	known(n.rpc("status", object{}, &s) == nil, "unavailable")
	known(s.NodeInfo.Network == str(n.profile, "chain_reference"), "chain_mismatch")
	known(!s.SyncInfo.CatchingUp, "catching_up")
	known(s.SyncInfo.LatestBlockHeight > 0 && len(s.SyncInfo.LatestBlockHash) == 32, "invalid_response")
	n.fresh(s.SyncInfo.LatestBlockTime)
	return &s
}
func (n *nodeClient) fresh(t time.Time) {
	now := n.now()
	known(!t.IsZero() && !t.After(now) && now.Sub(t) <= time.Duration(num(n.policy, "max_observation_age_seconds"))*time.Second, "stale")
}
func (n *nodeClient) block(h int64) *rpctypes.ResultBlock {
	known(h > 0, "invalid_response")
	var b rpctypes.ResultBlock
	known(n.rpc("block", object{"height": strconv.FormatInt(h, 10)}, &b) == nil, "unavailable")
	known(b.Block != nil && b.Block.Height == h && b.Block.ChainID == str(n.profile, "chain_reference") && len(b.BlockID.Hash) == 32, "incoherent_height")
	// Header hash alone does not authenticate the separately returned tx list.
	// Recompute the data root explicitly, before any Block.Hash fill-in behavior.
	known(bytes.Equal(b.Block.DataHash, cmttypes.Txs(b.Block.Txs).Hash()) && bytes.Equal(b.Block.EvidenceHash, b.Block.Evidence.Hash()), "incoherent_height")
	known(b.Block.LastCommit != nil && bytes.Equal(b.Block.LastCommitHash, b.Block.LastCommit.Hash()), "incoherent_height")
	known(bytes.Equal(b.BlockID.Hash, b.Block.Header.Hash()), "incoherent_height")
	return &b
}
func blockAnchor(b *rpctypes.ResultBlock) object {
	return object{"height": strconv.FormatInt(b.Block.Height, 10), "block_hash": "sha256:" + strings.ToLower(b.BlockID.Hash.String()), "block_time": nowStamp(b.Block.Time)}
}
func (n *nodeClient) checkGenesis(trust trustedArtifacts) {
	var res rpctypes.ResultGenesis
	known(n.rpc("genesis", object{}, &res) == nil, "unavailable")
	known(sameGenesis(trust.genesis, res.Genesis), "genesis_mismatch")
}
func (n *nodeClient) start(height any) *rpctypes.ResultBlock {
	s := n.status()
	h := s.SyncInfo.LatestBlockHeight
	if height != nil {
		value := uint64s(height.(string), true)
		known(value <= uint64(h), "incoherent_height")
		h = int64(value)
	}
	known(s.SyncInfo.LatestBlockHeight-h <= num(n.policy, "max_height_lag"), "stale")
	b := n.block(h)
	n.fresh(b.Block.Time)
	if h == s.SyncInfo.LatestBlockHeight {
		known(bytes.Equal(b.BlockID.Hash, s.SyncInfo.LatestBlockHash), "incoherent_height")
	}
	return b
}
func (n *nodeClient) finish(b *rpctypes.ResultBlock) object {
	again := n.block(b.Block.Height)
	known(bytes.Equal(again.BlockID.Hash, b.BlockID.Hash), "incoherent_height")
	s := n.status()
	known(s.SyncInfo.LatestBlockHeight >= b.Block.Height && s.SyncInfo.LatestBlockHeight-b.Block.Height <= num(n.policy, "max_height_lag"), "stale")
	if s.SyncInfo.LatestBlockHeight == b.Block.Height {
		known(bytes.Equal(s.SyncInfo.LatestBlockHash, b.BlockID.Hash), "incoherent_height")
	}
	n.fresh(b.Block.Time)
	return object{"trust": "configured_full_node", "node_trust_id": n.node["node_trust_id"], "profile_id": n.profile["profile_id"], "chain_id": n.profile["chain_id"], "genesis_hash": n.profile["genesis_hash"], "anchor": blockAnchor(b), "latest_height": strconv.FormatInt(s.SyncInfo.LatestBlockHeight, 10), "catching_up": false, "observed_at": nowStamp(n.now())}
}
func (n *nodeClient) query(path string, req, out any, h int64) *abci.ResponseQuery {
	b := marshalNative(req)
	var res rpctypes.ResultABCIQuery
	known(n.rpc("abci_query", object{"path": path, "data": fmt.Sprintf("%x", b), "height": strconv.FormatInt(h, 10), "prove": false}, &res) == nil, "unavailable")
	known(res.Response.Height == h, "incoherent_height")
	if res.Response.Code != 0 {
		return &res.Response
	}
	known(len(res.Response.Value) <= maxProto, "response_limit")
	known(unmarshalNative(res.Response.Value, out) == nil, "invalid_response")
	return nil
}
func absentError(res *abci.ResponseQuery, err error) bool {
	if res == nil {
		return false
	}
	space, code, log := errorsmod.ABCIInfo(err, false)
	return res.Code == code && res.Codespace == space && res.Log == log
}
func (n *nodeClient) account(h int64) object {
	full := str(n.policy, "claimant_account")
	a := rawAccount(full, str(n.profile, "chain_id"))
	var res authtypes.QueryAccountResponse
	err := n.query("/cosmos.auth.v1beta1.Query/Account", &authtypes.QueryAccountRequest{Address: a}, &res, h)
	if err != nil {
		expected := errorsmod.Wrap(sdkerrors.ErrKeyNotFound, status.Errorf(codes.NotFound, "account %s not found", a).Error())
		if absentError(err, expected) {
			return object{"status": "absent", "account": full}
		}
		panic(nodeFailure("not_found_unproven"))
	}
	known(res.Account != nil && res.Account.TypeUrl == "/cosmos.auth.v1beta1.BaseAccount", "unsupported_state")
	var acc authtypes.BaseAccount
	known(acc.Unmarshal(res.Account.Value) == nil && acc.Address == a, "invalid_response")
	var key any
	if acc.PubKey != nil {
		known(acc.PubKey.TypeUrl == "/cosmos.crypto.secp256k1.PubKey", "unsupported_state")
		var pub secp256k1.PubKey
		known(pub.Unmarshal(acc.PubKey.Value) == nil && len(pub.Key) == 33 && bytes.Equal(pub.Address(), address(a)), "invalid_response")
		key = object{"type_url": acc.PubKey.TypeUrl, "key_b64u": b64(pub.Key)}
	}
	return object{"status": "found", "account": full, "account_number": strconv.FormatUint(acc.AccountNumber, 10), "sequence": strconv.FormatUint(acc.Sequence, 10), "public_key": key}
}
func (n *nodeClient) allowance(h int64) object {
	granter := rawAccount(str(n.policy, "sponsor_account"), str(n.profile, "chain_id"))
	grantee := rawAccount(str(n.policy, "claimant_account"), str(n.profile, "chain_id"))
	var res feegrant.QueryAllowanceResponse
	err := n.query("/cosmos.feegrant.v1beta1.Query/Allowance", &feegrant.QueryAllowanceRequest{Granter: granter, Grantee: grantee}, &res, h)
	if err != nil {
		// SDK v0.53.8 + feegrant v0.2.0 exact absence path: keeper.getGrant
		// ErrNotFound -> gRPC Internal -> BaseApp ErrUnknownRequest. Not an arbitrary
		// HTTP404, gRPC Unknown, or substring match. Trace-mode logs remain unknown.
		expected := errorsmod.Wrap(sdkerrors.ErrUnknownRequest, status.Error(codes.Internal, sdkerrors.ErrNotFound.Wrap("fee-grant not found").Error()).Error())
		if absentError(err, expected) {
			return object{"status": "absent"}
		}
		panic(nodeFailure("not_found_unproven"))
	}
	g := res.Allowance
	known(g != nil && g.Granter == granter && g.Grantee == grantee && g.Allowance != nil && g.Allowance.TypeUrl == "/cosmos.feegrant.v1beta1.AllowedMsgAllowance", "unsupported_state")
	var allowed feegrant.AllowedMsgAllowance
	known(allowed.Unmarshal(g.Allowance.Value) == nil && len(allowed.AllowedMessages) == 1 && allowed.AllowedMessages[0] == claimURL && allowed.Allowance != nil && allowed.Allowance.TypeUrl == "/cosmos.feegrant.v1beta1.BasicAllowance", "unsupported_state")
	var basic feegrant.BasicAllowance
	known(basic.Unmarshal(allowed.Allowance.Value) == nil && len(basic.SpendLimit) == 1 && basic.SpendLimit[0].Denom == "uzrn" && basic.SpendLimit[0].Amount.IsPositive() && basic.Expiration != nil, "unsupported_state")
	exp := basic.Expiration.UTC()
	known(exp.Equal(exp.Truncate(time.Millisecond)), "unsupported_state")
	decimal(basic.SpendLimit[0].Amount.String(), 256, true)
	return object{"status": "found", "granter": granter, "grantee": grantee, "type_url": g.Allowance.TypeUrl, "inner_type_url": allowed.Allowance.TypeUrl, "allowed_messages": []string{claimURL}, "spend_limit_uzrn": basic.SpendLimit[0].Amount.String(), "expires_at": nowStamp(exp)}
}
func (n *nodeClient) potAndClaim(h int64) (any, any) {
	id := str(n.policy, "pot_id")
	a := rawAccount(str(n.policy, "claimant_account"), str(n.profile, "chain_id"))
	var res pottypes.QueryPotResponse
	err := n.query("/zerone.claiming_pot.v1.Query/QueryPot", &pottypes.QueryPotRequest{Id: id}, &res, h)
	if err != nil {
		// The registered keeper error exposes gRPC Unknown even through fmt %w;
		// pinned BaseApp maps it to sdk6, not the non-gRPC sdk18 branch.
		// Exact requested-pot log, codespace and code remain mandatory.
		expected := errorsmod.Wrap(sdkerrors.ErrUnknownRequest, fmt.Errorf("%w: %s", pottypes.ErrPotNotFound, id).Error())
		if absentError(err, expected) {
			return nil, nil
		}
		panic(nodeFailure("not_found_unproven"))
	}
	p := res.Pot
	known(p != nil && p.Id == id && p.Schedule != nil && p.Eligibility != nil && len(p.Eligibility.Whitelist) == 1 && p.Eligibility.Whitelist[0] == a, "unsupported_state")
	// Validate the bounded seed shape BEFORE the unpaginated QueryClaims call.
	known(p.TotalAmount == "222000" && p.Schedule.StartBlock > 0 && p.Schedule.EndBlock > p.Schedule.StartBlock && p.Schedule.EndBlock-p.Schedule.StartBlock == 1 && p.Schedule.CliffBlocks == 0 && p.Schedule.PeriodBlocks == 0 && p.Eligibility.MinStakingTier == 0 && p.Eligibility.MinRegistrationAge == 0, "unsupported_state")
	decimal(p.ClaimedAmount, 256, false)
	statuses := map[pottypes.PotStatus]string{0: "unspecified", 1: "active", 2: "depleted", 3: "expired"}
	s, ok := statuses[p.Status]
	known(ok, "unsupported_state")
	pot := object{"pot_id": p.Id, "status": s, "total_amount_uzrn": p.TotalAmount, "claimed_amount_uzrn": p.ClaimedAmount, "start_block": strconv.FormatUint(p.Schedule.StartBlock, 10), "end_block": strconv.FormatUint(p.Schedule.EndBlock, 10), "cliff_blocks": "0", "period_blocks": "0", "min_staking_tier": 0, "min_registration_age": "0", "whitelist": []string{a}}
	var cr pottypes.QueryClaimsResponse
	known(n.query("/zerone.claiming_pot.v1.Query/QueryClaims", &pottypes.QueryClaimsRequest{PotId: id}, &cr, h) == nil, "unavailable")
	known(len(cr.Claims) <= 1, "unsupported_state")
	var claim any
	if len(cr.Claims) == 1 {
		c := cr.Claims[0]
		known(c != nil && c.PotId == id && c.Claimant == a && c.ClaimedAt > 0 && c.ClaimedAt <= uint64(h), "unsupported_state")
		decimal(c.Amount, 256, true)
		claim = object{"pot_id": c.PotId, "claimant": c.Claimant, "amount_uzrn": c.Amount, "claimed_at": strconv.FormatUint(c.ClaimedAt, 10)}
	}
	return pot, claim
}
func unknownObservation(profile object, reason string) object {
	return object{"protocol": "agent-wallet-zerone.seed-observation/0.1", "status": "unknown", "profile_id": profile["profile_id"], "reason": reason}
}
func recoverNode(result *object, profile object) {
	if e := recover(); e != nil {
		if n, ok := e.(nodeFailure); ok {
			*result = unknownObservation(profile, string(n))
		} else if _, ok := e.(failure); ok {
			*result = unknownObservation(profile, "invalid_response")
		} else {
			panic(e)
		}
	}
}
func (n *nodeClient) inspect(height any, trust trustedArtifacts) (result object) {
	defer recoverNode(&result, n.profile)
	n.checkGenesis(trust)
	b := n.start(height)
	h := b.Block.Height
	pot, claim := n.potAndClaim(h)
	acc := n.account(h)
	allowance := n.allowance(h)
	var params pottypes.QueryParamsResponse
	known(n.query("/zerone.claiming_pot.v1.Query/QueryParams", &pottypes.QueryParamsRequest{}, &params, h) == nil && params.Params != nil, "unavailable")
	decimal(params.Params.MinClaimAmount, 256, false)
	sponsor := rawAccount(str(n.policy, "sponsor_account"), str(n.profile, "chain_id"))
	var balance banktypes.QueryBalanceResponse
	known(n.query("/cosmos.bank.v1beta1.Query/Balance", &banktypes.QueryBalanceRequest{Address: sponsor, Denom: "uzrn"}, &balance, h) == nil && balance.Balance != nil && balance.Balance.Denom == "uzrn", "unavailable")
	decimal(balance.Balance.Amount.String(), 256, false)
	var supply vesttypes.QuerySupplyCouplingAuditResponse
	known(n.query("/zerone.vesting_rewards.v1.Query/SupplyCouplingAudit", &vesttypes.QuerySupplyCouplingAuditRequest{}, &supply, h) == nil, "unavailable")
	for _, s := range []string{supply.TotalMinted, supply.CurrentSupply, supply.MaxSupply} {
		decimal(s, 256, false)
	}
	evidence := n.finish(b)
	return object{"protocol": "agent-wallet-zerone.seed-observation/0.1", "status": "observed", "evidence": evidence, "claimant": acc, "sponsor_account": n.policy["sponsor_account"], "sponsor_balance_uzrn": balance.Balance.Amount.String(), "pot": pot, "prior_claim": claim, "min_claim_amount_uzrn": params.Params.MinClaimAmount, "allowance": allowance, "supply": object{"total_minted_uzrn": supply.TotalMinted, "current_supply_uzrn": supply.CurrentSupply, "max_supply_uzrn": supply.MaxSupply}}
}
func (n *nodeClient) emptyMempool() {
	var r rpctypes.ResultUnconfirmedTxs
	known(n.rpc("num_unconfirmed_txs", object{}, &r) == nil && r.Count == 0 && r.Total == 0, "unsupported_state")
}
func (n *nodeClient) simulate(r object, trust trustedArtifacts) object {
	n.checkGenesis(trust)
	b := n.start(r["height"])
	h := b.Block.Height
	// Native Simulate uses checkState, not the query context. Refuse historical
	// claims and nonempty mempools; require unchanged latest height around the call.
	current := n.status()
	known(current.SyncInfo.LatestBlockHeight == h && bytes.Equal(current.SyncInfo.LatestBlockHash, b.BlockID.Hash), "incoherent_height")
	n.emptyMempool()
	plan := obj(r["plan"])
	var res txtypes.SimulateResponse
	known(n.query("/cosmos.tx.v1beta1.Service/Simulate", &txtypes.SimulateRequest{TxBytes: unb64(str(plan, "simulation_tx_bytes_b64u"))}, &res, h) == nil, "unavailable")
	known(res.GasInfo != nil, "invalid_response")
	n.emptyMempool()
	evidence := n.finish(b)
	known(str(evidence, "latest_height") == strconv.FormatInt(h, 10), "incoherent_height")
	return object{"status": "succeeded", "plan_id": plan["plan_id"], "simulation_tx_bytes_hash": plan["simulation_tx_bytes_hash"], "evidence": evidence, "code": 0, "gas_wanted": strconv.FormatUint(res.GasInfo.GasWanted, 10), "gas_used": strconv.FormatUint(res.GasInfo.GasUsed, 10)}
}
func (n *nodeClient) submit(r object, trust trustedArtifacts) (result object) {
	plan := obj(r["plan"])
	b := readPrivate(n.ctx, str(r, "signed_tx_path"), maxProto)
	summary := signedSummary(plan, b)
	hash := str(summary, "tx_hash")
	require(hash == str(r, "expected_tx_hash"), "signature_invalid")
	n.checkGenesis(trust)
	latest := n.status()
	known(uint64(latest.SyncInfo.LatestBlockHeight) < uint64s(str(n.policy, "timeout_height"), true), "stale")
	known(budgetError(n.ctx) == nil, "unavailable")
	require(n.now().Before(timestamp(str(obj(plan["commitment"]), "expires_at"))), "plan_mismatch")
	// Every failure after entering the single RPC transport is ambiguous, including
	// a conclusive CheckTx rejection. No second call, retry, or refund exists here.
	result = object{"status": "submission_unknown", "tx_hash": hash}
	defer func() {
		if recover() != nil {
			result = object{"status": "submission_unknown", "tx_hash": hash}
		}
	}()
	var res rpctypes.ResultBroadcastTx
	if n.rpc("broadcast_tx_sync", object{"tx": base64.StdEncoding.EncodeToString(b)}, &res) == nil && res.Code == 0 && bytes.Equal(res.Hash, txHashBytes(hash)) {
		known(budgetError(n.ctx) == nil, "unavailable")
		result = object{"status": "accepted", "tx_hash": hash}
	}
	return
}
func (n *nodeClient) lookup(r object, trust trustedArtifacts) (result object) {
	hash := str(r, "tx_hash")
	result = object{"status": "unknown", "tx_hash": hash, "reason": "unavailable"}
	defer func() {
		if e := recover(); e != nil {
			reason := "invalid_response"
			if f, ok := e.(nodeFailure); ok {
				reason = string(f)
			}
			result = object{"status": "unknown", "tx_hash": hash, "reason": reason}
		}
	}()
	n.checkGenesis(trust)
	latest := n.start(nil)
	var tx rpctypes.ResultTx
	rpcErr := n.rpc("tx", object{"hash": base64.StdEncoding.EncodeToString(txHashBytes(hash)), "prove": false}, &tx)
	if rpcErr != nil {
		// Exact Comet source absence contract only. A disabled index, pruned lookup,
		// arbitrary provider error or unavailable block never becomes absence.
		if rpcErr.Code == -32603 && rpcErr.Message == "Internal error" && rpcErr.Data == fmt.Sprintf("tx (%s) not found", hash) {
			return object{"status": "absent", "tx_hash": hash, "evidence": n.finish(latest)}
		}
		panic(nodeFailure("not_found_unproven"))
	}
	summary := signedSummary(obj(r["plan"]), tx.Tx)
	known(str(summary, "tx_hash") == hash && bytes.Equal(tx.Hash, txHashBytes(hash)) && tx.Height > 0 && tx.Height < latest.Block.Height, "invalid_response")
	inclusion := n.block(tx.Height)
	successor := n.block(tx.Height + 1)
	known(bytes.Equal(successor.Block.LastBlockID.Hash, inclusion.BlockID.Hash) && int(tx.Index) < len(inclusion.Block.Txs) && bytes.Equal(inclusion.Block.Txs[tx.Index], tx.Tx), "incoherent_height")
	// Results are committed by the next canonical header, not merely the tx index.
	var results rpctypes.ResultBlockResults
	// This one-height native result read is bounded by the same response ceiling.
	known(n.blockResults(tx.Height, &results) == nil && results.Height == tx.Height && int(tx.Index) < len(results.TxsResults), "unavailable")
	native := results.TxsResults[tx.Index]
	known(native != nil, "invalid_response")
	known(bytes.Equal(cmttypes.NewResults(results.TxsResults).Hash(), successor.Block.LastResultsHash), "incoherent_height")
	known(native.Code == tx.TxResult.Code && native.GasUsed == tx.TxResult.GasUsed && native.GasUsed >= 0, "invalid_response")
	_, claim := n.potAndClaim(tx.Height)
	var credit any
	if native.Code == 0 && claim != nil {
		rec := obj(claim)
		count := 0
		valid := false
		for _, event := range native.Events {
			if event.Type == "zerone.claiming_pot.pot_claimed" {
				count++
				attrs := map[string]string{}
				duplicate := false
				for _, a := range event.Attributes {
					if _, ok := attrs[a.Key]; ok {
						duplicate = true
					}
					attrs[a.Key] = a.Value
				}
				valid = !duplicate && attrs["pot_id"] == str(n.policy, "pot_id") && attrs["claimant"] == rawAccount(str(n.policy, "claimant_account"), str(n.profile, "chain_id")) && attrs["amount"] == str(rec, "amount_uzrn") && str(rec, "claimed_at") == strconv.FormatInt(tx.Height, 10)
			}
		}
		if count == 1 && valid {
			credit = rec["amount_uzrn"]
		}
	}
	acc := n.account(latest.Block.Height)
	known(str(acc, "status") == "found", "unsupported_state")
	allowance := n.allowance(latest.Block.Height)
	// Recheck both inclusion and successor before returning positive confirmation.
	again := n.block(tx.Height)
	known(bytes.Equal(again.BlockID.Hash, inclusion.BlockID.Hash), "incoherent_height")
	againSuccessor := n.block(tx.Height + 1)
	known(bytes.Equal(againSuccessor.BlockID.Hash, successor.BlockID.Hash), "incoherent_height")
	evidence := n.finish(latest)
	return object{"status": "included", "tx_hash": hash, "inclusion": blockAnchor(inclusion), "evidence": evidence, "code": native.Code, "gas_used": strconv.FormatInt(native.GasUsed, 10), "credited_amount_uzrn": credit, "claimant_sequence": acc["sequence"], "allowance": allowance}
}
func (n *nodeClient) blockResults(height int64, out *rpctypes.ResultBlockResults) *rpcFailure {
	return n.rpc("block_results", object{"height": strconv.FormatInt(height, 10)}, out)
}
