package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/rootmulti"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/merkle"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	cmthttp "github.com/cometbft/cometbft/rpc/client/http"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/protobuf/proto"

	schedule "github.com/zerone-chain/zerone/x/schedule/types"
)

const schedulerExpectationSchema = "zerone.local-scheduler-expectation/v1"

type dueExpectation struct {
	ScheduleID string `json:"schedule_id"`
	Height     uint64 `json:"height"`
}

type balanceExpectation struct {
	Address string `json:"address"`
	Amount  string `json:"amount_uzrn"`
}

// This is a list of known fixture records, not a /subspace completeness claim.
// State and genesis use the module's Go JSON encoding (numeric enums/integers).
// The genesis writer also retains explicit admission=false. Occurrences are
// never transaction hashes.
type schedulerExpectation struct {
	Schema            string                       `json:"schema"`
	ChainID           string                       `json:"chain_id"`
	StateHeight       int64                        `json:"state_height"`
	State             *schedule.GenesisState       `json:"state"`
	AbsentDue         []dueExpectation             `json:"absent_due"`
	AbsentOccurrences []*schedule.ExecutionReceipt `json:"absent_occurrences"`
	Balances          []balanceExpectation         `json:"balances"`
	RequireEmptyBlock bool                         `json:"require_empty_block"`
}

type stateKeyExpectation struct {
	Store      string
	Key, Value []byte
	Absent     bool
}

type stateKeyProof struct {
	Store       string              `json:"store"`
	KeyHex      string              `json:"key_hex"`
	Value       []byte              `json:"value,omitempty"`
	ValueSHA256 string              `json:"value_sha256,omitempty"`
	Absent      bool                `json:"absent"`
	ProofOps    *cmtcrypto.ProofOps `json:"proof_ops"`
}

type schedulerNodeReport struct {
	RPC    string          `json:"rpc"`
	Proofs []stateKeyProof `json:"proofs"`
}

type schedulerReport struct {
	StateHeight        int64                 `json:"state_height"`
	AnchorHeight       int64                 `json:"anchor_height"`
	ExpectationSHA256  string                `json:"expectation_sha256"`
	Scope              string                `json:"scope"`
	EmptyOrdinaryBlock bool                  `json:"empty_ordinary_block"`
	Nodes              []schedulerNodeReport `json:"nodes"`
}

func loadSchedulerExpectation(path, chainID string) (*schedulerExpectation, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	bz, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil {
		return nil, "", err
	}
	if len(bz) > 1<<20 {
		return nil, "", fmt.Errorf("scheduler expectation exceeds 1 MiB")
	}
	var e schedulerExpectation
	dec := json.NewDecoder(bytes.NewReader(bz))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return nil, "", err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, "", fmt.Errorf("trailing scheduler expectation data")
	}
	if _, err := schedulerKeys(&e, chainID); err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(bz)
	return &e, hex.EncodeToString(digest[:]), nil
}

func schedulerAnchorHeight(e *schedulerExpectation, requested, minimum int64) (int64, error) {
	if e.StateHeight < 1 || e.StateHeight > math.MaxInt64-2 {
		return 0, fmt.Errorf("scheduler state height is outside the proofable range")
	}
	anchor := e.StateHeight + 1
	if requested != 0 && requested != anchor {
		return 0, fmt.Errorf("scheduler proof at version %d requires header %d, not %d", e.StateHeight, anchor, requested)
	}
	if minimum < e.StateHeight+2 {
		return 0, fmt.Errorf("scheduler proof requires all nodes at height %d; minimum is %d", e.StateHeight+2, minimum)
	}
	return anchor, nil
}

