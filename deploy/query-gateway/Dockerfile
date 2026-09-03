# syntax=docker/dockerfile:1.7
ARG GO_IMAGE=golang:1.25.14-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437
ARG NGINX_IMAGE

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS readiness-builder
WORKDIR /src
COPY .sanitized-query-gateway-context-v1 /src/.sanitized-query-gateway-context-v1
COPY nginx-image.txt /src/nginx-image.txt
COPY readiness.go /src/readiness.go
ARG NGINX_IMAGE
ARG SOURCE_COMMIT
ARG BUILD_PROFILE
ARG RELEASE_TAG
ARG RELEASE_SIGNER_FINGERPRINT
ARG TARGETOS
ARG TARGETARCH
RUN mkdir -p /out && \
    test "${TARGETOS}/${TARGETARCH}" = "linux/amd64" && \
    grep -Fqx "${NGINX_IMAGE}" /src/nginx-image.txt && \
    grep -Fqx "profile=${BUILD_PROFILE}" /src/.sanitized-query-gateway-context-v1 && \
    grep -Fqx "source_commit=${SOURCE_COMMIT}" /src/.sanitized-query-gateway-context-v1 && \
    grep -Fqx "release_tag=${RELEASE_TAG}" /src/.sanitized-query-gateway-context-v1 && \
    grep -Fqx "release_signer_fingerprint=${RELEASE_SIGNER_FINGERPRINT}" /src/.sanitized-query-gateway-context-v1 && \
    grep -Fqx "platform=${TARGETOS}/${TARGETARCH}" /src/.sanitized-query-gateway-context-v1 && \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
      go build -trimpath -buildvcs=false -ldflags '-s -w' \
      -o /out/zerone-query-readiness /src/readiness.go

FROM ${NGINX_IMAGE}

RUN command -v nginx >/dev/null 2>&1 && \
    command -v envsubst >/dev/null 2>&1 && \
    nginx -V 2>&1 | grep -Fq -- '--with-http_auth_request_module' && \
    rm -f /etc/nginx/conf.d/default.conf && \
    mkdir -p /etc/zerone-query-gateway
COPY default.conf.template /etc/zerone-query-gateway/default.conf.template
COPY entrypoint.sh /usr/local/bin/zerone-query-gateway
COPY --from=readiness-builder /out/zerone-query-readiness /usr/local/bin/zerone-query-readiness
RUN chmod 0444 /etc/zerone-query-gateway/default.conf.template && \
    chmod 0555 /usr/local/bin/zerone-query-gateway /usr/local/bin/zerone-query-readiness

EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/zerone-query-gateway"]
