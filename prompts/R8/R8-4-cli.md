# R8-4 — CLI: Transaction & Query Commands

## Goal

Register all tx and query CLI commands for all 32 custom modules. Port human-friendly
helper commands (quick-register, dashboard, explore) from the draft. Every module's
messages must be submittable from the command line.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/cmd/legbled/cmd/` — CLI commands:
  - `quick_tx.go` — quick-register, quick-stake, quick-claim helpers
  - `dashboard.go` — status dashboard showing chain health, staking, knowledge stats
  - `explore.go` — explore command for browsing facts, domains, tools
  - `amounts.go` — human-friendly amount formatting (ZRN ↔ uzrn)
- Each module's `client/cli/` directory for tx and query commands

## Current State

`cmd/zeroned/cmd/root.go` has basic structure with `queryCmd` and `txCmd` subcommands.
Some modules may already have CLI commands registered. Need to audit and fill gaps.

## Steps

### 1. Audit Existing CLI Registration
Check which modules have `client/cli/` directories with `tx.go` and `query.go`:
```bash
for mod in $(ls x/); do
  echo -n "$mod: "
  ls x/$mod/client/cli/tx.go x/$mod/client/cli/query.go 2>/dev/null | wc -l
done
```

### 2. Create Missing CLI Commands
For each module missing CLI commands, create `x/<module>/client/cli/`:
- `tx.go` — one command per `Msg*` type
- `query.go` — one command per `Query*` request

Use the standard Cosmos SDK CLI pattern:
```go
func CmdSubmitClaim() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "submit-claim [domain] [content]",
        Short: "Submit a knowledge claim",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            clientCtx, err := client.GetClientTxContext(cmd)
            // ... build and broadcast msg
        },
    }
    flags.AddTxFlagsToCmd(cmd)
    return cmd
}
```

### 3. Register All Commands in Root
In `cmd/zeroned/cmd/root.go`, ensure `queryCmd` and `txCmd` include subcommands
from every module:
```go
queryCmd.AddCommand(
    authcli.GetQueryCmd(),
    stakingcli.GetQueryCmd(),
    knowledgecli.GetQueryCmd(),
    // ... all 32 modules
)
txCmd.AddCommand(
    authcli.GetTxCmd(),
    stakingcli.GetTxCmd(),
    knowledgecli.GetTxCmd(),
    // ... all 32 modules
)
```

### 4. Helper Commands
Port from draft:

#### quick-register
```
zeroned tx quick-register [moniker] --from <key>
```
Combines: auth.RegisterAccount + staking.RegisterValidator + home.CreateHome in one tx.

#### dashboard
```
zeroned query dashboard
```
Shows: chain height, validator count, total staked, knowledge stats, tool count, SSI score.

#### explore
```
zeroned query explore facts [domain]
zeroned query explore tools [--available]
zeroned query explore validators [--tier 3]
```
Browse chain state in human-friendly format.

#### amounts
```
zeroned amounts 1.5 ZRN     → 1500000 uzrn
zeroned amounts 1500000 uzrn → 1.5 ZRN
```

### 5. Add Genesis Helper
Extend `cmd/zeroned/cmd/genesis.go`:
- `add-genesis-account` (already exists — verify)
- `add-genesis-validator` — adds a gentx for a validator
- `prepare-genesis` — generates full testnet genesis (works with R8-3)

## Tests

1. `zeroned tx --help` lists all module subcommands
2. `zeroned query --help` lists all module subcommands
3. `zeroned amounts` conversion works both directions
4. Build succeeds with all CLI wiring: `go build ./cmd/zeroned/`

## Constraints

- Every Msg type must have a corresponding CLI command
- Every Query type must have a corresponding CLI command
- Use consistent naming: `zeroned tx <module> <action>` / `zeroned query <module> <action>`
- All amounts accept both ZRN and uzrn with auto-conversion
- Flag `--from` required on all tx commands
- This session focuses on CLI wiring — actual chain boot tested in R8-5 / R9