func schedulerKeys(e *schedulerExpectation, chainID string) ([]stateKeyExpectation, error) {
	if e == nil || e.Schema != schedulerExpectationSchema {
		return nil, fmt.Errorf("invalid scheduler expectation schema")
	}
	if err := verifyExpectedChainID(chainID, e.ChainID); err != nil {
		return nil, err
	}
	if e.StateHeight < 1 || e.StateHeight > math.MaxInt64-2 {
		return nil, fmt.Errorf("invalid scheduler state height")
	}
	if e.State == nil || e.State.Params == nil || e.State.Params.AcceptNewSchedules {
		return nil, fmt.Errorf("scheduler expectations require explicit closed admission")
	}
	if len(e.State.Schedules) > 32 || len(e.State.Receipts) > 128 || len(e.AbsentDue) > 128 || len(e.AbsentOccurrences) > 128 || len(e.Balances) > 16 {
		return nil, fmt.Errorf("scheduler fixture exceeds bounded record limits")
	}
	if err := e.State.ValidateForChainID(chainID); err != nil {
		return nil, fmt.Errorf("scheduler expectation: %w", err)
	}
	var keys []stateKeyExpectation
	seen := map[string]bool{}
	add := func(store string, key, value []byte, absent bool) error {
		identity := store + ":" + hex.EncodeToString(key)
		if seen[identity] {
			return fmt.Errorf("duplicate or contradictory expected key %s", identity)
		}
		seen[identity] = true
		keys = append(keys, stateKeyExpectation{store, key, value, absent})
		return nil
	}
	put := func(key, value []byte) error { return add(schedule.StoreKey, key, value, false) }
	putProto := func(key []byte, m proto.Message) error {
		bz, err := proto.Marshal(m)
		if err != nil {
			return err
		}
		return put(key, bz)
	}
	if err := putProto(schedule.ParamsKey, e.State.Params); err != nil {
		return nil, err
	}
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, e.State.NextScheduleId)
	if err := put(schedule.ScheduleCounterKey, counter); err != nil {
		return nil, err
	}
	if err := put(schedule.TotalEscrowKey, []byte(e.State.TotalEscrowUzrn)); err != nil {
		return nil, err
	}
	for _, s := range e.State.Schedules {
		if s.UpdatedHeight > uint64(e.StateHeight) {
			return nil, fmt.Errorf("schedule expectation is newer than state height")
		}
		if err := putProto(schedule.ScheduleKey(s.Id), s); err != nil {
			return nil, err
		}
		creator, err := sdk.AccAddressFromBech32(s.Creator)
		if err != nil {
			return nil, err
		}
		if err := put(schedule.CreatorKey(creator, s.Id), []byte{1}); err != nil {
			return nil, err
		}
		active := s.Status == schedule.ScheduleStatus_SCHEDULE_STATUS_ACTIVE
		if err := add(schedule.StoreKey, schedule.ActiveCreatorKey(creator, s.Id), []byte{1}, !active); err != nil {
			return nil, err
		}
		if active {
			if err := put(schedule.DueKey(s.NextExecutionHeight, s.Id), []byte{1}); err != nil {
				return nil, err
			}
		}
		// The next sequence must be absent even for terminal records: a duplicate
		// execution cannot hide behind an unchanged expected receipt.
		if err := add(schedule.StoreKey, schedule.ReceiptKey(s.Id, s.ExecutionCount+1), nil, true); err != nil {
			return nil, err
		}
	}
	for _, r := range e.State.Receipts {
		if r.ExecutedHeight > uint64(e.StateHeight) {
			return nil, fmt.Errorf("receipt expectation is newer than state height")
		}
		key := schedule.ReceiptKey(r.ScheduleId, r.Sequence)
		if err := putProto(key, r); err != nil {
			return nil, err
		}
		if err := put(schedule.OccurrenceKey(r.OccurrenceId), key); err != nil {
			return nil, err
		}
	}
	for _, d := range e.AbsentDue {
		if _, err := schedule.ParseScheduleID(d.ScheduleID); err != nil {
			return nil, err
		}
		if d.Height == 0 || d.Height > schedule.MaxSDKBlockHeight {
			return nil, fmt.Errorf("invalid absent due height")
		}
		if err := add(schedule.StoreKey, schedule.DueKey(d.Height, d.ScheduleID), nil, true); err != nil {
			return nil, err
		}
	}
	for _, r := range e.AbsentOccurrences {
		if err := schedule.ValidateReceipt(r); err != nil {
			return nil, err
		}
		if r.OccurrenceId != schedule.OccurrenceID(chainID, r.ScheduleId, r.Revision, r.Sequence, r.DueHeight) {
			return nil, fmt.Errorf("absent occurrence expectation has wrong chain-bound occurrence ID")
		}
		if err := add(schedule.StoreKey, schedule.OccurrenceKey(r.OccurrenceId), nil, true); err != nil {
			return nil, err
		}
	}
	for _, b := range e.Balances {
		address, err := sdk.AccAddressFromBech32(b.Address)
		if err != nil {
			return nil, err
		}
		amount, err := schedule.ParseNonNegativeAmount(b.Amount)
		if err != nil {
			return nil, err
		}
		pair := collections.Join(address, schedule.Denom)
		kc := collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey)
		key := make([]byte, kc.Size(pair))
		if _, err := kc.Encode(key, pair); err != nil {
			return nil, err
		}
		key = append(bytes.Clone(banktypes.BalancesPrefix.Bytes()), key...)
		value, err := banktypes.BalanceValueCodec.Encode(sdkmath.NewIntFromBigInt(amount))
		if err != nil {
			return nil, err
		}
		if err := add(banktypes.StoreKey, key, value, amount.Sign() == 0); err != nil {
			return nil, err
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Store != keys[j].Store {
			return keys[i].Store < keys[j].Store
		}
		return bytes.Compare(keys[i].Key, keys[j].Key) < 0
	})
	return keys, nil
}

