package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/pelletier/go-toml/v2"
)

const observerSchema = "zerone.local-observer-checkpoint/v1"

// Only public expectations belong here. The home is an explicit CLI argument,
// never a path followed from a report. H is a sampled applied checkpoint, not a
// claim about the exact stop/Commit boundary; header H+1 authenticates its root.
type observerExpectation struct {
	Schema            string `json:"schema"`
	Phase             string `json:"phase"`
	ChainID           string `json:"chain_id"`
	RPC               string `json:"rpc"`
	NodeID            string `json:"node_id"`
	ConsensusPubKey   []byte `json:"consensus_public_key"`
	GenesisSHA256     string `json:"genesis_sha256"`
	BinarySHA256      string `json:"binary_sha256"`
	HistoryBeforeJoin int64  `json:"history_before_join"`
	StateHeight       int64  `json:"state_height"`
	AppHash           string `json:"app_hash"`
}

type observerReport struct {
	observerExpectation
	ConsensusAddress string        `json:"consensus_address"`
	InitialHeight    int64         `json:"initial_height"`
	AppliedHeight    int64         `json:"observed_applied_height"`
	AppliedAppHash   string        `json:"observed_applied_app_hash"`
	VotingPower      int64         `json:"voting_power"`
	LastSignHeight   int64         `json:"last_sign_height"`
	CatchingUp       bool          `json:"catching_up"`
	AccountKeyring   bool          `json:"account_keyring_present"`
	StateSync        bool          `json:"state_sync_enabled"`
	CheckpointRecord string        `json:"checkpoint_record"`
	CheckpointProof  stateKeyProof `json:"checkpoint_proof"`
	Scope            string        `json:"scope"`
}

func observerAddress(e *observerExpectation) []byte {
	return ed25519.PubKey(e.ConsensusPubKey).Address()
}

func observerDigest(value string, size int) bool {
	bz, err := hex.DecodeString(value)
	return err == nil && len(bz) == size
}

// This local helper reads only named public inputs/signing state, never either
// Comet private-key file. Refuse links and non-regular files before opening.
func readObserverFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("observer input is missing or not a regular file: %s", filepath.Base(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("observer input changed while opening")
	}
	bz, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(bz)) > limit {
		return nil, fmt.Errorf("observer input exceeds limit or could not be read")
	}
	return bz, nil
}

func decodeObserverJSON(bz []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(bz))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		// Do not include untrusted JSON values (especially signing-state data).
		return fmt.Errorf("invalid observer JSON")
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing observer JSON data")
	}
	return nil
}

func loadObserverExpectation(path, chainID string) (*observerExpectation, error) {
	bz, err := readObserverFile(path, 16<<10)
	if err != nil {
		return nil, err
	}
	var e observerExpectation
	if err := decodeObserverJSON(bz, &e); err != nil {
		return nil, err
	}
	if err := validateObserverExpectation(&e, chainID); err != nil {
		return nil, err
	}
	return &e, nil
}

func validateObserverExpectation(e *observerExpectation, chainID string) error {
	if e.Schema != observerSchema || (e.Phase != "replay" && e.Phase != "restart") {
		return fmt.Errorf("invalid observer schema/phase")
	}
	if err := verifyExpectedChainID(chainID, e.ChainID); err != nil {
		return err
	}
	if len(e.ConsensusPubKey) != ed25519.PubKeySize || !observerDigest(e.NodeID, 20) ||
		!observerDigest(e.GenesisSHA256, 32) || !observerDigest(e.BinarySHA256, 32) || !observerDigest(e.AppHash, 32) {
		return fmt.Errorf("invalid observer identity or digest")
	}
	if e.HistoryBeforeJoin < 1 || e.StateHeight < e.HistoryBeforeJoin || e.StateHeight > math.MaxInt64-2 {
		return fmt.Errorf("observer checkpoint must cover pre-join history and allow H+2")
	}
	return nil
}

func verifyObserverZeroState(bz []byte) error {
	var state struct {
		Height    string `json:"height"`
		Round     *int64 `json:"round"`
		Step      *int64 `json:"step"`
		Signature []byte `json:"signature"`
		SignBytes []byte `json:"signbytes"`
	}
	if err := decodeObserverJSON(bz, &state); err != nil {
		return err
	}
	if state.Height != "0" || state.Round == nil || *state.Round != 0 || state.Step == nil || *state.Step != 0 || len(state.Signature) != 0 || len(state.SignBytes) != 0 {
		return fmt.Errorf("observer signing state is not pristine height/round/step zero")
	}
	return nil
}

