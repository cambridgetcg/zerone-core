.PHONY: build install test lint proto-gen proto-swagger-gen proto-check creed-check authority-check clean pr-check cosmovisor-init boot-test genesis-check \
       build-linux-amd64 build-linux-arm64 build-darwin-arm64 build-all release \
       darwin-acl-helper frontier-intake-darwin-acl-helper \
       frontier-intake-macos-package frontier-intake-macos-package-check

# Only protocol release tags may become the embedded version. Component or
# tooling tags must not silently relabel the validator binary.
VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Go can discover the parent repository when this checkout is a nested linked
# worktree. Scope Git to the checkout for compilation so vcs.revision and
# vcs.modified describe the same source as the explicit SDK commit field.
BUILD_GIT_DIR := $(shell git rev-parse --absolute-git-dir)
BUILD_WORK_TREE := $(shell git rev-parse --show-toplevel)
GO_BUILD_ENV = GIT_DIR="$(BUILD_GIT_DIR)" GIT_WORK_TREE="$(BUILD_WORK_TREE)"

LDFLAGS := -s -w \
           -X github.com/cosmos/cosmos-sdk/version.Name=zerone \
           -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned \
           -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
           -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

# The Makefile is a source artifact protected by the repository's off-chain
# hash discipline. Its hash is pinned at .makefile-hash; drift fails
# make creed-check. No on-chain Contribution record is implied.

build:
	mkdir -p build
	@if [ "$$(/usr/bin/uname -s)" = Darwin ]; then $(MAKE) darwin-acl-helper; fi
	$(GO_BUILD_ENV) go build -trimpath -buildvcs=true -ldflags "$(LDFLAGS)" -o build/zeroned ./cmd/zeroned

install:
	@if [ "$$(/usr/bin/uname -s)" = Darwin ]; then $(MAKE) darwin-acl-helper; fi
	@if [ "$$(/usr/bin/uname -s)" = Darwin ]; then \
		bin_dir="$$(go env GOBIN)"; \
		if [ -z "$$bin_dir" ]; then bin_dir="$$(go env GOPATH)/bin"; fi; \
		/bin/mkdir -p "$$bin_dir"; \
		bin_dir="$$(cd "$$bin_dir" && /bin/pwd -P)"; \
		bin_owner="$$(/usr/bin/stat -f '%u' "$$bin_dir")"; \
		bin_mode="$$(/usr/bin/stat -f '%Lp' "$$bin_dir")"; \
		{ [ "$$bin_owner" = 0 ] || [ "$$bin_owner" = "$$(/usr/bin/id -u)" ]; } || \
			{ echo "install directory must be owned by root or the current user" >&2; exit 1; }; \
		[ $$((0$$bin_mode & 022)) -eq 0 ] || \
			{ echo "install directory must not be group/world writable" >&2; exit 1; }; \
		test "$$(build/darwin-acl-check < "$$bin_dir")" = \
			"zerone-darwin-acl-v1 clear"; \
		install_tmp="$$(/usr/bin/mktemp "$$bin_dir/.darwin-acl-check.XXXXXX")"; \
		trap '/bin/rm -f "$$install_tmp"' EXIT; \
		/bin/cp build/darwin-acl-check "$$install_tmp"; \
		/bin/chmod 0555 "$$install_tmp"; \
		/bin/mv -f "$$install_tmp" "$$bin_dir/darwin-acl-check"; \
		test "$$("$$bin_dir/darwin-acl-check" < "$$bin_dir/darwin-acl-check")" = \
			"zerone-darwin-acl-v1 clear"; \
		test "$$("$$bin_dir/darwin-acl-check" < "$$bin_dir")" = \
			"zerone-darwin-acl-v1 clear"; \
	fi
	$(GO_BUILD_ENV) go install -buildvcs=true -ldflags "$(LDFLAGS)" ./cmd/zeroned

test:
	@if [ "$$(/usr/bin/uname -s)" = Darwin ]; then $(MAKE) darwin-acl-helper; fi
	go test ./... -count=1 -timeout 300s

lint:
	go vet ./...

proto-gen:
	cd proto && buf generate

