# syntax=docker/dockerfile:1.7
ARG NGINX_IMAGE
FROM ${NGINX_IMAGE}

RUN command -v nginx >/dev/null 2>&1 && \
    command -v envsubst >/dev/null 2>&1 && \
    rm -f /etc/nginx/conf.d/default.conf && \
    mkdir -p /etc/zerone-query-gateway
COPY default.conf.template /etc/zerone-query-gateway/default.conf.template
COPY entrypoint.sh /usr/local/bin/zerone-query-gateway
RUN chmod 0444 /etc/zerone-query-gateway/default.conf.template && \
    chmod 0555 /usr/local/bin/zerone-query-gateway

EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/zerone-query-gateway"]
