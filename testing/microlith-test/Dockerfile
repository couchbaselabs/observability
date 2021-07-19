# Taken from https://github.com/bats-core/bats-detik/blob/master/Dockerfile (MIT license)
FROM alpine:3.14

ARG KUBECTL_VERSION=v1.21.2
ARG HELM_VERSION=v3.6.1
ARG BATS_VERSION=1.3.0
ARG BATS_DETIK_VERSION=1.0.0
ARG BATS_ASSERT_VERSION=2.0.0
ARG BATS_SUPPORT_VERSION=0.3.0
ARG BATS_FILE_VERSION=0.3.0

# Add packages
RUN apk --no-cache add \
    curl \
    git \
    libc6-compat \
    openssh-client \
    bash \
	gettext

# Install BATS
RUN curl -LO "https://github.com/bats-core/bats-core/archive/refs/tags/v$BATS_VERSION.zip" && \
	unzip -q -d /tmp "v$BATS_VERSION.zip" && \
	cd "/tmp/bats-core-$BATS_VERSION" && \
	./install.sh /usr/local && \
	rm -rf "/tmp/bats-core-$BATS_VERSION" && \
    rm -f "v$BATS_VERSION.zip"

# Install kubectl
RUN curl -LO "https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl" && \
	chmod +x kubectl && \
	mv kubectl /usr/local/bin/

# Install Helm
RUN curl -LO "https://get.helm.sh/helm-$HELM_VERSION-linux-amd64.tar.gz" && \
	mkdir -p "/usr/local/helm-$HELM_VERSION" && \
	tar -xzf "helm-$HELM_VERSION-linux-amd64.tar.gz" -C "/usr/local/helm-$HELM_VERSION" && \
	ln -s "/usr/local/helm-$HELM_VERSION/linux-amd64/helm" /usr/local/bin/helm && \
	rm -f "helm-$HELM_VERSION-linux-amd64.tar.gz"

# Install Detik - we use cd to expand the variables
RUN curl -LO "https://github.com/bats-core/bats-detik/archive/refs/tags/v$BATS_DETIK_VERSION.zip" && \
	unzip -q -d /tmp "v$BATS_DETIK_VERSION.zip" && \
    mkdir -p /usr/local/lib/bats-detik/ && \
    cd "/tmp/bats-detik-$BATS_DETIK_VERSION" && \
	mv -f lib/* /usr/local/lib/bats-detik/ && \
	rm -rf "/tmp/bats-detik-$BATS_DETIK_VERSION" && \
    rm -f "v$BATS_DETIK_VERSION.zip"

RUN curl -LO "https://github.com/bats-core/bats-assert/archive/refs/tags/v$BATS_ASSERT_VERSION.zip" && \
	unzip -q -d /tmp "v$BATS_ASSERT_VERSION.zip" && \
    mkdir -p /usr/local/lib/bats-assert/ && \
    cd "/tmp/bats-assert-$BATS_ASSERT_VERSION" && \
	mv -f * /usr/local/lib/bats-assert/ && \
	rm -rf "/tmp/bats-assert-$BATS_ASSERT_VERSION" && \
    rm -f "v$BATS_ASSERT_VERSION.zip"

RUN curl -LO "https://github.com/bats-core/bats-support/archive/refs/tags/v$BATS_SUPPORT_VERSION.zip" && \
	unzip -q -d /tmp "v$BATS_SUPPORT_VERSION.zip" && \
    mkdir -p /usr/local/lib/bats-support/ && \
    cd "/tmp/bats-support-$BATS_SUPPORT_VERSION" && \
	mv -f * /usr/local/lib/bats-support/ && \
	rm -rf "/tmp/bats-support-$BATS_SUPPORT_VERSION" && \
    rm -f "v$BATS_SUPPORT_VERSION.zip"

RUN curl -LO "https://github.com/bats-core/bats-file/archive/refs/tags/v$BATS_FILE_VERSION.zip" && \
	unzip -q -d /tmp "v$BATS_FILE_VERSION.zip" && \
    mkdir -p /usr/local/lib/bats-file/ && \
    cd "/tmp/bats-file-$BATS_FILE_VERSION" && \
	mv -f * /usr/local/lib/bats-file/ && \
	rm -rf "/tmp/bats-file-$BATS_FILE_VERSION" && \
    rm -f "v$BATS_FILE_VERSION.zip"

ENV BATS_DETIK_ROOT=/usr/local/lib/bats-detik
ENV BATS_FILE_ROOT=/usr/local/lib/bats-file
ENV BATS_SUPPORT_ROOT=/usr/local/lib/bats-support
ENV BATS_ASSERT_ROOT=/usr/local/lib/bats-assert

# Use the same UID as Jenkins:
# For Jenkins versions < 2.62, this is 1000
RUN adduser -D -u 10000 testing
USER testing
WORKDIR /home/testing

COPY --chown=testing . ./

# This is the image we want to test against, override to change.
ENV COS_IMAGE=couchbase/observability-stack:v1
ENV COUCHBASE_SERVER_IMAGE=couchbase/server:6.6.2

# Run native tests if set to true, run kubernetes tests otherwise
ENV TEST_NATIVE=true
# The kubernetes namespace to run tests in, useful to just destroy
ENV TEST_NAMESPACE=test
# Test root, not good to use current directory with scripts launched by BATS
ENV TEST_ROOT=/home/testing
# Helper for k8s resources root
ENV TEST_KUBERNETES_RESOURCES_ROOT=/home/testing/kubernetes/resources
# Extra customisation per test
ENV TEST_CUSTOM_CONFIG=test-custom-config

ENTRYPOINT [ "/bin/bash", "/home/testing/entrypoint.sh" ]
# Pass in `bash` and it will just run a shell for example, otherwise it will run tests.