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
# Caveat for all the above: cbmultimanager is private, but observability is
# public. To allow for building the latter without the former, some of the make
# recipes modify their behaviour if the `OSS` variable is set.

################################################################################
# Variables
# ---------
# These can be changed by the build system, or however you see fit.
################################################################################

PRODUCT := couchbase-observability-stack

# These are overidden by the build system, so need to be optional
# if undefined.  The build system also doesn't use -e to override
# so we need to be careful here.
VERSION ?= 0.0.0
BLD_NUM ?= 999

# This controls the build version of docker used.
# Note that this is a separate variable for cbmultimanager's builds.
GO_VERSION := 1.19.0

# The target controls what's built as regards cross compilation.
# These are similar to target triplets in the C world e.g. x86_64-unknown-linux.
# The syntax is <platform>-<os>-<arch>.
# <platform> is always "docker" right now - it's reserved for future use (e.g. OpenShift)
# <os> and <arch> are valid GOOS and GOARCH values respectively (e.g. linux/amd64)
HOSTARCH = $(shell go env GOHOSTARCH)
BINARY_TARGET := $(shell go env GOHOSTOS)-$(HOSTARCH)
IMAGE_TARGET := docker-linux-$(HOSTARCH)

# These are all the Docker images that we can produce.
# NOTE: when adding a new image, ensure you've asked Build Team to set up the
# registries beforehand, otherwise the build may break.
IMAGES := couchbase-observability-stack
ifndef OSS
IMAGES := $(IMAGES) couchbase-cluster-monitor
endif

###############################################################################
# Static/Generated Variables
# These shouldn't need to be touched in most circumstances.
###############################################################################

# Static configuration parameters.
BUILDDIR := build
ARTIFACTSDIR := dist
DOCSDIR := docs
# This must match manifest.xml (in couchbase/manifest).
UPSTREAMDIR := upstream
TMP_DOCS_DIR := microlith/docs
CMOSCFG_SRC_DIR := config-svc
CMOSCFG_TMP_DRC_DIR := microlith/config-svc

# This is the path where cbmultimanager is checked out.
# This should match the manifest.xml (in couchbase/manifest).
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

# These are the directories that need to exist for the build to work.
# NOTE: dist-dir is *not* here, as that confuses Make.
DIRECTORIES := $(BUILDDIR) microlith/bin

# Use GNU Tar where available
ifneq (, $(shell which gtar))
TAR := gtar
else
TAR := tar
endif

###############################################################################

# Ensure the Makefile is clean by disabling all implicit rules
.SUFFIXES:

# Generic development rule, will just build the image
.PHONY: all
all: container

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

# This target is special: it's invoked by the build system, and needs to
# prepare everything that will be archived.
# NOTE: it uses the BINARY_TARGET of the machine it's running on, which for
# developers will likely be darwin-amd64, but will be linux-amd64 in the build
# system, which can have confusing results. For this reason, it's not
# recommended to use it by hand.
# It's declared as phony, even though it's a real directory, to make it
# always rebuild.
.PHONY: dist
dist: dist-dir
ifndef OSS
	$(MAKE) -C $(CBMULTIMANAGER_PATH) dist -e $(BUILD_ENV)
# cbmultimanager also creates an image .tar.gz, which is useless - we'll
# produce one tarball with both images
	-rm dist/couchbase-cluster-monitor-image_$(VERSION)-$(BLD_NUM).tgz
endif
	$(MAKE) image-artifacts -e $(BUILD_ENV)


