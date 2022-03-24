################################################################################
# CMOS Build Process
################################################################################

# To understand the way this Makefile works, and the reasoning for it, there are
# a few important concepts to understand relating to how this fits into the
# wider Couchbase build process. Couchbase employees can also refer to
# https://hub.internal.couchbase.com/confluence/display/CR/Grand+Unified+Build+and+Release+Process+for+Operator
#
# Essentially, for Docker images, the build happens in two phases, which for the
# purposes of these comments we will refer to as "compile" and "assemble". The
# gist of it is:
# 1. Build system checks out the couchbase-observability-stack manifest.xml
#    and runs `make dist`. This is the "compile" stage.
# 2. `make dist` creates a directory called `dist` and places its artifacts there.
#    These can be either:
#    a) Artifacts to release directly as they are - these must contain
#       `${VERSION}-${BUILD_NUMBER}` somewhere in the file name.
#    b) One or more .tgz files named `${PRODUCT}-image_${VERSION}-${BUILD_NUMBER}.tgz` -
#       these must contain all the files needed to build a Docker image,
#       including a Dockerfile.
#    These artifacts are all uploaded to the (internal) build server.
# 3. The .tgz image files are passed to a separate job which unpacks them,
#    runs `docker build`, and uploads the generated images to an internal registry.
#    This is the "assemble" stage.
#
# One other important consideration is that this same Makefile is used to run
# the builds for both couchbase/observability-stack *and* couchbase/cluster-monitor.
# The reason is that both of these require the `cbmultimanager` binary, and
# we need to ensure that the same binary goes into both (otherwise the builds
# become non-reproducible).
#

###############################################################################

# That's the requirements of the build system. Now we come to some self-imposed
# requirements.
#
# Caveat for all the above: cbmultimanager is private, but observability is
# public. To allow for building the latter without the former, some of the make
# recipes modify their behaviour if the `OSS` variable is set.
#
# First, the binaries. All the binaries in both couchbase-cluster-monitor and
# couchbase-observability-stack have their `package main` in a folder named
# `*/cmd/$BINARY`. We will build these  and copy them to $ARTIFACTSDIR.
#
# Linux builds (for now the only kind) are always done inside Docker. This is
# done to ensure the builds stay reproducible. It *is* possible to manually
# build a binary, e.g. for development, but this should not be used for any
# builds anywhere other than a developer's laptop.
#
# To minimise duplication, these binary builds are done using the same
# Dockerfile for all the binaries ()

################################################################################
# Variables
# ---------
# These can be changed by the build system, or however you see fit.
################################################################################

# Product defines the product/application name, and has a bearing on
# what the package artifacts are called.
PRODUCT := couchbase-observability-stack

# These are overidden by the build system, so need to be optional
# if undefined.  The build system also doesn't use -e to override
# so we need to be careful here.
VERSION ?= 0.0.0
BLD_NUM ?= 999

# This controls the build version of docker used.
# The only caveat, is the build system doesn't use this as the source
# of truth, so you'll want to update the defaults found in ./docker/...
GO_VERSION := 1.17.2

# The target controls what's built as regards cross compilation.
# These are similar to target triplets in the C world e.g. x86_64-unknown-linux.
# The syntax is <platform>-<os>-<arch>.
# <platform> is always "docker" right now - it's reserved for future use (e.g. OpenShift)
# <os> and <arch> are valid GOOS and GOARCH values respectively (e.g. linux/amd64)
BINARY_TARGET := $(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)
IMAGE_TARGET := docker-linux-amd64

# These are all the Docker images that we can produce
IMAGES := couchbase-observability-stack

ifndef OSS
IMAGES := $(IMAGES) couchbase-cluster-monitor
endif

# These are all the static binaries that we can produce.
BINARIES := 
ifndef OSS
BINARIES := $(BINARIES) cbmultimanager cbeventlog cbhealthagent
endif

###############################################################################
# Static/Generated Variables
# These shouldn't need to be touched in most circumstances.
###############################################################################

# Static configuration parameters.
BUILDDIR := build
ARTIFACTSDIR := dist
DOCSDIR := docs
UPSTREAMDIR := upstream
TMP_DOCS_DIR := microlith/docs
CMOSCFG_SRC_DIR := config-svc
CMOSCFG_TMP_DRC_DIR := microlith/config-svc

