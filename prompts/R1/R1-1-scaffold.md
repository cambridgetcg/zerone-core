# R1-1 — Repo Scaffold

## Goal

Set up the Zerone repo with build tooling, CI, proto generation, and
directory structure. After this session, `go build ./...` succeeds
(even if the binary does nothing yet) and CI validates every push.

## Working Directory

`/Users/yuai/Desktop/Zerone`

This repo is already `git init`'d with a README.md, REWRITE-PLAN.md, and .gitignore.

## Reference

Draft repo at `/Users/yuai/Desktop/legible_money/`:
- `go.mod` — dependency versions (Cosmos SDK v0.50.15, CometBFT v0.38.20, IBC v8.8.0)
- `proto/buf.yaml`, `proto/buf.gen.yaml` — proto tooling config
- `Makefile` — build targets (if exists)
- `.github/workflows/ci.yml` — CI pipeline

## Deliverables

### 1. `go.mod`

```
module github.com/zerone-chain/zerone

go 1.24.0

require (
    github.com/cosmos/cosmos-sdk v0.50.15
    github.com/cometbft/cometbft v0.38.20
    github.com/cosmos/ibc-go/v8 v8.8.0
    // ... copy all dependencies from draft go.mod
)
```

Copy the **full** `require` and `replace` blocks from the draft's `go.mod`.
Change only the module path (legible-money/legible → zerone-chain/zerone).

Run `go mod tidy` after copying.

### 2. Directory structure

```
Zerone/
├── app/                    # Application wiring (app.go, ante, genesis)
├── cmd/
│   └── zeroned/
│       └── main.go         # Binary entry point
├── proto/
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── zerone/             # All proto definitions under zerone/
│       └── module/
│           └── v1/         # Shared module options
├── x/                      # Custom modules (empty for now)
├── docs/                   # Documentation
├── scripts/                # Build and deployment scripts
├── tests/                  # Cross-stack tests
├── Makefile
├── .github/
│   └── workflows/
│       └── ci.yml
├── .goreleaser.yml         # Release automation
└── REWRITE-PLAN.md         # Already exists
```

### 3. `cmd/zeroned/main.go`

Minimal entry point. For now, just the Cosmos SDK root command with basic
subcommands (init, start, keys, status). Wire no custom modules yet — just
the standard SDK modules (bank, auth, staking, etc.) to prove the binary builds.

Reference: draft's `cmd/legbled/main.go` — adapt for Zerone naming.

### 4. `app/app.go` (skeleton)

Minimal app that:
- Registers standard Cosmos SDK modules (bank, auth, staking, distribution, etc.)
- Has placeholder comments for where Zerone custom modules will go
- Implements `InitChainer`, `BeginBlocker`, `EndBlocker`
- Registers x/upgrade module from day 1

Don't port any custom modules yet — just the skeleton that compiles.

### 5. `Makefile`

```makefile
.PHONY: build install test lint proto-gen

VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=zerone \
           -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o build/zeroned ./cmd/zeroned

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/zeroned

test:
	go test ./... -count=1 -timeout 300s

lint:
	go vet ./...

proto-gen:
	cd proto && buf generate

clean:
	rm -rf build/
```

### 6. `.github/workflows/ci.yml`

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: "1.24"

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Build
        run: make build

      - name: Lint
        run: make lint

      - name: Test
        run: make test
```

### 7. `proto/buf.yaml`

```yaml
version: v1
name: buf.build/zerone-chain/zerone
deps:
  - buf.build/cosmos/cosmos-sdk
  - buf.build/cosmos/cosmos-proto
  - buf.build/cosmos/gogo-proto
  - buf.build/googleapis/googleapis
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

### 8. `proto/buf.gen.yaml`

```yaml
version: v1
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: ..
    opt:
      - module=github.com/zerone-chain/zerone
  - plugin: buf.build/grpc/go
    out: ..
    opt:
      - module=github.com/zerone-chain/zerone
  - plugin: buf.build/grpc-ecosystem/gateway
    out: ..
    opt:
      - module=github.com/zerone-chain/zerone
      - logtostderr=true
```

## Verification

```bash
go build ./...              # compiles
go vet ./...                # clean
make build                  # produces build/zeroned
build/zeroned version       # prints version
build/zeroned init test-node --chain-id zerone-test  # initializes
```

## Commit

Use conventional commits:
```
feat: scaffold Zerone repo with build tooling, CI, and proto config
```

## Do NOT

- Port any custom modules (that's R1-3 and R1-4)
- Generate proto code yet (R1-2 sets up the shared types first)
- Copy the draft repo wholesale — start fresh, reference the draft
- Use any `legible` or `LGM` naming — it's all Zerone/ZRN now