// Genesis for this drill is constructed by four SDK gentxs. Check that source
// of membership as well as any explicit Comet genesis validators. Do not infer
// absence from an empty top-level validators array (gentxs populate InitChain).
func verifyObserverGenesis(bz []byte, e *observerExpectation) (map[string]bool, int64, error) {
	digest := sha256.Sum256(bz)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), e.GenesisSHA256) {
		return nil, 0, fmt.Errorf("observer genesis digest mismatch")
	}
	genesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(bz))
	if err != nil || genesis.Consensus == nil {
		return nil, 0, fmt.Errorf("invalid observer public genesis")
	}
	if err := verifyExpectedChainID(e.ChainID, genesis.ChainID); err != nil {
		return nil, 0, err
	}
	initial := genesis.InitialHeight
	if initial == 0 {
		initial = 1
	}
	if initial < 1 || e.HistoryBeforeJoin <= initial {
		return nil, 0, fmt.Errorf("observer requires history after genesis initial height")
	}
	var app struct {
		Bank struct {
			Params *banktypes.Params `json:"params"`
		} `json:"bank"`
		Genutil struct {
			Txs []struct {
				Body struct {
					Messages []struct {
						Type   string `json:"@type"`
						PubKey struct {
							Type string `json:"@type"`
							Key  []byte `json:"key"`
						} `json:"pubkey"`
					} `json:"messages"`
				} `json:"body"`
			} `json:"gen_txs"`
		} `json:"genutil"`
	}
	if err := json.Unmarshal(genesis.AppState, &app); err != nil || len(app.Genutil.Txs) != 4 {
		return nil, 0, fmt.Errorf("observer drill requires exactly four genesis gentxs")
	}
	// SDK bank InitGenesis calls SetParams, which removes deprecated SendEnabled
	// entries and persists Params at 0x05. This fixture must explicitly enable
	// default sends: its protobuf bool then guarantees a nonempty initial anchor.
	// Do not silently select another record for incompatible genesis settings.
	if app.Bank.Params == nil || !app.Bank.Params.DefaultSendEnabled || app.Bank.Params.Validate() != nil {
		return nil, 0, fmt.Errorf("observer requires bank genesis params with default_send_enabled=true for a nonempty bank/05 membership anchor")
	}
	members := map[string]bool{}
	for _, tx := range app.Genutil.Txs {
		if len(tx.Body.Messages) != 1 {
			return nil, 0, fmt.Errorf("unexpected genesis gentx messages")
		}
		msg := tx.Body.Messages[0]
		if msg.Type != "/cosmos.staking.v1beta1.MsgCreateValidator" || msg.PubKey.Type != "/cosmos.crypto.ed25519.PubKey" || len(msg.PubKey.Key) != ed25519.PubKeySize {
			return nil, 0, fmt.Errorf("unexpected genesis validator public key")
		}
		address := ed25519.PubKey(msg.PubKey.Key).Address()
		if bytes.Equal(address, observerAddress(e)) || members[string(address)] {
			return nil, 0, fmt.Errorf("observer is in genesis validator set or duplicate genesis identity")
		}
		members[string(address)] = true
	}
	for _, validator := range genesis.Consensus.Validators {
		if validator.PubKey == nil || bytes.Equal(validator.PubKey.Address(), observerAddress(e)) || !members[string(validator.PubKey.Address())] {
			return nil, 0, fmt.Errorf("unexpected explicit genesis consensus member")
		}
	}
	return members, initial, nil
}

