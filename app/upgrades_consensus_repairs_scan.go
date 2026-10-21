package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Each census has its own budget, charged only for its declared reads. Unrelated
// alignment observations, SDK staking history, bank accounts, etc. are streamed,
// never retained or charged against a census's budget. Exceeding a budget refuses
// without truncation; a stopped-copy measurement is required before activation.
const consensusRepairsMaxRecords = 100_000
const consensusRepairsMaxBytes = 64 << 20

type ConsensusRepairsSize struct {
	Records int `json:"records"`
	Bytes   int `json:"bytes"`
}

type ConsensusRepairsReadSize struct {
	Store  string `json:"store"`
	Prefix string `json:"prefix_hex"`
	Exact  bool   `json:"exact_key"`
	ConsensusRepairsSize
}

type consensusRepairsRead struct {
	store string
	key   []byte
	exact bool
}

func (app *ZeroneApp) consensusRepairsReads() (map[string][]consensusRepairsRead, error) {
	balanceKey := func(module string) ([]byte, error) {
		return collections.EncodeKeyWithPrefix(banktypes.BalancesPrefix.Bytes(), collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey), collections.Join(authtypes.NewModuleAddress(module), "uzrn"))
	}
	qualificationBalance, err := balanceKey("qualification")
	if err != nil {
		return nil, err
	}
	challengeBalance, err := balanceKey("capture_challenge")
	if err != nil {
		return nil, err
	}
	supply, err := collections.EncodeKeyWithPrefix(banktypes.SupplyKey.Bytes(), collections.StringKey, "uzrn")
	if err != nil {
		return nil, err
	}
	return map[string][]consensusRepairsRead{
		"seat_runoffs":           {{"zerone_gov", []byte{0x10}, false}, {"zerone_gov", []byte{0x12}, true}},
		"reformulations":         {{"knowledge", []byte{0x06}, true}, {"knowledge", []byte{0x62}, false}, {"knowledge", []byte{0x63}, false}, {"knowledge", []byte{0x65}, false}},
		"verification_histories": {{"capture_defense", []byte{0x05}, false}},
		"qualification_escrow":   {{"qualification", []byte{0x01}, false}, {"qualification", []byte{0x00}, true}, {"bank", qualificationBalance, true}},
		"challenge_backing":      {{"capture_challenge", []byte{0x01}, false}, {"capture_challenge", []byte{0x02}, false}, {"bank", challengeBalance, true}},
		"alignment_inputs":       {{"vesting_rewards", []byte{0x0b}, true}, {"zerone_staking", []byte{0x01}, false}, {"zerone_ontology", []byte{0x02}, false}, {"bank", supply, true}},
		"halt_reasons":           {{"emergency", []byte{0x02}, false}, {"zerone_gov", []byte{0x01}, false}, {"zerone_gov", []byte{0x06}, false}, {"zerone_gov", []byte{0x0d}, true}, {"zerone_gov", []byte{0x14}, false}},
		"adapter_minima":         {{"substrate_bridge", []byte{0x84}, false}},
		"ibc_refunds":            {{"zerone_ibcrl", []byte{0x01}, false}, {"zerone_ibcrl", []byte{0x02}, false}, {"zerone_ibcrl", []byte{0x03}, false}},
	}, nil
}

// Frame every store/key/value so neither concatenation nor store boundaries can
// alias. The same encoding is used for the source, merged expectation and result.
func consensusRepairsHashRow(h hash.Hash, store string, key, value []byte) {
	var size [8]byte
	for _, field := range [][]byte{[]byte(store), key, value} {
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		h.Write(size[:])
		h.Write(field)
	}
}

