package types

import (
	"strings"
	"testing"
)

func TestGenesisStateValidateClaimScheduleIndexes(t *testing.T) {
	schedule := func(id, claimID string) *VestingSchedule {
		return &VestingSchedule{Id: id, ClaimId: claimID}
	}

	tests := []struct {
		name    string
		state   *GenesisState
		wantErr string
	}{
		{
			name: "valid explicit index selects one duplicate legacy schedule",
			state: &GenesisState{
				Params:          DefaultParams(),
				CategoryConfigs: DefaultCategoryConfigs(),
				VestingSchedules: []*VestingSchedule{
					schedule("vesting-a", "claim-1"),
					schedule("vesting-b", "claim-1"),
				},
				ClaimScheduleIndexes: []*ClaimScheduleIndex{
					{ClaimId: "claim-1", VestingId: "vesting-b"},
				},
			},
		},
		{
			name: "dangling vesting id",
			state: &GenesisState{
				Params:           DefaultParams(),
				CategoryConfigs:  DefaultCategoryConfigs(),
				VestingSchedules: []*VestingSchedule{schedule("vesting-a", "claim-1")},
				ClaimScheduleIndexes: []*ClaimScheduleIndex{
					{ClaimId: "claim-1", VestingId: "missing"},
				},
			},
			wantErr: "missing vesting schedule",
		},
		{
			name: "claim mismatch",
			state: &GenesisState{
				Params:           DefaultParams(),
				CategoryConfigs:  DefaultCategoryConfigs(),
				VestingSchedules: []*VestingSchedule{schedule("vesting-a", "claim-2")},
				ClaimScheduleIndexes: []*ClaimScheduleIndex{
					{ClaimId: "claim-1", VestingId: "vesting-a"},
				},
			},
			wantErr: "with claim_id",
		},
		{
			name: "duplicate claim index",
			state: &GenesisState{
				Params:           DefaultParams(),
				CategoryConfigs:  DefaultCategoryConfigs(),
				VestingSchedules: []*VestingSchedule{schedule("vesting-a", "claim-1")},
				ClaimScheduleIndexes: []*ClaimScheduleIndex{
					{ClaimId: "claim-1", VestingId: "vesting-a"},
					{ClaimId: "claim-1", VestingId: "vesting-a"},
				},
			},
			wantErr: "duplicate claim schedule index",
		},
		{
			name: "incomplete explicit index",
			state: &GenesisState{
				Params:          DefaultParams(),
				CategoryConfigs: DefaultCategoryConfigs(),
				VestingSchedules: []*VestingSchedule{
					schedule("vesting-a", "claim-1"),
					schedule("vesting-b", "claim-2"),
				},
				ClaimScheduleIndexes: []*ClaimScheduleIndex{
					{ClaimId: "claim-1", VestingId: "vesting-a"},
				},
			},
			wantErr: "has schedules but no claim schedule index",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.state.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
