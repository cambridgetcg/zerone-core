package keeper

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func decodeContributionRecord(raw []byte, modelID string) (*types.ContributionRecord, error) {
	if len(raw) > recordInventoryMaxBytes {
		return nil, fmt.Errorf("contribution record exceeds inventory bounds")
	}
	if err := types.ValidateRawPolicyField(raw, 9); err != nil {
		return nil, fmt.Errorf("contribution attribution policy: %w", err)
	}
	var record types.ContributionRecord
	if err := proto.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("decode contribution record: %w", err)
	}
	if hasUnknownRecordFields(record.ProtoReflect()) {
		return nil, fmt.Errorf("contribution record contains unsupported fields")
	}
	if record.ModelId != modelID {
		return nil, fmt.Errorf("contribution primary key does not match model identity")
	}
	if err := types.ValidateContributionRecord(&record); err != nil {
		return nil, err
	}
	return &record, nil
}

// GetContributionRecordChecked distinguishes an absent declaration from an
// unreadable or unsupported one. Callers must propagate the latter.
func (k Keeper) GetContributionRecordChecked(ctx context.Context, modelID string) (*types.ContributionRecord, bool, error) {
	if modelID == "" {
		return nil, false, nil
	}
	raw, err := k.storeService.OpenKVStore(ctx).Get(types.ContributionByModelKey(modelID))
	if err != nil {
		return nil, false, err
	}
	if raw == nil {
		return nil, false, nil
	}
	record, err := decodeContributionRecord(raw, modelID)
	if err != nil {
		return nil, false, err
	}
	return record, true, nil
}

// GetAllContributionRecordsChecked shares the existing strict primary-record
// inventory bounds and iterator checks. It never returns a partial inventory.
func (k Keeper) GetAllContributionRecordsChecked(ctx context.Context) ([]*types.ContributionRecord, error) {
	var records []*types.ContributionRecord
	err := k.scanRecordPrimary(ctx, types.ContributionByModelKeyPrefix, func() proto.Message { return &types.ContributionRecord{} }, func(record proto.Message, id string) error {
		contribution := record.(*types.ContributionRecord)
		if contribution.ModelId != id {
			return fmt.Errorf("contribution primary key does not match model identity")
		}
		if err := types.ValidateContributionRecord(contribution); err != nil {
			return err
		}
		records = append(records, contribution)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}
