FROM grafana/grafana:7.5.2 as grafana-official
FROM grafana/loki:2.0.0 as loki-official
FROM prom/prometheus:v2.27.1 as prometheus-official
FROM prom/alertmanager:v0.22.1 as alertmanager-official
FROM prom/pushgateway:v1.4.0 as pushgateway-official
FROM prom/node-exporter:v1.1.2 as node-exporter-official

FROM golang:1.16 as healthcheck-builder

RUN apt-get update && \
    apt-get install -y sqlcipher libssl-dev openssl openssh-client git && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /src/github.com/couchbaselabs/cbmultimanager/ /bin

# Support cloning the private repo
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN --mount=type=ssh git clone --depth 1 git@github.com:couchbaselabs/cbmultimanager.git /src/github.com/couchbaselabs/cbmultimanager/

WORKDIR /src/github.com/couchbaselabs/cbmultimanager

# Statically build it to allow reuse on Alpine Linux
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o /bin/cbmultimanager ./cmd/cbmultimanager
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o /bin/cbeventlog ./cmd/cbeventlog

FROM alpine:3.13 as fluentbit-builder
RUN apk add --no-cache build-base gcc abuild binutils cmake git flex bison fts-dev gtest-dev zlib-dev

ARG FLUENT_BIT_VER=1.7.7
RUN mkdir -p /src/fluent-bit && \
    git clone --depth 1 --branch v${FLUENT_BIT_VER} https://github.com/fluent/fluent-bit.git /src/fluent-bit

WORKDIR /src/fluent-bit

COPY fluent-bit/*.patch ./
RUN git apply *.patch && \
    cmake -B build \
		-DCMAKE_INSTALL_PREFIX=/fluent-bit \
		-DCMAKE_BUILD_TYPE=None \
		-DFLB_CORO_STACK_SIZE=24576 \
        -DFLB_RELEASE=On \
        -DFLB_SHARED_LIB=No \
		-DFLB_TESTS_INTERNAL=No \
		-DFLB_TESTS_RUNTIME=No \
		-DFLB_TLS=Yes \
		-DFLB_HTTP_SERVER=Yes \
         . && \
    make -C build install

# Combined image
FROM alpine:3.13

RUN apk add --no-cache ca-certificates bash libcap openssl musl-utils tzdata tini curl libgcc fts

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

WORKDIR /

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

# PROM PUSH GATEWAY
# From https://github.com/prometheus/pushgateway/blob/master/Dockerfile
COPY --from=pushgateway-official /bin/pushgateway /bin/pushgateway

EXPOSE 9091
RUN mkdir -p /pushgateway && chown nobody:nobody /pushgateway

# NODE EXPORTER
# From https://github.com/prometheus/node_exporter/blob/master/Dockerfile
COPY --from=node-exporter-official /bin/node_exporter /bin/node_exporter
EXPOSE      9100

# Customisation to support node exporter access to proc/sys/rootfs: mount here and/or set to wherever mounted.
ENV NODE_EXPORTER_HOST_PROC="/node-exporter/host/proc"
ENV NODE_EXPORTER_HOST_ROOTFS="/node-exporter/host/rootfs"
ENV NODE_EXPORTER_HOST_SYS="/node-exporter/host/sys"
# To provide node exporter custom collector output if needed:
ENV NODE_EXPORTER_CUSTOM_COLLECTOR="/node-exporter/custom/"

# Add support for auto-provisioning now
COPY grafana/provisioning/ /etc/grafana/provisioning/
COPY grafana/cblogo.png /usr/share/grafana/public/img/cblogo.png
ENV GF_SECURITY_ADMIN_PASSWORD=password
ENV GF_USERS_ALLOW_SIGN_UP=false

COPY prometheus/ /etc/prometheus/
COPY alertmanager/ /etc/alertmanager/

# Couchbase healthcheck
RUN mkdir -p /data /priv && \
    openssl req -x509 -nodes -days 365 -subj "/C=CA/ST=QC/O=Couchbase, Inc./CN=couchbase.com" -addext "subjectAltName=DNS:couchbase.com" -newkey rsa:2048 -keyout /priv/server.key -out /priv/server.crt;

COPY --from=healthcheck-builder /bin/cbmultimanager /bin/cbmultimanager
COPY --from=healthcheck-builder /bin/cbeventlog /bin/cbeventlog
RUN chmod a+x /bin/cbmultimanager /bin/cbeventlog

EXPOSE 7196 7197

# Golang Fluent Bit
COPY --from=fluentbit-builder /fluent-bit/ fluent-bit/
COPY fluent-bit/fluent-bit.conf.microlith /fluent-bit/etc/fluent-bit.conf
RUN mkdir -p /opt/couchbase/var/lib/couchbase/logs/ && chmod 777 /opt/couchbase/var/lib/couchbase/logs/
ENV COUCHBASE_LOGS=/opt/couchbase/var/lib/couchbase/logs/
EXPOSE 22020

# Customisation for entrypoints and run as a non-root user
COPY entrypoints/ /entrypoints/
COPY --from=grafana-official /run.sh /entrypoints/grafana.sh
COPY --from=pushgateway-official /bin/pushgateway /entrypoints/pushgateway

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
    chown -R couchbase:couchbase /pushgateway && \
    chown -R couchbase:couchbase /data /priv && \
    chown -R couchbase:couchbase /opt/couchbase/var/lib/couchbase/logs/ && \
    chown -R couchbase:couchbase /entrypoints

USER       $CB_UID
ENTRYPOINT [ "/sbin/tini", "--", "/run.sh" ]