func verifyObserverHome(home string, e *observerExpectation) (map[string]bool, int64, error) {
	for _, sub := range []string{"", "config", "data"} {
		info, err := os.Lstat(filepath.Join(home, sub))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, 0, fmt.Errorf("observer home requires real home/config/data directories")
		}
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, 0, err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "keyring") {
			return nil, 0, fmt.Errorf("observer must not contain an account keyring")
		}
	}
	state, err := readObserverFile(filepath.Join(home, "data", "priv_validator_state.json"), 4096)
	if err != nil {
		return nil, 0, err
	}
	if err := verifyObserverZeroState(state); err != nil {
		return nil, 0, err
	}
	config, err := readObserverFile(filepath.Join(home, "config", "config.toml"), 128<<10)
	if err != nil {
		return nil, 0, err
	}
	var cfg struct {
		Signer *string `toml:"priv_validator_laddr"`
		RPC    struct {
			Unsafe *bool   `toml:"unsafe"`
			Pprof  *string `toml:"pprof_laddr"`
		} `toml:"rpc"`
		StateSync struct {
			Enable *bool `toml:"enable"`
		} `toml:"statesync"`
		Instrumentation struct {
			Prometheus *bool `toml:"prometheus"`
		} `toml:"instrumentation"`
	}
	if err := toml.Unmarshal(config, &cfg); err != nil || cfg.Signer == nil || *cfg.Signer != "" || cfg.RPC.Unsafe == nil || *cfg.RPC.Unsafe || cfg.RPC.Pprof == nil || *cfg.RPC.Pprof != "" || cfg.StateSync.Enable == nil || *cfg.StateSync.Enable || cfg.Instrumentation.Prometheus == nil || *cfg.Instrumentation.Prometheus {
		return nil, 0, fmt.Errorf("observer config must explicitly disable state sync, remote signing, unsafe RPC, profiling and metrics")
	}
	genesis, err := readObserverFile(filepath.Join(home, "config", "genesis.json"), 16<<20)
	if err != nil {
		return nil, 0, err
	}
	return verifyObserverGenesis(genesis, e)
}

func verifyObserverFreshHome(home string) error {
	entries, err := os.ReadDir(filepath.Join(home, "data"))
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != "priv_validator_state.json" {
		return fmt.Errorf("fresh observer data must contain only initial zero signing state; database copying is forbidden")
	}
	return nil
}

func verifyObserverStatuses(e *observerExpectation, rpcs []string, statuses []*coretypes.ResultStatus, members map[string]bool) (int, error) {
	if len(rpcs) != 5 || len(statuses) != 5 {
		return -1, fmt.Errorf("observer verification requires five endpoints")
	}
	seenNodes, seenValidators := map[string]bool{}, map[string]bool{}
	observerIndex := -1
	for i, status := range statuses {
		if status == nil {
			return -1, fmt.Errorf("missing node status")
		}
		if err := verifyExpectedChainID(e.ChainID, status.NodeInfo.Network); err != nil {
			return -1, err
		}
		id := string(status.NodeInfo.ID())
		if !observerDigest(id, 20) || seenNodes[id] {
			return -1, fmt.Errorf("duplicate or invalid node identity")
		}
		seenNodes[id] = true
		v := status.ValidatorInfo
		if v.PubKey == nil || !bytes.Equal(v.PubKey.Address(), v.Address) {
			return -1, fmt.Errorf("status consensus identity mismatch")
		}
		if rpcs[i] == e.RPC {
			observerIndex = i
			if id != e.NodeID || !bytes.Equal(v.PubKey.Bytes(), e.ConsensusPubKey) || !bytes.Equal(v.Address, observerAddress(e)) || v.VotingPower != 0 {
				return -1, fmt.Errorf("observer identity/restart mismatch or nonzero voting power")
			}
			if status.SyncInfo.CatchingUp || status.SyncInfo.LatestBlockHeight < e.StateHeight+2 {
				return -1, fmt.Errorf("observer has not caught up past checkpoint H+1")
			}
		} else {
			address := string(v.Address)
			if !members[address] || seenValidators[address] || v.VotingPower <= 0 {
				return -1, fmt.Errorf("validator endpoints do not identify four distinct genesis members")
			}
			seenValidators[address] = true
		}
	}
	if observerIndex < 0 || len(seenValidators) != 4 {
		return -1, fmt.Errorf("expected observer endpoint is missing")
	}
	return observerIndex, nil
}

func verifyObserverSet(e *observerExpectation, members map[string]bool, validators *coretypes.ResultValidators, height int64) error {
	if validators == nil || validators.BlockHeight != height || validators.Total != 4 || len(validators.Validators) != 4 {
		return fmt.Errorf("observer requires an exact height-pinned four-validator set")
	}
	seen := map[string]bool{}
	for _, validator := range validators.Validators {
		if validator == nil || validator.PubKey == nil || !bytes.Equal(validator.PubKey.Address(), validator.Address) || validator.VotingPower <= 0 {
			return fmt.Errorf("invalid height-pinned validator identity/power")
		}
		address := string(validator.Address)
		if bytes.Equal(validator.Address, observerAddress(e)) || !members[address] || seen[address] {
			return fmt.Errorf("observer is included or height-pinned membership differs from genesis")
		}
		seen[address] = true
	}
	return nil
}

