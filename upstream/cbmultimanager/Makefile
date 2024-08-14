################################################################################
# CMOS Build Process
################################################################################

# To understand the way this Makefile works, and the reasoning for it, there are
# a few important concepts to understand relating to how this fits into the
# wider Couchbase build process. To best understand this, open the below wiki
# page alongside this README while reading it:
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

# One other important consideration is that this Makefile is also used by
# the Makefile of couchbaselabs/observability (a.k.a.
# couchbase-observability-stack). This is done because the observability Docker
# image also needs the cbmultimanager binary. We need to ensure that any changes
# to this repo also trigger a rebuild of the CMOS microlith image, and also if
# (for whatever reason) we need to do a rebuild, it uses the exact same version
# of cbmultimanager as the initial build did.
#
# For this reason, you will see the observability Makefile recursively invoke
# this Makefile, overriding some fo the variables (namely VERSION, BLD_NUM,
# and some of the output paths, e.g. ARTIFACTSDIR).
#

# Linux binary builds are always done inside Docker. This is done to ensure the
# builds stay reproducible and don't depend on any build environment files,
# as well as to cross-compile. It *is* possible to manually build a binary, e.g.
# for development, but this should not be used for any builds anywhere other
# than a developer's laptop.
#
# Non-Linux builds invoke `go build` directly, under the assumption that the
# system on which they are running is already set up with all the requirements.
# As a consequence, they use the GOOS and GOARCH of the host.
#
# To minimise duplication, these binary builds are done using the same
# Dockerfile for all the binaries (docker/Dockerfile.build), which recursively
# invokes this Makefile. This can get a little confusing, so we establish a
# naming convention: goals that are invoked by this "builder" Dockerfile
# are referred to as "intermediate" goals, while goals that are invoked either
# by a human or the build system are referred to as "target" goals.
#
# Intermediate and target goals also differ in where they place their
# results: intermediate binaries are placed in $INTERMEDIATE_BINDIR, while target
# binaries are first placed in $TARGET_BINDIR, and then copied to $ARTIFACTSDIR,
# following the naming convention that the build system will expect when it
# archives the binaries. For example, the cbmultimanager binary starts life as
# build/intermediate/linux-amd64/cbmultimanager, then gets copied to
# build/cbmultimanager-linux-amd64, and finally
# dist/cbmultimanager-linux-amd64-0.0.0-999.

################################################################################
# Variables
# ---------
# These are "semi-constants" - they're not constants, since they can be
# overridden by the build system, but they should stay the same for the life
# of a single build cycle.
################################################################################
PRODUCT := couchbase-cluster-monitor

# These are overidden by the build system, so need to be optional
# if undefined.  The build system also doesn't use -e to override
# so we need to be careful here.
VERSION ?= 0.0.0
BLD_NUM ?= 999

# This controls the versions of the tools used at build-time
GO_VERSION := 1.19
NODE_VERSION := 16
NFPM_VERSION := 2.15.1

# The target controls what's built as regards cross compilation.
# These are similar to target triplets in the C world e.g. x86_64-unknown-linux.
# The syntax is <platform>-<os>-<arch> for images, and <os>-<arch> for binaries.
# <platform> is always "docker" right now - it's reserved for future use (e.g.
# OpenShift).
# <os> and <arch> are valid GOOS and GOARCH values respectively (e.g. linux/amd64)
HOSTARCH := $(shell go env GOHOSTARCH)
BINARY_TARGET := $(shell go env GOHOSTOS)-$(HOSTARCH)
IMAGE_TARGET := docker-linux-$(shell go env GOHOSTARCH)
BINARY_DISTRO := centos

# These are all the Docker images that we can produce.
# NOTE: when adding a new image, ensure you've asked Build Team to set up the
# registries beforehand, otherwise the build may break.
IMAGES := couchbase-cluster-monitor

# These are all the static binaries that we can produce.
BINARIES := cbmultimanager cbeventlog cbhealthagent