proto-swagger-gen:
	@echo "Generating Swagger from proto files..."
	cd proto && buf generate --template buf.gen.swagger.yaml
	go run scripts/merge_swagger.go
	rm -rf tmp-swagger-gen

proto-check:
	@bash scripts/proto-audit.sh

creed-check:
	@bash scripts/check_creed_hash.sh
	@bash scripts/check_useful_work_hash.sh
	@bash scripts/check_tok_substrate_hash.sh
	@bash scripts/check_strange_loop_hash.sh
	@bash scripts/check_sub_creed_hashes.sh
	@bash scripts/check_recursion_doctrine_hash.sh
	@bash scripts/check_phase_1_spec_hash.sh
	@bash scripts/check_specs_and_plans_hashes.sh
	@bash scripts/check_makefile_hash.sh
	@bash scripts/check_readme_hash.sh
	@bash scripts/check_recursion_manifest.sh

authority-check:
	@go test ./tools/authority-graph -count=1
	@go run ./tools/authority-graph report >/dev/null
	@result=$$(go run ./tools/authority-graph target-gate 2>/dev/null); status=$$?; \
		if [ "$$status" -eq 0 ]; then \
			echo "authority target-gate unexpectedly succeeded" >&2; \
			exit 1; \
		fi; \
		printf '%s\n' "$$result" | grep -F '"status": "TARGET_GATE_REFUSED"' >/dev/null && \
		printf '%s\n' "$$result" | grep -F '"dual-staking-ledgers"' >/dev/null

recursion-check:
	@bash scripts/recursion-check.sh

# ── Cross-compile targets ──────────────────────────────────────────────

build-linux-amd64:
	mkdir -p build
	$(GO_BUILD_ENV) GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags "$(LDFLAGS)" -o build/zeroned-linux-amd64 ./cmd/zeroned

build-linux-arm64:
	mkdir -p build
	$(GO_BUILD_ENV) GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags "$(LDFLAGS)" -o build/zeroned-linux-arm64 ./cmd/zeroned

build-darwin-arm64:
	@case "$$(/usr/bin/uname -s)" in Darwin) ;; *) \
		echo "build-darwin-arm64 requires macOS so its mandatory ACL helper can be built" >&2; \
		exit 1 ;; esac
	mkdir -p build
	@$(MAKE) darwin-acl-helper
	$(GO_BUILD_ENV) GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags "$(LDFLAGS)" -o build/zeroned-darwin-arm64 ./cmd/zeroned

build-all: build-linux-amd64 build-linux-arm64 build-darwin-arm64

release: build-all frontier-intake-macos-package
	@echo "Release artifacts built:"
	@ls -la build/zeroned-linux-amd64 build/zeroned-linux-arm64 \
		build/zeroned-darwin-arm64 build/darwin-acl-check \
		build/frontier-intake-macos.tar \
		build/frontier-intake-macos.manifest.json
	@echo ""
	@cd build && set -e; checksum_tmp=; \
		trap '[ -z "$$checksum_tmp" ] || /bin/rm -f "$$checksum_tmp"' EXIT HUP INT TERM; \
		for f in zeroned-linux-amd64 zeroned-linux-arm64 \
			zeroned-darwin-arm64 darwin-acl-check frontier-intake-macos.tar \
			frontier-intake-macos.manifest.json; do \
			checksum_tmp="$$(/usr/bin/mktemp ".$$f.sha256.XXXXXX")"; \
			LC_ALL=C LANG=C /usr/bin/shasum -a 256 "$$f" > "$$checksum_tmp"; \
			/bin/chmod 0444 "$$checksum_tmp"; \
			/bin/mv -f "$$checksum_tmp" "$$f.sha256"; \
			checksum_tmp=; \
		done
	@echo "Checksums:"
	@cat build/zeroned-linux-amd64.sha256 build/zeroned-linux-arm64.sha256 \
		build/zeroned-darwin-arm64.sha256 build/darwin-acl-check.sha256 \
		build/frontier-intake-macos.tar.sha256 \
		build/frontier-intake-macos.manifest.json.sha256

clean:
	rm -rf build/

pr-check: lint test proto-check creed-check authority-check build
	@echo "PR check passed"

