ARG PROD_VERSION=1.0.0
ARG PROD_BUILD=999

FROM grafana/grafana:8.3.6 as grafana-official
FROM grafana/loki:2.4.1 as loki-official
FROM prom/prometheus:v2.33.3 as prometheus-official
FROM prom/alertmanager:v0.23.0 as alertmanager-official
FROM jaegertracing/all-in-one:1.34.0 as jaegaer-official
FROM hairyhenderson/gomplate:stable AS gomplate-official

FROM golang:1.17 as builder
# Builder image so ignore pinning and we do want recommended packages
# hadolint ignore=DL3008,DL3015
RUN apt-get update && apt-get install -y sqlcipher libssl-dev openssl openssh-client && \
    rm -rf /var/lib/apt/lists/*

# Set up SSH access
# hadolint ignore=SC2174
RUN mkdir -p /bin /etc/couchbase && \
    mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

# Build the Prometheus merge tool so we can ensure we have a static build
ARG PROM_CONFIG_TOOL_VERSION=v0.3.0
WORKDIR /src/github.com/lablabs/prometheus-alert-overrider
RUN curl -Lo /src/prom.tgz "https://github.com/lablabs/prometheus-alert-overrider/archive/refs/tags/$PROM_CONFIG_TOOL_VERSION.tar.gz" && \
    tar --strip-components 1 -xzvf /src/prom.tgz && \
    CGO_ENABLED=1 GOOS=linux go build -trimpath -a -ldflags '-linkmode external -extldflags "-static"' -tags netgo -o /bin/prometheus_merge main.go

# Build the configuration service
WORKDIR /src/github.com/couchbaselabs/observability/config-svc
COPY config-svc/ .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -a -o /bin/cmoscfg -tags netgo ./cmd/cmoscfg

# Generate documentation and include locally as a static site
# We do not include Antora, just use it: https://gitlab.com/antora/docker-antora/-/blob/main/LICENSE
FROM antora/antora:3.0.0-alpha.9 as antora-generator
RUN mkdir -p /antora/cmos
ENV HOME=/antora

# Antora requires the input be a Git repo with at least one commit: https://gitlab.com/antora/antora/-/issues/284
# No need to pin
# hadolint ignore=DL3018
RUN apk add --upgrade --no-cache git && \
    git init /antora/docs && \
    touch /antora/docs/.gitignore && \
    git -C /antora/docs add . && \
    git config --global user.name "ignore.me" && \
    git config --global user.email "ignore.me" && \
    git -C /antora/docs commit -m "ignore me"

# Put the changeable stuff here
COPY docs/ /antora/docs/

# Couchbase proprietary start
COPY cbmultimanager-docs/ /antora/cbmultimanager
RUN git init /antora/cbmultimanager && \
    touch /antora/cbmultimanager/.gitignore && \
    git -C /antora/cbmultimanager add . && \
    git -C /antora/cbmultimanager commit -m "ignore me"

# antora-container-playbook defaults to OSS, override it
COPY docs/antora-container-playbook-proprietary.yml /antora/docs/antora-container-playbook.yml
# Couchbase proprietary end


RUN antora --to-dir /antora/cmos/ /antora/docs/antora-container-playbook.yml

# Combined image using Nginx as a base to support a simple web server plus proxying into endpoints
FROM nginx:alpine
ARG PROD_VERSION
ARG PROD_BUILD

# hadolint ignore=DL3018
RUN apk add --upgrade --no-cache ca-certificates bash libcap openssl musl-utils tzdata tini curl libgcc fts gettext jq sudo

COPY --from=grafana-official /usr/share/grafana /usr/share/grafana
COPY --from=grafana-official /etc/grafana /etc/grafana

COPY --from=loki-official /usr/bin/loki /usr/bin/loki

COPY --from=prometheus-official /bin/prometheus /bin/prometheus
COPY --from=prometheus-official /bin/promtool /bin/promtool
COPY --from=prometheus-official /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml
COPY --from=prometheus-official /usr/share/prometheus/console_libraries/ /usr/share/prometheus/console_libraries/
COPY --from=prometheus-official /usr/share/prometheus/consoles/ /usr/share/prometheus/consoles/

# Support scripted access to kubernetes API, e.g. to get credentials from Secrets
ARG KUBECTL_VERSION=v1.22.3
ENV KUBECTL_VERSION=$KUBECTL_VERSION
RUN curl -LO "https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl" && \
	chmod +x kubectl && \
	mv kubectl /usr/local/bin/

# Add the Prometheus config handling tool
# https://github.com/lablabs/prometheus-alert-overrider
ARG PROM_CONFIG_TOOL_VERSION=v0.3.0
ENV PROM_CONFIG_TOOL_VERSION=$PROM_CONFIG_TOOL_VERSION
COPY --from=builder /bin/prometheus_merge /bin/prometheus_merge
RUN chmod a+x /bin/prometheus_merge

# Helm - to support deployment from this container of custom tools
ARG HELM_VERSION=v3.7.1
ENV HELM_VERSION=$HELM_VERSION
RUN curl -LO "https://get.helm.sh/helm-$HELM_VERSION-linux-amd64.tar.gz" && \
	mkdir -p "/usr/local/helm-$HELM_VERSION" && \
	tar -xzf "helm-$HELM_VERSION-linux-amd64.tar.gz" -C "/usr/local/helm-$HELM_VERSION" && \
	ln -s "/usr/local/helm-$HELM_VERSION/linux-amd64/helm" /usr/local/bin/helm && \
	rm -f "helm-$HELM_VERSION-linux-amd64.tar.gz"

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
RUN export GF_GID_NAME="$(getent group $GF_GID | cut -d':' -f1)" && \
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
# Note: we removed `/etc/loki` from the last line - we'll chown it later.
# Otherwise this is identical to the command from the upstream Dockerfile.
RUN addgroup -g 10001 -S loki && \
    adduser -u 10001 -S loki -G loki && \
    mkdir -p /loki/rules && \
    mkdir -p /loki/rules-temp && \
    chown -R loki:loki /loki

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
# VOLUME     [ "/prometheus" ]

# ALERT MANAGER
# From https://github.com/prometheus/alertmanager/blob/master/Dockerfile
COPY --from=alertmanager-official /bin/amtool      /bin/amtool
COPY --from=alertmanager-official /bin/alertmanager /bin/alertmanager
COPY --from=alertmanager-official /etc/alertmanager/alertmanager.yml     /etc/alertmanager/alertmanager.yml

RUN mkdir -p /alertmanager && \
    chown -R nobody:nobody etc/alertmanager /alertmanager

# USER       nobody
EXPOSE     9093
# VOLUME     [ "/alertmanager" ]

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

# Send traces to us
ENV JAEGER_AGENT_HOST=localhost

# VOLUME ["/tmp"]
# ENTRYPOINT ["/go/bin/all-in-one-linux"]

# Add support for auto-provisioning now
COPY grafana/grafana.ini /etc/grafana/grafana.ini
COPY grafana/provisioning/ /etc/grafana/provisioning/
COPY grafana/cblogo.png /usr/share/grafana/public/img/cblogo.png
COPY grafana/cmos-doc-redirect.html /usr/share/grafana/public/
ENV GF_SECURITY_ADMIN_PASSWORD=password
ENV GF_USERS_ALLOW_SIGN_UP=false

# Ensure we install Grafana plugins at build time, not at runtime
# We use a slightly different syntax of GF_INSTALL_PLUGINS to allow pinning versions:
# should be a comma-separated list of plugins, optionally with a space and a version to download that version.
# If no version is given, use the latest.
ARG GF_INSTALL_PLUGINS="marcusolsson-json-datasource 1.3.0"
COPY ./scripts/install-grafana-plugins.sh /scripts/install-grafana-plugins.sh
RUN /bin/bash /scripts/install-grafana-plugins.sh \
# Blank out the environment variable to prevent trying to install the plugins again at runtime
# End-users can still override it to install additional plugins, but this should prevent startup
# failures in an air-gapped environment
ENV GF_INSTALL_PLUGINS=""

COPY prometheus/ /etc/prometheus/
COPY alertmanager/ /etc/alertmanager/
COPY jaeger/ /etc/jaeger/
COPY loki/ /etc/loki
ENV CMOS_LOGS_ROOT=/logs

# Configuration Service

COPY --from=builder /bin/cmoscfg /bin/cmoscfg
COPY --from=builder /src/github.com/couchbaselabs/observability/config-svc/entrypoint.sh /entrypoints/cmoscfg.sh

# Simple static webserver
COPY nginx/ /etc/nginx/
COPY html/ /html/
EXPOSE 8080
# Add generated documentation as well to Nginx
COPY --from=antora-generator /antora/cmos /html/docs

# Install Gomplate for nginx config
COPY --from=gomplate-official /gomplate /bin/gomplate

# Couchbase cluster monitor - need these to keep OSS generic
RUN mkdir -p /data /priv /ui

# Couchbase proprietary start
# Cannot consolidate into single RUN as need to wrap in start/end tags
# hadolint ignore=DL3059
RUN openssl req -x509 -nodes -days 365 \
    -subj "/C=CA/ST=QC/O=Couchbase, Inc./CN=couchbase.com" \
    -addext "subjectAltName=DNS:couchbase.com" \
    -newkey rsa:2048 \
    -keyout /priv/server.key -out /priv/server.crt;

COPY ./bin/cbmultimanager-linux-amd64 /bin/cbmultimanager
COPY ./bin/cbeventlog-linux-amd64 /bin/cbeventlog

RUN chmod a+x /bin/cbmultimanager /bin/cbeventlog

EXPOSE 7196 7197

RUN mkdir -p /licenses && \
    curl -Lo /licenses/couchbase-cluster-monitor-notices.txt \
    https://raw.githubusercontent.com/couchbase/product-metadata/master/couchbase-cluster-monitor/blackduck/${PROD_VERSION}/notices.txt
# Couchbase proprietary end

# Customisation for entrypoints and run as a non-root user
COPY entrypoints/ /entrypoints/
COPY --from=grafana-official /run.sh /entrypoints/grafana.sh

ENV TINI_SUBREAPER=
COPY run.sh /run.sh
COPY scripts/collect-information.sh /collect-information.sh
RUN chmod a+x /run.sh /collect-information.sh /entrypoints/*

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
    mkdir -p /etc/prometheus/couchbase/custom && \
    mkdir -p /etc/cmos /tmp/support /tmp/loki/scratch && \
    chown -R couchbase:couchbase /entrypoints /logs /etc/prometheus/couchbase/ && \
    chown -R couchbase:couchbase /data /priv /ui /etc/cmos /tmp/support /tmp/loki

# Location of dynamic target information for Prometheus
ENV PROMETHEUS_DYNAMIC_INTERNAL_DIR=/etc/prometheus/couchbase/monitoring/
ENV PROMETHEUS_DYNAMIC_DIR=/etc/prometheus/couchbase/

# Add all useful and required information

COPY licenses/* /licenses/
# NOTE: this creates a chicken-and-egg effect - if we update notices, we need to run two builds to ensure it gets picked up
# TODO: this assumes couchbase-cluster-monitor and couchbase-observability-stack have the same version number
RUN curl -Lo /licenses/notices.txt \
    https://raw.githubusercontent.com/couchbase/product-metadata/master/couchbase-observability-stack/blackduck/${PROD_VERSION}/notices.txt && \
    mkdir -p /html/licenses && cp /licenses/* /html/licenses/ && chown -R couchbase:couchbase /html

COPY docs/modules/ROOT/pages/support.adoc /help.1

COPY git-commit.txt /etc/cmos-release.txt
RUN printf "Couchbase Observability Stack version ${PROD_VERSION}-${PROD_BUILD}\n%s" "$(cat /etc/cmos-release.txt)" > /etc/cmos-release.txt

USER       $CB_UID
ENTRYPOINT [ "/sbin/tini", "--", "/run.sh" ]
