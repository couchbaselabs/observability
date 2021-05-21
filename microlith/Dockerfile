FROM grafana/grafana:7.5.2 as grafana-official
FROM grafana/loki:2.0.0 as loki-official
FROM prom/prometheus as prometheus-official

# Combined image
FROM alpine:3.13

RUN apk add --no-cache ca-certificates bash libcap openssl musl-utils tzdata

COPY --from=grafana-official /usr/share/grafana /usr/share/grafana
COPY --from=grafana-official /etc/grafana /etc/grafana

COPY --from=loki-official /usr/bin/loki /usr/bin/loki
COPY --from=loki-official /etc/loki /etc/loki

COPY --from=prometheus-official /bin/prometheus /bin/prometheus
COPY --from=prometheus-official /bin/promtool /bin/promtool
COPY --from=prometheus-official /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml
COPY --from=prometheus-official /usr/share/prometheus/console_libraries/ /usr/share/prometheus/console_libraries/
COPY --from=prometheus-official /usr/share/prometheus/consoles/ /usr/share/prometheus/consoles/

# GRAFANA:
# From https://github.com/grafana/grafana/blob/main/Dockerfile
ARG GF_UID="472"
ARG GF_GID="0"

ENV PATH="/usr/share/grafana/bin:$PATH" \
    GF_PATHS_CONFIG="/etc/grafana/grafana.ini" \
    GF_PATHS_DATA="/var/lib/grafana" \
    GF_PATHS_HOME="/usr/share/grafana" \
    GF_PATHS_LOGS="/var/log/grafana" \
    GF_PATHS_PLUGINS="/var/lib/grafana/plugins" \
    GF_PATHS_PROVISIONING="/etc/grafana/provisioning"

WORKDIR $GF_PATHS_HOME

RUN if [ ! $(getent group "$GF_GID") ]; then \
      addgroup -S -g $GF_GID grafana; \
    fi

RUN export GF_GID_NAME=$(getent group $GF_GID | cut -d':' -f1) && \
    mkdir -p "$GF_PATHS_HOME/.aws" && \
    adduser -S -u $GF_UID -G "$GF_GID_NAME" grafana && \
    mkdir -p "$GF_PATHS_PROVISIONING/datasources" \
             "$GF_PATHS_PROVISIONING/dashboards" \
             "$GF_PATHS_PROVISIONING/notifiers" \
             "$GF_PATHS_PROVISIONING/plugins" \
             "$GF_PATHS_PROVISIONING/access-control" \
             "$GF_PATHS_LOGS" \
             "$GF_PATHS_PLUGINS" \
             "$GF_PATHS_DATA" && \
    cp "$GF_PATHS_HOME/conf/sample.ini" "$GF_PATHS_CONFIG" && \
    cp "$GF_PATHS_HOME/conf/ldap.toml" /etc/grafana/ldap.toml && \
    chown -R "grafana:$GF_GID_NAME" "$GF_PATHS_DATA" "$GF_PATHS_HOME/.aws" "$GF_PATHS_LOGS" "$GF_PATHS_PLUGINS" "$GF_PATHS_PROVISIONING" && \
    chmod -R 777 "$GF_PATHS_DATA" "$GF_PATHS_HOME/.aws" "$GF_PATHS_LOGS" "$GF_PATHS_PLUGINS" "$GF_PATHS_PROVISIONING"

EXPOSE 3000
# USER grafana

# LOKI:
# From https://github.com/grafana/loki/blob/main/cmd/loki/Dockerfile
RUN addgroup -g 10001 -S loki && \
    adduser -u 10001 -S loki -G loki
RUN mkdir -p /loki/rules && \
    mkdir -p /loki/tmprules && \
    mkdir -p /loki/rules-temp && \
    chown -R loki:loki /etc/loki /loki

# See https://github.com/grafana/loki/issues/1928
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

EXPOSE 3100
# USER loki

# PROMETHEUS
# From https://github.com/prometheus/prometheus/blob/main/Dockerfile
RUN ln -s /usr/share/prometheus/console_libraries /usr/share/prometheus/consoles/ /etc/prometheus/
RUN mkdir -p /prometheus && \
    chown -R nobody:nobody /etc/prometheus /prometheus

# USER       nobody
EXPOSE     9090
VOLUME     [ "/prometheus" ]
WORKDIR    /prometheus

# Customisation for entrypoints and run as a non-root user
WORKDIR /entrypoints
COPY --from=grafana-official /run.sh /entrypoints/grafana.sh
COPY --from=loki-official /usr/bin/loki /entrypoints/loki
COPY --from=prometheus-official /bin/prometheus /entrypoints/prometheus

RUN apk add --no-cache tini
ENV TINI_SUBREAPER=
COPY run.sh /run.sh
RUN chmod a+x /run.sh /entrypoints/*

ARG CB_UID="8453"
ARG CB_GID="8453"
RUN addgroup -g $CB_GID -S couchbase && \
    adduser -u $CB_UID -S couchbase -G couchbase

# Copied from `chown` lines in the original, just substitute our user
RUN chown -R couchbase:couchbase /etc/prometheus /prometheus && \
    chown -R couchbase:couchbase /etc/loki /loki && \
    chown -R couchbase:couchbase "$GF_PATHS_DATA" "$GF_PATHS_HOME/.aws" "$GF_PATHS_LOGS" "$GF_PATHS_PLUGINS" "$GF_PATHS_PROVISIONING"

USER       $CB_UID
ENTRYPOINT [ "/sbin/tini", "--", "/run.sh" ]

# s6 might be a better option as lot more configuration and support for per-service but requires a bit of config to get env, etc. sorted first
# ADD https://github.com/just-containers/s6-overlay/releases/download/v2.2.0.3/s6-overlay-amd64-installer /tmp/s6-overlay-amd64-installer
# RUN chmod +x /tmp/s6-overlay-amd64-installer && \
#     /tmp/s6-overlay-amd64-installer / && \
#     rm -f /tmp/s6-overlay-amd64-installer
# WORKDIR /
# COPY --from=grafana-official /run.sh /etc/services.d/grafana/run
# COPY --from=loki-official /usr/bin/loki /etc/services.d/loki/run
# COPY --from=prometheus-official /bin/prometheus /etc/services.d/prometheus/run
# ENTRYPOINT [ "/init" ]