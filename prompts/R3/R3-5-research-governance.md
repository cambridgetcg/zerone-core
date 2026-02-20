# R3-5 — Research Fund 2-of-2 Governance

## Goal

Implement the 2-of-2 designated voter system for research fund spending.
This is the mechanism where ONE agrees on fund disbursement — both designated
voters must sign, or the funds don't move.

## Dependencies

- R3-4 (gov module) must be complete

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/gov/keeper/research_spend.go` — from B22-2
- `/Users/yuai/Desktop/legible_money/reports/batch-22/B22-2-research-governance.md`
- `/Users/yuai/Desktop/legible_money/x/vesting_rewards/keeper/rewards.go` — DisburseFromResearchFund

## Design

Two designated voters control research fund spending:
- **Voter 1:** Cold wallet (offline signing)
- **Voter 2:** Vault signing key (remote, challenge-response authenticated)

Both must vote YES for funds to move. Either voting NO blocks the proposal.
This is NOT stake-weighted — it's explicit 2-of-2 consensus.

## Implementation

### Research Spend Proposal (extends gov module)

Add to `x/gov/keeper/research_spend.go`:

```go
type ResearchSpendProposal struct — already defined in R3-4 types.proto
```

### New messages (add to gov tx.proto if not already)

```protobuf
message MsgSubmitResearchSpend {
  string proposer = 1;          // must be voter1 or voter2
  string title = 2;
  string description = 3;
  string recipient = 4;         // bech32
  string amount = 5;            // uzrn
  string justification = 6;    // detailed reasoning (audit trail)
}

message MsgVoteResearchSpend {
  string voter = 1;             // must be voter1 or voter2
  string proposal_id = 2;
  string vote = 3;              // "yes" or "no"
  string reasoning = 4;         // stored on-chain
}
```

### Keeper methods

```go
// SubmitResearchSpend creates a new research fund spending proposal.
// Only voter1 or voter2 can submit.
func (k Keeper) SubmitResearchSpend(ctx sdk.Context, msg *types.MsgSubmitResearchSpend) (*types.ResearchSpendProposal, error)

// VoteResearchSpend records a vote on a research spend proposal.
// Only voter1 or voter2 can vote. Each votes exactly once.
func (k Keeper) VoteResearchSpend(ctx sdk.Context, msg *types.MsgVoteResearchSpend) error

// CheckResearchProposals is called in BeginBlocker to:
// - Advance DISCUSSION → VOTING when discussion period ends
// - Execute proposals with 2 YES votes
// - Expire proposals past voting deadline
func (k Keeper) CheckResearchProposals(ctx sdk.Context)
```

### Execution flow

```go
func (k Keeper) executeResearchSpend(ctx sdk.Context, proposal *types.ResearchSpendProposal) error {
    recipientAddr, _ := sdk.AccAddressFromBech32(proposal.Recipient)
    amount, _ := sdk.ParseCoinsNormalized(proposal.Amount + "uzrn")

    // Check research fund has sufficient balance
    balance := k.bankKeeper.GetBalance(ctx, researchFundAddr, "uzrn")
    if balance.Amount.LT(amount[0].Amount) {
        return types.ErrInsufficientResearchFunds
    }

    // Disburse via vesting_rewards keeper
    return k.vestingKeeper.DisburseFromResearchFund(ctx, recipientAddr, amount)
}
```

### State transitions (in BeginBlocker)

```
DISCUSSION: block >= submitted + discussion_blocks → VOTING
VOTING:
  - Both YES → PASSED → execute → record result
  - Either NO → FAILED
  - block >= voting_start + voting_blocks → EXPIRED
```

### On-chain audit trail

Every proposal stores:
- Both votes with reasoning text
- Execution block and result
- All queryable via CLI and gRPC

### CLI commands

```
zeroned tx gov submit-research-spend [title] [description] [recipient] [amount] [justification]
zeroned tx gov vote-research-spend [proposal-id] [yes/no] [reasoning]
zeroned query gov research-proposals
zeroned query gov research-proposal [id]
```

### KV storage

Use the gov module's existing store with a dedicated prefix:
```go
ResearchProposalPrefix = []byte{0x30}
ResearchProposalCountKey = []byte{0x31}
```

## Tests

| Test | Validates |
|------|-----------|
| `TestResearchSpend_SubmitByVoter1` | Voter 1 can submit |
| `TestResearchSpend_SubmitByVoter2` | Voter 2 can submit |
| `TestResearchSpend_SubmitByNonVoter` | Non-voter → error |
| `TestResearchSpend_BothYes` | 2 YES → funds transferred to recipient |
| `TestResearchSpend_OneNo` | Either NO → proposal fails, funds stay |
| `TestResearchSpend_Timeout` | Voting expires → EXPIRED |
| `TestResearchSpend_DoubleVote` | Same voter twice → error |
| `TestResearchSpend_NonVoterVote` | Non-voter tries to vote → error |
| `TestResearchSpend_InsufficientFunds` | Amount > balance → error |
| `TestResearchSpend_DiscussionPeriod` | Cannot vote during discussion |
| `TestResearchSpend_AuditTrail` | Votes + reasoning recorded |
| `TestResearchSpend_EmptyVoters` | Both voters empty → submit fails |
| `TestResearchVoters_GovernanceOnly` | Voter addresses only changeable via standard governance |

## Verification

```bash
go build ./...
go test ./x/gov/... ./x/vesting_rewards/... -count=1 -v
```

## Commit

```
feat(gov): 2-of-2 research fund governance — submit, vote, execute, audit trail
```

## Do NOT

- Use stake-weighted voting for research spend
- Allow execution without both votes
- Skip the discussion period
- Allow voter address changes without standard governance proposal
- Bypass the vesting_rewards SendRestriction