// Stream in store/key order, merging only the tiny expected write set. No full
// before/after image is materialised. The visitor must clone bytes it retains.
// The separate 0x87 digest explicitly pins claim-integrity state as well.
func (app *ZeroneApp) consensusRepairsStream(ctx sdk.Context, writes map[string]map[string][]byte, visit func(string, []byte, []byte) error) (digest, claimIntegrity string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("consensus repairs store read fault: %v", r)
		}
	}()
	names := make([]string, 0, len(app.keys))
	for name := range app.keys {
		names = append(names, name)
	}
	for name := range writes {
		if app.keys[name] == nil {
			return "", "", fmt.Errorf("unknown consensus repairs write store %s", name)
		}
	}
	sort.Strings(names)
	full, claim := sha256.New(), sha256.New()
	for _, name := range names {
		emit := func(key, value []byte) error {
			consensusRepairsHashRow(full, name, key, value)
			if name == "knowledge" && bytes.HasPrefix(key, []byte{0x87}) {
				consensusRepairsHashRow(claim, name, key, value)
			}
			if visit != nil {
				return visit(name, key, value)
			}
			return nil
		}
		if err := func() (err error) {
			it := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
			defer func() {
				if closeErr := it.Close(); err == nil {
					err = closeErr
				}
			}()
			keys := make([]string, 0, len(writes[name]))
			for key := range writes[name] {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			next := 0
			for ; it.Valid(); it.Next() {
				// SDK cache iterators report an error for normal exhaustion. Check
				// while valid, not by matching backend-specific error messages.
				if err := it.Error(); err != nil {
					return err
				}
				key := it.Key()
				for next < len(keys) && keys[next] < string(key) {
					if err := emit([]byte(keys[next]), writes[name][keys[next]]); err != nil {
						return err
					}
					next++
				}
				value := it.Value()
				if next < len(keys) && keys[next] == string(key) {
					value = writes[name][keys[next]]
					next++
				}
				if err := emit(key, value); err != nil {
					return err
				}
			}
			for ; next < len(keys); next++ {
				if err := emit([]byte(keys[next]), writes[name][keys[next]]); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return "", "", fmt.Errorf("scan %s: %w", name, err)
		}
	}
	return hex.EncodeToString(full.Sum(nil)), hex.EncodeToString(claim.Sum(nil)), nil
}

func (app *ZeroneApp) consensusRepairsCensusRows(ctx sdk.Context, report *ConsensusRepairsCensus) (map[string]map[string][]byte, error) {
	reads, err := app.consensusRepairsReads()
	if err != nil {
		return nil, err
	}
	return app.consensusRepairsCensusRowsFor(ctx, report, consensusRepairsCensusNames, reads)
}

func (app *ZeroneApp) consensusRepairsCensusRowsFor(ctx sdk.Context, report *ConsensusRepairsCensus, names []string, reads map[string][]consensusRepairsRead) (map[string]map[string][]byte, error) {
	return app.consensusRepairsCensusRowsWithVisitor(ctx, report, names, reads, nil)
}

// A visitor observes streamed rows without retaining or charging them to a
// selected-input budget. V1 uses no visitor and keeps its original read manifest.
func (app *ZeroneApp) consensusRepairsCensusRowsWithVisitor(ctx sdk.Context, report *ConsensusRepairsCensus, names []string, reads map[string][]consensusRepairsRead, visit func(string, []byte, []byte)) (map[string]map[string][]byte, error) {
	rows := make(map[string]map[string][]byte)
	report.Stores = make(map[string]ConsensusRepairsSize)
	report.Prefixes = make(map[string]map[string]ConsensusRepairsSize)
	report.Censuses = make(map[string]ConsensusRepairsSize)
	report.Reads = make(map[string][]ConsensusRepairsReadSize)
	report.Digests = make(map[string]string)
	for store := range app.keys {
		report.Stores[store] = ConsensusRepairsSize{}
		report.Prefixes[store] = make(map[string]ConsensusRepairsSize)
	}
	hashes := make(map[string]hash.Hash)
	for _, name := range names {
		hashes[name] = sha256.New()
		report.Censuses[name] = ConsensusRepairsSize{}
		for _, read := range reads[name] {
			report.Reads[name] = append(report.Reads[name], ConsensusRepairsReadSize{Store: read.store, Prefix: hex.EncodeToString(read.key), Exact: read.exact})
		}
	}
	digest, _, err := app.consensusRepairsStream(ctx, nil, func(store string, key, value []byte) error {
		n := len(key) + len(value)
		report.Records++
		report.Bytes += n
		size := report.Stores[store]
		size.Records++
		size.Bytes += n
		report.Stores[store] = size
		if report.Prefixes[store] == nil {
			report.Prefixes[store] = make(map[string]ConsensusRepairsSize)
		}
		prefix := hex.EncodeToString(key[:min(1, len(key))])
		size = report.Prefixes[store][prefix]
		size.Records++
		size.Bytes += n
		report.Prefixes[store][prefix] = size
		selected := false
		for _, name := range names {
			for i, read := range reads[name] {
				if store != read.store || !bytes.HasPrefix(key, read.key) || (read.exact && !bytes.Equal(key, read.key)) {
					continue
				}
				size := report.Censuses[name]
				size.Records++
				size.Bytes += n
				report.Censuses[name] = size
				report.Reads[name][i].Records++
				report.Reads[name][i].Bytes += n
				if size.Records > consensusRepairsMaxRecords || size.Bytes > consensusRepairsMaxBytes {
					return fmt.Errorf("%s census resource ceiling exceeded (100000 selected records/64 MiB)", name)
				}
				consensusRepairsHashRow(hashes[name], store, key, value)
				selected = true
				break // charge a row only once per census
			}
		}
		// Charge every selected budget before a visitor can decode the row.
		if visit != nil {
			visit(store, key, value)
		}
		if selected {
			if rows[store] == nil {
				rows[store] = make(map[string][]byte)
			}
			rows[store][string(key)] = bytes.Clone(value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	report.Digest = digest
	for _, name := range names {
		report.Digests[name] = hex.EncodeToString(hashes[name].Sum(nil))
	}
	return rows, nil
}
