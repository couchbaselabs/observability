bldNum = $(if $(BLD_NUM),$(BLD_NUM),999)
version = $(if $(VERSION),$(VERSION),1.0.4)
productVersion = $(version)-$(bldNum)
ARTIFACTS = build/artifacts/

# This allows the container tags to be explicitly set.
DOCKER_USER = couchbase
DOCKER_TAG = v1

# What exact revision is this?
GIT_REVISION := $(shell git rev-parse HEAD)

# Set this to, for example beta1, for a beta release.
# This will affect the "-v" version strings and docker images.
# This is analogous to revisions in DEB and RPM archives.
revision = $(if $(REVISION),$(REVISION),)

.PHONY: all build lint container container-public container-lint container-scan dist test-dist container-clean clean examples

all: clean build lint container container-lint container-scan dist test-dist examples

build:
	echo "Nothing to do - repackaging only"

image-artifacts: build
	mkdir -p $(ARTIFACTS)
	cp -rv microlith/* $(ARTIFACTS)

# This target (and only this target) is invoked by the production build job.
# This job will archive all files that end up in the dist/ directory.
dist: image-artifacts
	rm -rf dist
	mkdir -p dist
	tar -C $(ARTIFACTS) -czvf dist/couchbase-observability-stack-image_$(productVersion).tgz .
	rm -rf $(ARTIFACTS)

lint:
	tools/shellcheck.sh
	tools/licence-lint.sh

# NOTE: This target is only for local development. While we use this Dockerfile
# (for now), the actual "docker build" command is located in the Jenkins job
# "couchbase-operator-docker". We could make use of this Makefile there as
# well, but it is quite possible in future that the canonical Dockerfile will
# need to be moved to a separate repo in which case the "docker build" command
# can't be here anyway.
container: build
	docker build -f microlith/Dockerfile -t ${DOCKER_USER}/observability-stack:${DOCKER_TAG} microlith/

container-lint: build lint
	docker run --rm -i hadolint/hadolint < microlith/Dockerfile

container-scan: container
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy \
		--severity "HIGH,CRITICAL" --ignore-unfixed --exit-code 1 --no-progress ${DOCKER_USER}/observability-stack:${DOCKER_TAG}
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -e CI=true wagoodman/dive \
		${DOCKER_USER}/observability-stack:${DOCKER_TAG}

# This target pushes the containers to a public repository.
# A typical one liner to deploy to the cloud would be:
# 	make container-public -e DOCKER_USER=couchbase DOCKER_TAG=2.0.0
container-public: container
	docker push ${DOCKER_USER}/observability-stack:${DOCKER_TAG}
	docker push ${DOCKER_USER}/observability-stack-test:${DOCKER_TAG}

# Build and run the examples
example-kubernetes: container
	examples/kubernetes/run.sh

example-native: container
	examples/native/run.sh

examples: clean container example-kubernetes example-native

# Deal with automated testing
container-test:
	docker build -f testing/microlith-test/Dockerfile -t ${DOCKER_USER}/observability-stack-test:${DOCKER_TAG} testing/microlith-test/

test-kubernetes: container container-test
	DOCKER_USER=${DOCKER_USER} DOCKER_TAG=${DOCKER_TAG} testing/kubernetes/run.sh

test-native: container container-test
	DOCKER_USER=${DOCKER_USER} DOCKER_TAG=${DOCKER_TAG} testing/native/run.sh

test: clean container container-test test-kubernetes test-native

# Special target to verify the internal release pipeline will work as well
# Take the archive we would make and extract it to a local directory to then run the docker builds on
test-dist: dist
	rm -rf test-dist/
	mkdir -p test-dist/
	tar -xzvf dist/couchbase-observability-stack-image_$(productVersion).tgz -C test-dist/
	docker build -f test-dist/Dockerfile test-dist/ -t ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG}

# Remove our images then remove dangling ones to prevent any caching
container-clean:
	docker rmi -f ${DOCKER_USER}/observability-stack:${DOCKER_TAG} \
				  ${DOCKER_USER}/observability-stack-test:${DOCKER_TAG} \
				  ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG}
	docker image prune --force

clean: container-clean
	rm -rf $(ARTIFACTS) bin/ dist/ test-dist/
	examples/native/stop.sh
	rm -f examples/native/logs/*.log
	examples/kubernetes/stop.sh
