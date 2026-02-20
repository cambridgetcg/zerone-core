# R3-4 — Governance Module: LIP Lifecycle + Quorum

## Goal

Port the LIP (proposal) governance system with stake-weighted
voting, quorum enforcement, and parameter change execution. This is
the standard governance mechanism — the 2-of-2 research fund voting
(R3-5) builds on top of this.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/gov/` — full module
- `/Users/yuai/Desktop/legible_money/x/gov/keeper/keeper_test.go` — 73 tests
- `/Users/yuai/Desktop/legible_money/reports/batch-21/B21-2-parameter-audit.md` — gov params

## Proto Files

### `proto/zerone/gov/v1/types.proto`
```protobuf
message LIP {
  string id = 1;
  string title = 2;
  string description = 3;
  string category = 4;              // "parameter", "upgrade", "text", "research_spend"
  string proposer = 5;
  string status = 6;                // "draft", "review", "last_call", "voting", "passed", "failed", "withdrawn"
  uint64 submitted_at_block = 7;
  uint64 voting_start_block = 8;
  uint64 voting_end_block = 9;

  // Stake tracking
  string total_stake = 10;          // uzrn staked to support
  uint64 support_threshold_bps = 11;

  // Voting results
  string yes_stake = 12;
  string no_stake = 13;
  string abstain_stake = 14;
  uint64 quorum_threshold_bps = 15; // on 1,000,000 scale (unified)

  // Parameter change (if category = "parameter")
  repeated ParamChange param_changes = 16;
}

message ParamChange {
  string module = 1;
  string key = 2;
  string value = 3;
}

message Vote {
  string voter = 1;
  string lip_id = 2;
  string option = 3;               // "yes", "no", "abstain"
  string stake_weight = 4;
  string reasoning = 5;
  uint64 voted_at_block = 6;
}

// Designated voters for 2-of-2 research fund spending
message ResearchFundVoters {
  string voter1_address = 1;        // founder cold wallet
  string voter2_address = 2;        // AI vault address
}
```

### `proto/zerone/gov/v1/genesis.proto`
```protobuf
message Params {
  // Voting
  uint64 voting_period_blocks = 1;            // default: 102,816 (≈72h)
  uint64 discussion_period_blocks = 2;        // default: 68,544 (≈48h)
  uint64 quorum_threshold_bps = 3;            // default: 334,000 (33.4%) — on 1M scale!
  uint64 support_threshold_bps = 4;           // default: 500,000 (50%)

  // Staking
  string min_lip_stake = 5;                   // uzrn to submit
  string min_vote_stake = 6;                  // uzrn to vote

  // Category-specific
  repeated CategoryConfig category_configs = 7;

  // Research fund voters (2-of-2 designated)
  ResearchFundVoters research_fund_voters = 8;

  // Research fund timing
  uint64 research_discussion_blocks = 9;      // default: 68,544
  uint64 research_voting_blocks = 10;         // default: 102,816
}

message CategoryConfig {
  string category = 1;
  uint64 required_stake_bps = 2;              // stake threshold to advance
  uint64 review_blocks = 3;
}
```

### `proto/zerone/gov/v1/tx.proto`
- MsgSubmitLIP
- MsgStakeLIP (advance from draft to review)
- MsgAdvanceLIPStage
- MsgCastVote
- MsgWithdrawLIP
- MsgUpdateParams

### `proto/zerone/gov/v1/query.proto`
- QueryLIP, QueryLIPs (with status/category filters)
- QueryVote, QueryVotes (by LIP)
- QueryTallyResult
- QueryParams

## Key Implementation

### LIP lifecycle state machine

```
Draft → (stake meets threshold) → Review → (time passes) → Last Call → (time passes) → Voting → Pass/Fail
                                                                                      ↓
                                                                                  Withdrawn (at any stage before voting)
```

### Quorum enforcement (from B22 audit)

**ALL BPS on 1,000,000 scale** (the draft had gov on 10,000 — fix this):

```go
func (k Keeper) checkQuorum(ctx sdk.Context, lip *types.LIP) bool {
    totalBonded := k.stakingKeeper.GetTotalBondedTokens(ctx)
    totalVoted := addBigInt(lip.YesStake, lip.NoStake, lip.AbstainStake)
    quorum := totalBonded * lip.QuorumThresholdBps / 1_000_000
    return totalVoted >= quorum
}
```

### BeginBlocker

```go
func (k Keeper) BeginBlocker(ctx sdk.Context) {
    // 1. Advance LIPs through stages based on time
    // 2. Tally voting-period LIPs that have expired
    // 3. Execute passed parameter changes
    // 4. Handle research fund proposals (R3-5)
}
```

### Parameter change execution

```go
func (k Keeper) executeParamChanges(ctx sdk.Context, changes []types.ParamChange) error {
    for _, change := range changes {
        // Look up the module's param keeper
        // Set the new value
        // Emit event
    }
}
```

## Tests

Port 73 tests. Add:

| Test | Validates |
|------|-----------|
| `TestQuorum_1MScale` | Quorum uses 1,000,000 scale (not 10,000) |
| `TestLIPLifecycle_FullPath` | Draft → Review → LastCall → Voting → Pass |
| `TestLIPLifecycle_FailQuorum` | Not enough votes → fails |
| `TestLIPLifecycle_Withdraw` | Can withdraw before voting |
| `TestParamChange_Executes` | Passed proposal changes the param |
| `TestVoting_StakeWeighted` | Votes weighted by delegation |
| `TestResearchVoters_InParams` | ResearchFundVoters stored in params |

## Verification

```bash
make proto-gen
go build ./...
go test ./x/gov/... -count=1 -v
```

## Commit

```
feat(gov): LIP governance — proposals, stake-weighted voting, quorum, param changes
```

## Do NOT

- Use 10,000 BPS scale (the draft had this — fix to 1,000,000)
- Skip quorum enforcement
- Allow parameter changes without a passed proposal
- Implement research fund voting here (that's R3-5)
