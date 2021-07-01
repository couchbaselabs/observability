FROM grafana/grafana:8.0.3 as grafana-official
FROM grafana/loki:2.2.1 as loki-official
FROM prom/prometheus:v2.28.0 as prometheus-official
FROM prom/alertmanager:v0.22.2 as alertmanager-official
FROM jaegertracing/all-in-one:1.23.0 as jaegaer-official

# Combined image using Nginx as a base to support a simple web server plus proxying into endpoints
FROM nginx:alpine
# hadolint ignore=DL3018
RUN apk add --upgrade --no-cache ca-certificates bash libcap openssl musl-utils tzdata tini curl libgcc fts gettext

COPY --from=grafana-official /usr/share/grafana /usr/share/grafana
COPY --from=grafana-official /etc/grafana /etc/grafana

COPY --from=loki-official /usr/bin/loki /usr/bin/loki
COPY --from=loki-official /etc/loki /etc/loki

COPY --from=prometheus-official /bin/prometheus /bin/prometheus
COPY --from=prometheus-official /bin/promtool /bin/promtool
COPY --from=prometheus-official /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml
COPY --from=prometheus-official /usr/share/prometheus/console_libraries/ /usr/share/prometheus/console_libraries/
COPY --from=prometheus-official /usr/share/prometheus/consoles/ /usr/share/prometheus/consoles/

# Try to keep the originals as they are and fix anything later to make it easy to merge changes

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

# hadolint ignore=SC2046
RUN if [ ! $(getent group "$GF_GID") ]; then \
      addgroup -S -g $GF_GID grafana; \
    fi

# hadolint ignore=DL3059,DL4006,SC2155
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

WORKDIR /

# LOKI:
# From https://github.com/grafana/loki/blob/main/cmd/loki/Dockerfile
RUN addgroup -g 10001 -S loki && \
    adduser -u 10001 -S loki -G loki && \
    mkdir -p /loki/rules && \
    mkdir -p /loki/tmprules && \
    mkdir -p /loki/rules-temp && \
    chown -R loki:loki /etc/loki /loki

# See https://github.com/grafana/loki/issues/1928
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

EXPOSE 3100
# USER loki

# PROMETHEUS
# From https://github.com/prometheus/prometheus/blob/main/Dockerfile
RUN ln -s /usr/share/prometheus/console_libraries /usr/share/prometheus/consoles/ /etc/prometheus/ && \
    mkdir -p /prometheus && \
    chown -R nobody:nobody /etc/prometheus /prometheus

# USER       nobody
EXPOSE     9090
VOLUME     [ "/prometheus" ]

# ALERT MANAGER
# From https://github.com/prometheus/alertmanager/blob/master/Dockerfile
COPY --from=alertmanager-official /bin/amtool      /bin/amtool
COPY --from=alertmanager-official /bin/alertmanager /bin/alertmanager
COPY --from=alertmanager-official /etc/alertmanager/alertmanager.yml     /etc/alertmanager/alertmanager.yml

RUN mkdir -p /alertmanager && \
    chown -R nobody:nobody etc/alertmanager /alertmanager

# USER       nobody
EXPOSE     9093
VOLUME     [ "/alertmanager" ]

# JAEGER ALL IN ONE
# https://github.com/jaegertracing/jaeger/blob/master/cmd/all-in-one/Dockerfile
COPY --from=jaegaer-official /go/bin/all-in-one-linux /go/bin/all-in-one-linux
COPY --from=jaegaer-official /etc/jaeger/ /etc/jaeger/

# Agent zipkin.thrift compact
EXPOSE 5775/udp

# Agent jaeger.thrift compact
EXPOSE 6831/udp

# Agent jaeger.thrift binary
EXPOSE 6832/udp

# Agent config HTTP
EXPOSE 5778

# Collector HTTP
EXPOSE 14268

# Collector gRPC
EXPOSE 14250

# Web HTTP
EXPOSE 16686

# Default configuration file for setting sampling strategies
ENV SAMPLING_STRATEGIES_FILE=/etc/jaeger/sampling_strategies.json

# VOLUME ["/tmp"]
# ENTRYPOINT ["/go/bin/all-in-one-linux"]

# Add support for auto-provisioning now
COPY grafana/grafana.ini /etc/grafana/grafana.ini
COPY grafana/provisioning/ /etc/grafana/provisioning/
COPY grafana/cblogo.png /usr/share/grafana/public/img/cblogo.png
ENV GF_SECURITY_ADMIN_PASSWORD=password
ENV GF_USERS_ALLOW_SIGN_UP=false

COPY prometheus/ /etc/prometheus/
COPY alertmanager/ /etc/alertmanager/
COPY jaeger/ /etc/jaeger/

# Simple static webserver
COPY nginx/ /etc/nginx/
COPY html/ /html/
EXPOSE 80

# Customisation for entrypoints and run as a non-root user
COPY entrypoints/ /entrypoints/
COPY --from=grafana-official /run.sh /entrypoints/grafana.sh

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
    chown -R couchbase:couchbase "$GF_PATHS_DATA" "$GF_PATHS_HOME/.aws" "$GF_PATHS_LOGS" "$GF_PATHS_PLUGINS" "$GF_PATHS_PROVISIONING" && \
    chown -R couchbase:couchbase /etc/alertmanager /alertmanager && \
    chown -R couchbase:couchbase /html /etc/nginx/ /var/cache/nginx && \
    chown -R couchbase:couchbase /etc/jaeger/ && \
    mkdir -p /logs && chmod 777 /logs && \
    mkdir -p /etc/prometheus/couchbase/monitoring/ && chmod 777 /etc/prometheus/couchbase/monitoring/ && \
    chown -R couchbase:couchbase /entrypoints /logs /etc/prometheus/couchbase/

# Location of dynamic target information for Prometheus
ENV PROMETHEUS_DYNAMIC_INTERNAL_DIR=/etc/prometheus/couchbase/monitoring/
ENV PROMETHEUS_DYNAMIC_DIR=/etc/prometheus/couchbase/

# Add all useful and required information
COPY licenses/* /licenses/
COPY README.md /help.1

USER       $CB_UID
ENTRYPOINT [ "/sbin/tini", "--", "/run.sh" ]