# This is the path where cbmultimanager is checked out.
# This should match the manifest.xml.
CBMULTIMANAGER_PATH := $(UPSTREAMDIR)/cbmultimanager

# Extract the various components of the image target
IMAGE_PLATFORM := $(word 1,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_OS := $(word 2,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_ARCH := $(word 3,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_BINARY_TARGET := $(IMAGE_OS)-$(IMAGE_ARCH)

###############################################################################
# Dynamic/Derived Variables
# These shouldn't need to be touched by hand at all.
###############################################################################

# Variable for propagating build arguments.
BUILD_ENV := VERSION=$(VERSION) BLD_NUM=$(BLD_NUM) UPSTREAMDIR=$(abspath $(UPSTREAMDIR)) ARTIFACTSDIR=$(abspath $(ARTIFACTSDIR))

# These are the directories that need to exist for the build to work
# Note: ARTIFACTSDIR is not here, because it'd conflict with the phony `dist` target
DIRECTORIES := $(BUILDDIR) microlith/bin # FIXME variable

# Use GNU Tar where available
ifneq (, $(shell which gtar))
TAR := gtar
else
TAR := tar
endif

###############################################################################

# Ensure the Makefile is clean by disabling all implicit rules
.SUFFIXES:

# Clean up any potential mess
.PHONY: clean
clean: images-clean
	rm -rf $(ARTIFACTSDIR) $(BUILDDIR) microlith/bin microlith/cbmultimanager-docs microlith/docs microlith/config-svc
	-rm microlith/git-commit.txt
ifndef oss
	rm -rf microlith/cbmultimanager-docs
	$(MAKE) -C $(CBMULTIMANAGER_PATH) -e $(BUILD_ENV) clean
endif

.PHONY: images-clean
images-clean:
	-docker rmi couchbase/observability-stack:v1
	-docker rmi couchbase/observability-stack-oss:v1

.PHONY: dist
dist: image-artifacts
ifndef OSS
	$(MAKE) -C $(CBMULTIMANAGER_PATH) dist -e $(BUILD_ENV)
endif

# This one builds the container images locally.
# As a nice consequence, it also tests that the build system would be able to
# build them properly.
.PHONY: container
container: dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz
	tools/build-container-from-archive.sh dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz couchbase/observability-stack:v1

.PHONY: container-oss
container-oss:
	$(MAKE) -e OSS=true image-artifacts
	tools/build-container-from-archive.sh dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz couchbase/observability-stack:v1

######################################################################################
# Testing-related targets

# NOTE: on Ansible linting failure due to YAML formatting, a pre-commit hook can be used to autoformat: https://pre-commit.com/
# Install pre-commit then run: pre-commit run --all-files
.PHONY: Lint
lint: container-lint
	tools/asciidoc-lint.sh
	tools/shellcheck.sh
	ansible-lint
	tools/licence-lint.sh
	tools/dashboards-lint.sh
	tools/rules-lint.sh
	docker run --rm -i -v  ${PWD}/config-svc:/app -w /app golangci/golangci-lint:v1.42.1 golangci-lint run -v

.PHONY: test-loki-rules
test-loki-rules:
	testing/loki_alerts/run_all.sh

.PHONY: container-lint
	tools/hadolint.sh

.PHONY: container-scan
container-scan: container
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image \
		--severity "HIGH,CRITICAL" --ignore-unfixed --exit-code 1 --no-progress \
		couchbase/observability-stack:$(VERSION)-$(BLD_NUM)
	-docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -e CI=true wagoodman/dive \
		couchbase/observability-stack:$(VERSION)-$(BLD_NUM)

.PHONY: test-kubernetes
test-kubernetes: TEST_SUITE ?= integration/kubernetes
test-kubernetes:
	# TODO (CMOS-97): no smoke suite for kubernetes yet
	testing/run-k8s.sh ${TEST_SUITE}

.PHONY: test-containers
test-containers:
	testing/run-containers.sh ${TEST_SUITE}

.PHONY: test-native
test-native:
	testing/run-native.sh ${TEST_SUITE}

.PHONY: test
test: clean test-unit container-oss test-native test-containers test-kubernetes

.PHONY: test-unit
test-unit:
	DOCKER_BUILDKIT=1 docker build --target=unit-test config-svc/

.PHONY: generate-screenshots
generate-screenshots: container-oss
	tools/generate-screenshots.sh

.PHONY: docs
docs:
	# || true is needed so the Makefile does not error when hitting CTRL+C
	(docker-compose -f docs/docker-compose.yml up || true) && docker-compose -f docs/docker-compose.yml down

######################################################################################

.PHONY: image-artifacts
image-artifacts: dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz

ifndef OSS
dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz: \
	microlith/bin \
	microlith/bin/cbmultimanager-$(IMAGE_BINARY_TARGET) \
	microlith/bin/cbeventlog-$(IMAGE_BINARY_TARGET) \
	microlith/entrypoints/cbmultimanager.sh \
	microlith/Dockerfile \
	microlith/docs \
	microlith/cbmultimanager-docs \
	microlith/config-svc \
	microlith/git-commit.txt \
	| dist-dir
else
dist/couchbase-observability-stack-image_$(VERSION)-$(BLD_NUM).tgz: \
	microlith/Dockerfile.oss \
	microlith/docs \
	microlith/config-svc \
	microlith/git-commit.txt \
	| dist-dir
endif
ifdef OSS
	$(eval TARFLAGS := --exclude Dockerfile --transform='flags=r;s|Dockerfile.oss|Dockerfile|')
endif
	$(TAR) $(TARFLAGS) -C microlith -czf $@ $(foreach file,$(shell echo microlith/*),$(notdir $(file)))

ifndef OSS
microlith/bin/cbmultimanager-$(IMAGE_BINARY_TARGET):
	$(MAKE) -C $(CBMULTIMANAGER_PATH) $(BUILDDIR)/cbmultimanager-$(IMAGE_BINARY_TARGET) -e $(BUILD_ENV) -e BINARY_TARGET=$(IMAGE_BINARY_TARGET)
	cp $(CBMULTIMANAGER_PATH)/build/cbmultimanager-$(IMAGE_BINARY_TARGET) $@

microlith/bin/cbeventlog-$(IMAGE_BINARY_TARGET):
	$(MAKE) -C $(CBMULTIMANAGER_PATH) $(BUILDDIR)/cbeventlog-$(IMAGE_BINARY_TARGET) -e $(BUILD_ENV) -e BINARY_TARGET=$(IMAGE_BINARY_TARGET)
	cp $(CBMULTIMANAGER_PATH)/build/cbeventlog-$(IMAGE_BINARY_TARGET) $@
endif


microlith/Dockerfile.oss: microlith/Dockerfile
	sed '/^# Couchbase proprietary start/,/^# Couchbase proprietary end/d' $< > $@


microlith/docs: $(wildcard $(DOCSDIR/**))
	cp -R $(DOCSDIR) $@

ifndef OSS
microlith/cbmultimanager-docs: $(wildcard $(CBMULTIMANAGER_PATH/docs/**))
	cp -R $(CBMULTIMANAGER_PATH)/docs $@

microlith/entrypoints/cbmultimanager.sh: $(CBMULTIMANAGER_PATH)/docker/couchbase-cluster-monitor-entrypoint.sh
	cp $< $@
endif

microlith/config-svc: $(wildcard $(CMOSCFG_SRC_DIR/**))
	cp -R $(CMOSCFG_SRC_DIR) $@

microlith/git-commit.txt:
	echo "Version: $(version)" >> microlith/git-commit.txt
	echo "Build: $(productVersion)" > microlith/git-commit.txt
	echo "Revision: $(revision)" >> microlith/git-commit.txt
	echo "Git commit: $(GIT_REVISION)" >> microlith/git-commit.txt

# This is slightly funky because `dist` above is a phony target, but is *also*
# the name of a real directory.
.PHONY: dist-dir
dist-dir:
	mkdir -p $(ARTIFACTSDIR)

# Helper to make any directories required (except `dist` itself).
$(DIRECTORIES):
	mkdir -p $@
