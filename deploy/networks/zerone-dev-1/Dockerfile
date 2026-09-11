# Input MUST be the fresh allowlisted build-context.py output, never the repository root.
FROM golang:1.25.14-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437 AS build
WORKDIR /source
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY app ./app
COPY cmd ./cmd
COPY x ./x
COPY internal ./internal
COPY docs/swagger-ui ./docs/swagger-ui
COPY source-context.json ./
# Read the exact source metadata exported from Git; no unbound build-arg revision.
RUN SOURCE_COMMIT=$(sed -n 's/^  "source_commit": "\([0-9a-f]*\)",$/\1/p' source-context.json) && \
    test "${#SOURCE_COMMIT}" = 40 && \
    VERSION="dev-1-$(printf '%s' "$SOURCE_COMMIT" | cut -c1-12)" && \
    go build -mod=readonly -trimpath -buildvcs=false \
      -ldflags="-s -w -X github.com/cosmos/cosmos-sdk/version.Name=zerone -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned -X github.com/cosmos/cosmos-sdk/version.Version=$VERSION -X github.com/cosmos/cosmos-sdk/version.Commit=$SOURCE_COMMIT" \
      -o /out/zeroned ./cmd/zeroned && \
    printf '{"schema":"zerone-dev-build/v1","source_commit":"%s","version":"%s","binary_sha256":"%s"}\n' \
      "$SOURCE_COMMIT" "$VERSION" "$(sha256sum /out/zeroned | cut -d' ' -f1)" > /out/build.json

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
RUN sed -i \
      -e 's|http://deb.debian.org/debian-security|http://snapshot.debian.org/archive/debian-security/20260713T000000Z|g' \
      -e 's|http://deb.debian.org/debian|http://snapshot.debian.org/archive/debian/20260713T000000Z|g' \
      /etc/apt/sources.list.d/debian.sources && \
    printf 'Acquire::Check-Valid-Until "false";\n' >/etc/apt/apt.conf.d/99snapshot && \
    apt-get update && apt-get install -y --no-install-recommends python3 ca-certificates && \
    rm -rf /var/lib/apt/lists/* && mkdir -p /opt/zerone-dev /data && chmod 0700 /data
COPY --from=build /out/zeroned /usr/local/bin/zeroned
COPY --from=build /out/build.json /opt/zerone-dev/build.json
COPY source-context.json /opt/zerone-dev/source-context.json
COPY runtime/runtime.py runtime/gateway.py /opt/zerone-dev/
RUN chmod 0555 /usr/local/bin/zeroned /opt/zerone-dev/runtime.py /opt/zerone-dev/gateway.py && \
    chmod 0444 /opt/zerone-dev/build.json /opt/zerone-dev/source-context.json
EXPOSE 26656 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["python3", "-I", "-B", "/opt/zerone-dev/runtime.py", "run"]