func verifyStateProof(response abci.ResponseQuery, expected stateKeyExpectation, stateHeight int64, root []byte) (err error) {
	// Malformed remote proof operators must be a verification failure, never an
	// unchecked panic or an absence result. The SDK runtime handles both ICS23 layers.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("malformed state proof: %v", r)
		}
	}()
	if stateHeight < 1 || response.Height != stateHeight {
		return fmt.Errorf("state proof height %d, expected %d", response.Height, stateHeight)
	}
	if response.Code != 0 {
		return fmt.Errorf("state query failed with code %d", response.Code)
	}
	if len(root) != sha256.Size {
		return fmt.Errorf("invalid state root length")
	}
	if !bytes.Equal(response.Key, expected.Key) {
		return fmt.Errorf("state proof key mismatch")
	}
	if response.ProofOps == nil || len(response.ProofOps.Ops) == 0 {
		return fmt.Errorf("state proof is missing")
	}
	path := merkle.KeyPath{}.AppendKey([]byte(expected.Store), merkle.KeyEncodingURL).AppendKey(expected.Key, merkle.KeyEncodingHex).String()
	runtime := rootmulti.DefaultProofRuntime()
	if expected.Absent {
		if len(response.Value) != 0 {
			return fmt.Errorf("expected absent key has a value")
		}
		return runtime.VerifyAbsence(response.ProofOps, root, path)
	}
	if !bytes.Equal(response.Value, expected.Value) {
		return fmt.Errorf("state proof value differs from exact expectation")
	}
	return runtime.VerifyValue(response.ProofOps, root, path, expected.Value)
}

func verifySchedulerNode(client *cmthttp.HTTP, endpoint string, e *schedulerExpectation, keys []stateKeyExpectation, anchor *cmttypes.Block) (schedulerNodeReport, error) {
	result := schedulerNodeReport{RPC: endpoint}
	if anchor == nil || anchor.Height != e.StateHeight+1 || anchor.ChainID != e.ChainID {
		return result, fmt.Errorf("scheduler anchor height/chain mismatch")
	}
	if e.RequireEmptyBlock {
		ctx, cancel := rpcContext()
		block, err := client.Block(ctx, &e.StateHeight)
		cancel()
		if err != nil {
			return result, err
		}
		if block.Block == nil || block.Block.Height != e.StateHeight || block.Block.ChainID != e.ChainID || !block.BlockID.Equals(anchor.LastBlockID) {
			return result, fmt.Errorf("empty block is not linked to verified scheduler anchor")
		}
		if err := block.Block.ValidateBasic(); err != nil {
			return result, err
		}
		if err := verifyBlockHashBinding(block.BlockID, block.Block); err != nil {
			return result, err
		}
		if len(block.Block.Data.Txs) != 0 {
			return result, fmt.Errorf("scheduler occurrence block %d contains ordinary transactions", e.StateHeight)
		}
	}
	for _, key := range keys {
		ctx, cancel := rpcContext()
		response, err := client.ABCIQueryWithOptions(ctx, "/store/"+key.Store+"/key", key.Key, rpcclient.ABCIQueryOptions{Height: e.StateHeight, Prove: true})
		cancel()
		if err != nil {
			return result, err
		}
		if err := verifyStateProof(response.Response, key, e.StateHeight, anchor.AppHash); err != nil {
			return result, fmt.Errorf("%s key %x: %w", key.Store, key.Key, err)
		}
		p := stateKeyProof{Store: key.Store, KeyHex: hex.EncodeToString(key.Key), Absent: key.Absent, ProofOps: response.Response.ProofOps}
		if !key.Absent {
			p.Value = response.Response.Value
			digest := sha256.Sum256(p.Value)
			p.ValueSHA256 = hex.EncodeToString(digest[:])
		}
		result.Proofs = append(result.Proofs, p)
	}
	return result, nil
}