# These are all the .deb/.rpm packages that we can produce.
PACKAGES := cbhealthagent

###############################################################################
# Static/Generated Variables
# These shouldn't need to be touched in most circumstances.
###############################################################################

# Flags for the Go linker.
# Note that these are added to / overridden depending on the platform - see later
BASE_LDFLAGS := \
  -X github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta.Version=$(VERSION) \
  -X github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta.BuildNumber=$(BLD_NUM) \
  -linkmode external

# Static configuration parameters.
# NOTE: if this is a couchbase-observability-stack build, some of these will be
# overridden.
BUILDDIR := build
ARTIFACTSDIR := dist
DOCKERDIR := docker
UPSTREAMDIR := $(abspath ..)

###############################################################################
# Dynamic/Derived Variables
# These shouldn't need to be touched by hand.
###############################################################################

# This is where built binaries are placed by the "target" steps.
TARGET_BINDIR := $(BUILDDIR)
INTERMEDIATE_BINDIR := $(BUILDDIR)/intermediate/$(BINARY_TARGET)

# Break apart the target variables into something more useful
BINARY_OS := $(word 1,$(subst -, ,$(BINARY_TARGET)))
BINARY_ARCH := $(word 2,$(subst -, ,$(BINARY_TARGET)))
BINARY_PLATFORM := $(BINARY_OS)/$(BINARY_ARCH)