# This one builds the container images locally from the artifact archive.
# As a nice consequence, it also tests that the build system would be able to
# build them properly.
.PHONY: container
container: image-artifacts
	for archive in $(ARTIFACTSDIR)/*-image*.tgz; do \
		TAG=v1 tools/build-container-from-archive.sh "$$archive" $(HOSTARCH);\
	done

.PHONY: container-oss
container-oss:
	$(MAKE) -e OSS=true image-artifacts
	for archive in $(ARTIFACTSDIR)/*-image*.tgz; do \
		TAG=v1 tools/build-container-from-archive.sh "$$archive" $(HOSTARCH);\
	done

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
	tools/licence-lint-cbmultimanager.sh
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
# Image-related targets

# Finally, this creates the Docker image archives.
.PHONY: image-artifacts
image-artifacts: $(addprefix $(BUILDDIR)/images/,$(IMAGES)) | dist-dir
	tar -C $(BUILDDIR)/images -czf $(ARTIFACTSDIR)/$(PRODUCT)-image_$(VERSION)-$(BLD_NUM).tgz $(IMAGES)

ifndef OSS
$(BUILDDIR)/images/couchbase-cluster-monitor:
# NOTE: here `build` is really upstream/couchbase-cluster-monitor/build
# Trying to override it in its various recursive make invocations causes problems, so instead build
# it in its root and then copy it over
	$(MAKE) -C $(CBMULTIMANAGER_PATH) build/images/couchbase-cluster-monitor -e $(BUILD_ENV)
	cp -R $(CBMULTIMANAGER_PATH)/build/images/couchbase-cluster-monitor $@
	cp $(CBMULTIMANAGER_PATH)/docker/Dockerfile.couchbase-cluster-monitor $@/Dockerfile

$(BUILDDIR)/images/couchbase-observability-stack: \
	microlith/bin \
	microlith/bin/cbmultimanager-linux-amd64 \
	microlith/bin/cbeventlog-linux-amd64 \
	microlith/bin/cbmultimanager-linux-arm64 \
	microlith/bin/cbeventlog-linux-arm64 \
	microlith/entrypoints/cbmultimanager.sh \
	microlith/Dockerfile \
	microlith/docs \
	microlith/cbmultimanager-docs \
	microlith/config-svc \
	microlith/git-commit.txt
else
$(BUILDDIR)/images/couchbase-observability-stack: \
	microlith/Dockerfile.oss \
	microlith/docs \
	microlith/config-svc \
	microlith/git-commit.txt
endif
	mkdir -p $@
	cp -r microlith/* $@
ifdef OSS
	rm $@/Dockerfile
	mv $@/Dockerfile.oss $@/Dockerfile
endif

microlith/Dockerfile.oss: microlith/Dockerfile
	sed '/^# Couchbase proprietary start/,/^# Couchbase proprietary end/d' $< > $@

###############################################################################
# Image dependencies

ifndef OSS
microlith/bin/cbmultimanager-%:
	$(MAKE) -C $(CBMULTIMANAGER_PATH) $(BUILDDIR)/cbmultimanager-$* -e $(BUILD_ENV) -e BINARY_TARGET=$*
	cp $(CBMULTIMANAGER_PATH)/build/cbmultimanager-$* $@

microlith/entrypoints/cbmultimanager.sh: $(CBMULTIMANAGER_PATH)/docker/couchbase-cluster-monitor-entrypoint.sh
	cp $< $@

microlith/bin/cbeventlog-%:
	$(MAKE) -C $(CBMULTIMANAGER_PATH) $(BUILDDIR)/cbeventlog-$* -e $(BUILD_ENV) -e BINARY_TARGET=$*
	cp $(CBMULTIMANAGER_PATH)/build/cbeventlog-$* $@
endif

microlith/docs: $(wildcard $(DOCSDIR/**))
	cp -R $(DOCSDIR) $@

ifndef OSS
microlith/cbmultimanager-docs: $(wildcard $(CBMULTIMANAGER_PATH/docs/**))
	cp -R $(CBMULTIMANAGER_PATH)/docs $@
endif

microlith/config-svc: $(wildcard $(CMOSCFG_SRC_DIR/**))
	cp -R $(CMOSCFG_SRC_DIR) $@

microlith/git-commit.txt:
	echo "Version: $(version)" >> microlith/git-commit.txt
	echo "Build: $(productVersion)" > microlith/git-commit.txt
	echo "Revision: $(revision)" >> microlith/git-commit.txt
	echo "Git commit: $(GIT_REVISION)" >> microlith/git-commit.txt

# Helper to make any directories required (except `dist` itself).
$(DIRECTORIES):
	mkdir -p $@

.PHONY: dist-dir
dist-dir:
	mkdir -p $(ARTIFACTSDIR)