func verifyObserverAnchor(e *observerExpectation, block *cmttypes.Block) error {
	if block == nil || block.ChainID != e.ChainID || block.Height != e.StateHeight+1 {
		return fmt.Errorf("observer checkpoint requires its chain's header H+1")
	}
	root, _ := hex.DecodeString(e.AppHash)
	if !bytes.Equal(root, block.AppHash) {
		return fmt.Errorf("observer applied checkpoint AppHash differs from header H+1 pre-state AppHash")
	}
	return nil
}

type observerCheckpointClient interface {
	ABCIInfo(context.Context) (*coretypes.ResultABCIInfo, error)
	ABCIQueryWithOptions(context.Context, string, cmtbytes.HexBytes, rpcclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error)
}

func verifyObserverCheckpoint(client observerCheckpointClient, e *observerExpectation, initial int64) (*observerReport, error) {
	ctx, cancel := rpcContext()
	info, err := client.ABCIInfo(ctx)
	cancel()
	if err != nil {
		return nil, err
	}
	if info == nil || info.Response.LastBlockHeight < e.StateHeight || (e.Phase == "restart" && info.Response.LastBlockHeight <= e.StateHeight) || len(info.Response.LastBlockAppHash) != 32 {
		return nil, fmt.Errorf("observer applied height/root does not cover checkpoint after replay/restart")
	}
	// Anchor the initialized SDK bank Params record, not an arbitrary absent key.
	// Empty-valued bank indexes can make ICS23 nonmembership proofs unverifiable;
	// their acceptance is deliberately unchanged. This exact nonempty membership
	// witness ties retained version H to its already quorum-authenticated H+1 root.
	key := banktypes.ParamsKey.Bytes()
	ctx, cancel = rpcContext()
	response, err := client.ABCIQueryWithOptions(ctx, "/store/bank/key", key, rpcclient.ABCIQueryOptions{Height: e.StateHeight, Prove: true})
	cancel()
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Response.Value) == 0 {
		return nil, fmt.Errorf("observer bank/05 membership anchor is missing or empty; no fallback proof")
	}
	// This is not a fabricated expected parameter value: authenticate the returned
	// exact canonical bytes under the fixed store/key/height and trusted root.
	var params banktypes.Params
	value := response.Response.Value
	if err := params.Unmarshal(value); err != nil || params.Validate() != nil || len(params.SendEnabled) != 0 {
		return nil, fmt.Errorf("observer bank/05 anchor is not an SDK bank Params record")
	}
	canonical, err := params.Marshal()
	if err != nil || !bytes.Equal(value, canonical) {
		return nil, fmt.Errorf("observer bank/05 anchor has noncanonical parameter bytes")
	}
	root, _ := hex.DecodeString(e.AppHash)
	if err := verifyStateProof(response.Response, stateKeyExpectation{Store: banktypes.StoreKey, Key: key, Value: value}, e.StateHeight, root); err != nil {
		return nil, fmt.Errorf("observer retained checkpoint proof: %w", err)
	}
	digest := sha256.Sum256(value)
	return &observerReport{
		observerExpectation: *e, ConsensusAddress: strings.ToUpper(hex.EncodeToString(observerAddress(e))), InitialHeight: initial,
		AppliedHeight: info.Response.LastBlockHeight, AppliedAppHash: strings.ToUpper(hex.EncodeToString(info.Response.LastBlockAppHash)),
		CheckpointRecord: "cosmos.bank.v1beta1.Params",
		CheckpointProof:  stateKeyProof{Store: banktypes.StoreKey, KeyHex: hex.EncodeToString(key), Value: value, ValueSHA256: hex.EncodeToString(digest[:]), ProofOps: response.Response.ProofOps},
		Scope:            "local public-genesis P2P replay; sampled applied H bound to canonical H+1; retained bank/05 Params membership only, not completeness or accounting; no production join or clean-machine claim",
	}, nil
}