IMAGE_PLATFORM := $(word 1,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_OS := $(word 2,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_ARCH := $(word 3,$(subst -, ,$(IMAGE_TARGET)))
IMAGE_BINARY_TARGET := $(IMAGE_OS)-$(IMAGE_ARCH)


# All binary builds get these arguments, acting as a stable interface between
# the make system and any binaries emitted.  Thus the correct version is
# propagated, the correct compiler to use, and what target to build for.
DOCKER_BUILD_ARGS := \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg NODE_VERSION=$(NODE_VERSION) \
	--build-arg VERSION=$(VERSION) \
	--build-arg BLD_NUM=$(BLD_NUM) \
  	--build-arg BINARY_TARGET=$(BINARY_TARGET) \
	--build-arg BINARY_DISTRO=$(BINARY_DISTRO) \
	--build-arg BINARY_ARCH=$(BINARY_ARCH) \
	--build-arg IMAGE_BINARY_TARGET=$(IMAGE_BINARY_TARGET) \
	--build-arg NFPM_VERSION=$(NFPM_VERSION) \
	--build-arg BUILD_PATH=/tmp/cmos-build \
	--progress plain

# Variable for propagating build arguments.
BUILD_ENV := \
	VERSION=$(VERSION) \
	BLD_NUM=$(BLD_NUM) \
	ARTIFACTSDIR=$(ARTIFACTSDIR) \
	DOCKERDIR=$(DOCKERDIR) \
	UPSTREAMDIR=$(UPSTREAMDIR) \
	BINARY_TARGET=$(BINARY_TARGET)

# Finally, these are all the directories that need to exist for the build system to work.
DIRECTORIES := \
	$(INTERMEDIATE_BINDIR) \
	$(TARGET_BINDIR)

################################################################################
# GOALS
# First, the top-level goals that a human (or the build system) is likely to
# invoke.
################################################################################

# Ensure the Makefile is clean by disabling all implicit rules
.SUFFIXES:

# This is what will run if you just run `make`. It'll build the binaries and
# a Docker image.
.PHONY: all
all: binary-artifacts container

# Helper to clean up any potential mess
.PHONY: clean
clean:
	rm -rf $(BUILDDIR) $(ARTIFACTSDIR) ui/dist/app

# Recursively call make dist twice
.PHONY: dist
dist:
	$(MAKE) dist-inter -e BINARY_TARGET=linux-arm64 IMAGE_BINARY_TARGET=linux-arm64
	$(MAKE) dist-inter -e BINARY_TARGET=linux-amd64 IMAGE_BINARY_TARGET=linux-amd64
	-rm dist/couchbase-cluster-monitor-image_$(VERSION)-$(BLD_NUM).tgz
	$(foreach req,$(dist-image-requirements-for.couchbase-cluster-monitor),cp $(req) $(BUILDDIR)/images/couchbase-cluster-monitor/$(notdir $(req)) && ) true
	tar -C $(BUILDDIR)/images -cvzf $(ARTIFACTSDIR)/$(PRODUCT)-image_$(VERSION)-$(BLD_NUM).tgz $(IMAGES)

dist-image-requirements-for.couchbase-cluster-monitor := \
	$(BUILDDIR)/cbmultimanager-linux-amd64\
	$(BUILDDIR)/cbeventlog-linux-amd64\
	$(BUILDDIR)/cbmultimanager-linux-arm64\
	$(BUILDDIR)/cbeventlog-linux-arm64\
	$(BUILDDIR)/couchbase-cluster-monitor-entrypoint.sh\
	$(BUILDDIR)/LICENSE

# This target is special: it's invoked by the build system, and needs to
# prepare everything that will be archived.
# NOTE: it uses the BINARY_TARGET of the machine it's running on, which for
# developers will likely be darwin-amd64, but will be linux-amd64 in the build
# system, which can have confusing results. For this reason, it's not
# recommended to use it by hand.
# It's declared as phony, even though it's a real directory, to make it
# always rebuild.
.PHONY: dist-inter
dist-inter: dist-dir
	$(MAKE) image-artifacts binary-artifacts package-artifacts -e $(BUILD_ENV)
	$(MAKE) image-artifacts binary-artifacts package-artifacts -e $(BUILD_ENV) BINARY_DISTRO=debian


# This builds the container images locally, using the image-artifacts.
# As a nice consequence, it also tests that the build system would be able to
# build them properly.
.PHONY: container
container: image-artifacts
	for archive in $(ARTIFACTSDIR)/*-image*.tgz; do \
		TAG=v1 tools/build-container-from-archive.sh "$$archive" $(HOSTARCH);\
	done

# Helper to make any directories required (except `dist` itself).
$(DIRECTORIES):
	mkdir -p $@

.PHONY: dist-dir
dist-dir:
	mkdir -p $(ARTIFACTSDIR)

# Run various linters over the code.
# NOTE: Jenkins does not use this goal, instead the linters that are run in CV
# are specified in jenkins/Jenkinsfile.
# hadolint: ignored warnings for pinning packages of each docker image.
.PHONY: lint
lint:
	docker run --mount type=bind,source="$(PWD)"/docker,target=/docker  --rm -i hadolint/hadolint:latest-debian \
	hadolint `ls docker/Dockerfile.*` --ignore DL3008 --ignore DL3033 --ignore DL3018 -t warning
	docker run --rm -i -v ${PWD}:/work -w /work golangci/golangci-lint:v1.42.1 golangci-lint run -v
	tools/shellcheck.sh
	tools/licence-lint.sh
	go run ./tools/validate-checker-docs.go

# Run security scanners and size analysis on our Docker image.
.PHONY: container-scan
container-scan: container
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image \
		--severity "HIGH,CRITICAL" --ignore-unfixed --exit-code 1 --no-progress \
		couchbase/cluster-monitor:$(VERSION)-$(BLD_NUM)
	-docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -e CI=true wagoodman/dive \
		couchbase/cluster-monitor:$(VERSION)-$(BLD_NUM)

# Utility target to run generated code
# Docs runs its own `go generate` because `./...` won't match because of build tags
.PHONY: generate
generate: docs/modules/ROOT/pages/checkers.adoc
	go generate ./...

################################################################################
# All the binaries  have their `package main` in a folder named
# `*/cmd/$BINARY`. We will build these  and copy them to $ARTIFACTSDIR.
#
# See the comment at the top of the file for some explanatory notes.

# This target builds all the binaries.
.PHONY: binary-artifacts
binary-artifacts: $(addprefix $(ARTIFACTSDIR)/,$(addsuffix -$(BINARY_TARGET)-$(VERSION)-$(BLD_NUM),$(BINARIES)))

# This step just copies the "target" binary into its proper place in ARTIFACTSDIR.
$(ARTIFACTSDIR)/%-$(BINARY_TARGET)-$(VERSION)-$(BLD_NUM): $(TARGET_BINDIR)/%-$(BINARY_TARGET) dist-dir $(TARGET_BINDIR)
	cp $< $@

# This is a "target" binary goal.
# For Linux it runs the intermediate goal inside Docker.
# For other OSes it runs the goal directly, under the assumption that we're on that OS.
#
# cbmultimanager specifically needs the UI to be built first, as that requires NPM and
# can't be done in the generic Dockerfile.build.
#
# Ensure they don't get deleted if running as an intermediate target.
.PRECIOUS: $(TARGET_BINDIR)/%-$(BINARY_TARGET)
$(TARGET_BINDIR)/cbmultimanager-$(BINARY_TARGET): ui/dist/app
$(TARGET_BINDIR)/%-$(BINARY_TARGET): | $(TARGET_BINDIR) $(INTERMEDIATE_BINDIR)
ifeq ($(BINARY_OS),linux)
	$(eval DOCKER_BUILD_IMAGE := cmos-build-$*)
	DOCKER_BUILDKIT=1 docker build \
		-f $(DOCKERDIR)/Dockerfile.build \
		-t $(DOCKER_BUILD_IMAGE) \
		--platform=$(BINARY_PLATFORM) \
		--build-arg GOAL=$(INTERMEDIATE_BINDIR)/$* \
		$(DOCKER_BUILD_ARGS) \
		.
	docker run --rm --entrypoint /bin/cat \
		$(DOCKER_BUILD_IMAGE) \
		/tmp/cmos-build/$(INTERMEDIATE_BINDIR)/$* > $(INTERMEDIATE_BINDIR)/$*
	docker rmi -f $(DOCKER_BUILD_IMAGE)
else
	$(MAKE) -C $(CURDIR) -e $(BUILD_ENV) $(INTERMEDIATE_BINDIR)/$*
endif
	chmod +x $(INTERMEDIATE_BINDIR)/$*
	cp $(INTERMEDIATE_BINDIR)/$* $@
	

# This is an "intermediate" binary goal, which just runs `go build`.
# Do not call this by hand.
# The first prerequisite MUST be the path to the main package for that binary.
$(INTERMEDIATE_BINDIR)/cbhealthagent: agent/cmd/cbhealthagent $(shell find . -type f -name '*.go') | $(INTERMEDIATE_BINDIR)
$(INTERMEDIATE_BINDIR)/cbmultimanager: cluster-monitor/cmd/cbmultimanager $(shell find . -type f -name '*.go') | $(INTERMEDIATE_BINDIR)
$(INTERMEDIATE_BINDIR)/cbeventlog: cluster-monitor/cmd/cbeventlog $(shell find . -type f -name '*.go') | $(INTERMEDIATE_BINDIR)
$(INTERMEDIATE_BINDIR)/%:
# HERE BE DRAGONS!
# This is a shell `if`, not a make `ifeq`, because $@ would be evaluated
# too early and not have the appropriate value.
# When it comes to the actual flag values: ideally we'd use one set of Go flags
# for all the binaries, but cbhealthagent needs cgo on darwin (but not on any
# other platforms, where we'd ideally keep it disabled just in case). On Linux,
# however, it is apparently allergic to `-extldflags=-static` and will
# runtime.abort() (read: SIGTRAP) while setting up thread-local storage (?!)
# if statically built on linux. cbmultimanager linked statically, meanwhile, runs
# just fine on Linux. Darwin is also allergic to it, but will simply fail to link
# the binary rather than aborting at runtime.
#
# So the matrix we need is:
# cbmultimanager, mac => cgo, dynamic
# cbmultimanager, linux => cgo, static
# cbhealthagent, mac => cgo, dynamic
# anything else => no cgo
	export GOOS=$(BINARY_OS) GOARCH=$(BINARY_ARCH);\
	export GO_LDFLAGS='$(BASE_LDFLAGS)';\
	if [ "$(notdir $@)" = "cbmultimanager" ] || [ "$(notdir $@)" = "cbhealthagent" ]; then \
		export CGO_ENABLED=1 ;\
		if [ "$(notdir $@)" = "cbmultimanager" ] && [ "$(BINARY_OS)" != "darwin" ]; then \
			export GO_LDFLAGS="$$GO_LDFLAGS -extldflags=-static";\
		fi;\
	else \
		export CGO_ENABLED=0 ;\
	fi;\
	if [ "$(HOSTARCH)" != "arm64" ] && [ "$(BINARY_ARCH)" = "amd64" ] && [ "$(BINARY_OS)" = "linux" ]; then \
		export CC=x86_64-linux-gnu-gcc ;\
	fi;\
	if [ "$(HOSTARCH)" != "amd64" ] && [ "$(BINARY_ARCH)" = "arm64" ] && [ "$(BINARY_OS)" = "linux" ]; then \
		export CC=aarch64-linux-gnu-gcc ;\
	fi;\
	echo "Build environment: $$(env | grep GO)";\
	go build $(GOBUILDFLAGS) -tags netgo,sqlite_omit_load_extension -o $@ -ldflags "$$GO_LDFLAGS" ./$<

# cbmultimanager needs the UI built to get properly packaged into the binary
ui/dist/app: $(wildcard ui/src/**/*) $(wildcard ui/*)
	docker build $(DOCKER_BUILD_ARGS) -t cmos-ui-build -f $(DOCKERDIR)/Dockerfile.ui-build .
	mkdir -p ui/dist/app
	docker run --rm --entrypoint /bin/cat cmos-ui-build /tmp/ui-build.tar | tar -C ui/dist/app -xf -
	docker rmi cmos-ui-build

#############################################################

# This builds the DEB and RPM packages.
.PHONY: package-artifacts
ifeq ($(BINARY_DISTRO),centos)
package-artifacts: \
	$(addprefix $(ARTIFACTSDIR)/,$(addsuffix -$(BINARY_TARGET)-$(VERSION)-$(BLD_NUM).rpm,$(PACKAGES)))
else
package-artifacts: \
	$(addprefix $(ARTIFACTSDIR)/,$(addsuffix -$(BINARY_TARGET)-$(VERSION)-$(BLD_NUM).deb,$(PACKAGES)))
endif

# package-path is a helper to work out the path to a package artifact. Arguments:
# 1: the name of the package
# 2: the format (deb/rpm)
define package-path
$(ARTIFACTSDIR)/$(1)-$(BINARY_TARGET)-$(VERSION)-$(BLD_NUM).$(2)
endef

# package-target defines the recipe for a package. It's not defined as a pattern for *all* executables,
# because the % represents the package format (deb/rpm).
# The first pre-requisite must be the path to this package's nFPM config.
define package-target
	docker build $(DOCKER_BUILD_ARGS) --build-arg PACKAGE_FORMAT=$* --build-arg NFPM_CONFIG_PATH=$< -t cmos-package -f docker/Dockerfile.package-build .
	docker run --rm --entrypoint /bin/cat cmos-package /tmp/package.$* > $@
	docker rmi cmos-package
endef

$(call package-path,cbhealthagent,%): \
	agent/build/nfpm.yml \
	build/cbhealthagent-$(IMAGE_BINARY_TARGET) \
	build/fluent-bit-$(BINARY_TARGET)-$(BINARY_DISTRO) \
	build/etc/fluent-bit | dist-dir
	$(call package-target)

#############################################################

# Finally, this creates the Docker image archives.
.PHONY: image-artifacts
image-artifacts: $(addprefix $(BUILDDIR)/images/,$(IMAGES)) | dist-dir
	tar -C $(BUILDDIR)/images -cvzf $(ARTIFACTSDIR)/$(PRODUCT)-image_$(VERSION)-$(BLD_NUM).tgz $(IMAGES)

# This prepares the Docker image ingredients.
# NOTE: it doesn't use normal pre-requisites, instead invoking `make`
# recursively, because we need to override BINARY_TARGET for the binary
# builds to match IMAGE_TARGET. (Just listing out the pre-requisites won't work
# because their rules use BINARY_TARGET, which might not match.)
$(BUILDDIR)/images/%:
	mkdir -p $@
# the 'true' at the end is because all the requirements will be suffixed with &&
	$(foreach req,$(image-requirements-for.$*),$(MAKE) -e $(BUILD_ENV) -e BINARY_TARGET=$(IMAGE_BINARY_TARGET) $(req) && ) true
	$(foreach req,$(image-requirements-for.$*),cp $(req) $(BUILDDIR)/images/$*/$(notdir $(req)) && ) true
	cp $(DOCKERDIR)/Dockerfile.$* $(BUILDDIR)/images/$*/Dockerfile

image-requirements-for.couchbase-cluster-monitor := \
	$(BUILDDIR)/cbmultimanager-$(IMAGE_BINARY_TARGET)\
	$(BUILDDIR)/cbeventlog-$(IMAGE_BINARY_TARGET)\
	$(BUILDDIR)/couchbase-cluster-monitor-entrypoint.sh\
	$(BUILDDIR)/LICENSE

$(BUILDDIR)/%-entrypoint.sh: $(DOCKERDIR)/%-entrypoint.sh
	cp $< $@

$(BUILDDIR)/LICENSE: LICENSE
	cp $< $@

#############################################################

# Fluent Bit is a little special, so it has its own rules.
# NOTE: it uses linux-amd64 directly, rather than referencing $(BINARY_TARGET),
# because the build steps will likely be very different for other targets.
$(BUILDDIR)/fluent-bit-$(IMAGE_BINARY_TARGET)-$(BINARY_DISTRO): $(wildcard $(UPSTREAMDIR)/fluent-bit/**/*)
	$(eval FLUENT_BIT_BUILD_IMAGE := cmos-build-fluent-bit)
	DOCKER_BUILDKIT=1 docker build -f docker/Dockerfile.fluent-bit-linux-$(BINARY_DISTRO) -t $(FLUENT_BIT_BUILD_IMAGE) $(DOCKER_BUILD_ARGS) $(UPSTREAMDIR)
	docker run --rm --entrypoint /bin/cat $(FLUENT_BIT_BUILD_IMAGE) /work/fluent-bit/build/bin/fluent-bit > $(TARGET_BINDIR)/$(notdir $@)
	docker rmi -f $(FLUENT_BIT_BUILD_IMAGE)

# Fluent Bit's config is a little special too
$(BUILDDIR)/etc/fluent-bit: $(wildcard $(UPSTREAMDIR)/couchbase-fluent-bit/conf/**/*) $(wildcard agent/pkg/fluentbit/conf/**/*)
	mkdir -p $@
	cp -r "$(UPSTREAMDIR)/couchbase-fluent-bit/conf/." $@
	-rm $@/fluent-bit*.conf
	cp -r agent/pkg/fluentbit/conf/* $@

###############################################################################
# Generated code

docs/modules/ROOT/pages/checkers.adoc: \
	cluster-monitor/pkg/values/checker_defs.yaml \
	docs/modules/ROOT/pages/checkers.adoc.tmpl \
	docs/modules/ROOT/pages/checkers_gen.go
	go generate -tags gen ./docs/modules/ROOT/pages
