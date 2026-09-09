package keeper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/gov/types"
)

const (
	maxAccountingProcessRecords    = 100_000
	maxAccountingProcessBytes      = 32 << 20
	maxAccountingProcessValueBytes = 1 << 20
	maxAccountingProcessKeyBytes   = 1024
	maxAccountingProcessJSONTokens = 65_536
	maxAccountingProcessJSONDepth  = 32
)

// ValidateAccountingMigration requires quiescent legacy proposal and delayed-
// action queues for the exact state-bound handoff. It reads raw primary records
// rather than query iterators that can skip malformed entries. It does not
// classify escrow owners, change history, audit every ballot/configuration, or
// authorize migration by itself; the app binds the complete store and balances.
func (k Keeper) ValidateAccountingMigration(ctx sdk.Context) (returnErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			returnErr = fmt.Errorf("governance migration preflight could not read process state")
		}
	}()
	var records, inputBytes uint64
	lips := make(map[string]string)
	for _, prefix := range [][]byte{types.LIPKeyPrefix, types.ResearchSpendKeyPrefix, types.SeatElectionKeyPrefix, types.PhaseTransitionKeyPrefix} {
		err := func() (err error) {
			iter := storetypes.KVStorePrefixIterator(ctx.KVStore(k.storeKey), prefix)
			defer func() { err = errors.Join(err, terminalLIPIteratorError(iter), iter.Close()) }()
			for ; iter.Valid(); iter.Next() {
				key, value := iter.Key(), iter.Value()
				records++
				if records > maxAccountingProcessRecords || len(key) < 2 || len(key) > maxAccountingProcessKeyBytes ||
					len(value) == 0 || len(value) > maxAccountingProcessValueBytes {
					return fmt.Errorf("governance migration process record exceeds its shape or resource bounds")
				}
				rowBytes := uint64(len(key) + len(value))
				if rowBytes > maxAccountingProcessBytes-inputBytes {
					return fmt.Errorf("governance migration process bytes exceed their resource bound")
				}
				inputBytes += rowBytes
				switch prefix[0] {
				case types.LIPKeyPrefix[0]:
					var lip types.LIP
					if err := decodeAccountingProcessJSON(value, &lip); err != nil {
						return fmt.Errorf("LIP process: %w", err)
					}
					if !accountingProcessID(lip.Id) || !bytes.Equal(key, types.LIPKey(lip.Id)) {
						return fmt.Errorf("governance migration LIP key/payload mismatch")
					}
					if !types.IsTerminal(lip.Stage) {
						return fmt.Errorf("governance migration requires every LIP to be terminal")
					}
					if err := types.ValidateLIPExecutionError(&lip); err != nil {
						return err
					}
					lips[lip.Id] = lip.Category
				case types.ResearchSpendKeyPrefix[0]:
					var prop types.ResearchSpendProposal
					if err := decodeAccountingProcessJSON(value, &prop); err != nil {
						return fmt.Errorf("research process: %w", err)
					}
					if prop.ProposalId == 0 || !bytes.Equal(key, types.ResearchSpendKey(prop.ProposalId)) {
						return fmt.Errorf("governance migration research key/payload mismatch")
					}
					if !types.IsTerminalResearchStage(types.ResearchSpendStage(prop.Stage)) {
						return fmt.Errorf("governance migration requires every research spend to be terminal")
					}
				case types.SeatElectionKeyPrefix[0]:
					var prop types.SeatElectionProposal
					if err := decodeAccountingProcessJSON(value, &prop); err != nil {
						return fmt.Errorf("seat process: %w", err)
					}
					if prop.ProposalId == 0 || !bytes.Equal(key, types.SeatElectionKey(prop.ProposalId)) {
						return fmt.Errorf("governance migration seat key/payload mismatch")
					}
					if !types.IsTerminalSeatStage(prop.Stage) {
						return fmt.Errorf("governance migration requires every seat election to be terminal")
					}
				case types.PhaseTransitionKeyPrefix[0]:
					var prop types.PhaseTransitionProposal
					if err := decodeAccountingProcessJSON(value, &prop); err != nil {
						return fmt.Errorf("phase process: %w", err)
					}
					if !accountingProcessID(prop.LipID) || !bytes.Equal(key, types.PhaseTransitionKey(prop.LipID)) {
						return fmt.Errorf("governance migration phase key/payload mismatch")
					}
					category, found := lips[prop.LipID]
					if !found || !types.IsPhaseTransitionCategory(category) || prop.IsRollback != (category == types.CategoryPhaseRollback) {
						return fmt.Errorf("governance migration phase metadata has no matching terminal LIP")
					}
					if !types.IsTerminalPhaseTransitionStage(prop.Stage) {
						return fmt.Errorf("governance migration requires every phase transition to be terminal")
					}
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func accountingProcessID(id string) bool {
	return id != "" && utf8.ValidString(id) && !strings.ContainsRune(id, '\x00') && len(id) < maxAccountingProcessKeyBytes
}

// decodeAccountingProcessJSON checks the token stream before typed decoding so
// duplicate/aliased stage fields cannot hide an active process. Containers and
// total tokens are bounded before generated repeated fields can allocate.
func decodeAccountingProcessJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > maxAccountingProcessValueBytes || !utf8.Valid(raw) {
		return fmt.Errorf("invalid process JSON bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	tokens := 0
	token := func() (json.Token, error) {
		tokens++
		if tokens > maxAccountingProcessJSONTokens {
			return nil, fmt.Errorf("process JSON token bound exceeded")
		}
		return decoder.Token()
	}
	var consume func(int) error
	consume = func(depth int) error {
		if depth > maxAccountingProcessJSONDepth {
			return fmt.Errorf("process JSON nesting bound exceeded")
		}
		next, err := token()
		if err != nil {
			return err
		}
		delimiter, container := next.(json.Delim)
		if !container {
			return nil
		}
		switch delimiter {
		case '{':
			names := make(map[string]struct{})
			for decoder.More() {
				key, err := token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("process JSON object key is not a string")
				}
				if _, duplicate := names[name]; duplicate {
					return fmt.Errorf("process JSON repeats an object key")
				}
				names[name] = struct{}{}
				if err := consume(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := consume(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid process JSON delimiter")
		}
		_, err = token()
		return err
	}
	if err := consume(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("process JSON must contain one value")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return fmt.Errorf("process JSON must be an object")
	}
	allowed := make(map[string]struct{})
	typ := reflect.TypeOf(target).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if field.IsExported() && name != "" && name != "-" {
			allowed[name] = struct{}{}
		}
	}
	for name := range fields {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("process JSON has an unknown or aliased field")
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}