cosmovisor-init: build
	@mkdir -p cosmovisor/genesis/bin
	@mkdir -p cosmovisor/upgrades/v1.0.0-testnet/bin
	@cp build/zeroned cosmovisor/genesis/bin/zeroned
	@echo "Cosmovisor initialized with zeroned at cosmovisor/genesis/bin/"

boot-test: build
	@./scripts/boot-test.sh build/zeroned

genesis-check:
	@go run tools/genesis-check/main.go --genesis $(GENESIS)

# macOS mode bits do not include NFSv4 ACL grants. Build one small native
# descriptor inspector for both zeroned and frontier-intake. The generated
# Mach-O remains untracked and enters the checksummed release artifact set
# rather than source control; distributors must bind it to a signed release.
darwin-acl-helper:
	@case "$$(/usr/bin/uname -s)" in Darwin) ;; *) \
		echo "Darwin ACL helper can only be built on macOS" >&2; \
		exit 1 ;; esac
	@/bin/mkdir -p build tools/frontier-intake/build
	@tmp="$$(/usr/bin/mktemp build/.darwin-acl-check.XXXXXX)"; \
		frontier_tmp="$$(/usr/bin/mktemp tools/frontier-intake/build/.darwin-acl-check.XXXXXX)"; \
		trap '/bin/rm -f "$$tmp" "$$frontier_tmp"' EXIT; \
		/usr/bin/xcrun --sdk macosx clang \
			-std=c11 -Os -Wall -Wextra -Werror -Wpedantic -Wconversion \
			-Wsign-conversion -Wformat=2 -Wshadow -fstack-protector-strong \
			-Wl,-no_uuid \
			-mmacosx-version-min=12.0 -arch arm64 -arch x86_64 \
			-o "$$tmp" tools/darwin-acl-check/main.c && \
		/usr/bin/lipo "$$tmp" -verify_arch arm64 x86_64 && \
		/usr/bin/codesign --force --sign - --identifier org.zerone.darwin-acl-check \
			--timestamp=none "$$tmp" >/dev/null && \
		/usr/bin/codesign --verify --strict "$$tmp" && \
		/bin/chmod 0555 "$$tmp" && \
		/bin/mv -f "$$tmp" build/darwin-acl-check && \
		/bin/cp -p build/darwin-acl-check "$$frontier_tmp" && \
		/bin/mv -f "$$frontier_tmp" tools/frontier-intake/build/darwin-acl-check && \
		/usr/bin/cmp build/darwin-acl-check tools/frontier-intake/build/darwin-acl-check && \
		test "$$(build/darwin-acl-check < build/darwin-acl-check)" = \
			"zerone-darwin-acl-v1 clear" && \
		test "$$(build/darwin-acl-check < build)" = \
			"zerone-darwin-acl-v1 clear" && \
		test "$$(tools/frontier-intake/build/darwin-acl-check < tools/frontier-intake/build/darwin-acl-check)" = \
			"zerone-darwin-acl-v1 clear" && \
		test "$$(tools/frontier-intake/build/darwin-acl-check < tools/frontier-intake/build)" = \
			"zerone-darwin-acl-v1 clear"

frontier-intake-darwin-acl-helper: darwin-acl-helper

# One deterministic, read-only archive is the atomic Frontier macOS release
# unit. Its internal manifest binds the exact Bun bundle and universal helper;
# the sidecar digest is integrity metadata, not provenance. Production
# distributors sign the archive bytes in the protected signing workflow.
frontier-intake-macos-package: darwin-acl-helper
	@command -v bun >/dev/null 2>&1 || \
		{ echo "frontier-intake-macos-package requires Bun" >&2; exit 1; }
	@cd tools/frontier-intake && \
		bun build intake.ts --target=bun --outfile=build/frontier-intake.js
	@SOURCE_COMMIT="$(COMMIT)" \
		bun tools/frontier-intake/scripts/macos-artifact.ts build

frontier-intake-macos-package-check:
	@SOURCE_COMMIT="$(COMMIT)" \
		bun tools/frontier-intake/scripts/macos-artifact.ts verify "$(COMMIT)"
