bldNum = $(if $(BLD_NUM),$(BLD_NUM),999)
version = $(if $(VERSION),$(VERSION),1.0.0)
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

.PHONY: all build lint container container-oss container-public container-lint container-scan dist test-dist container-clean clean examples test test-kubernetes test-native container-test

# TODO: add 'test examples'
all: clean build lint container container-oss container-lint container-scan dist test-dist

build:
	echo "Version: $(version)" >> microlith/git-commit.txt
	echo "Build: $(productVersion)" > microlith/git-commit.txt
	echo "Revision: $(revision)" >> microlith/git-commit.txt
	echo "Git commit: $(GIT_REVISION)" >> microlith/git-commit.txt

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

lint: container-lint
	tools/shellcheck.sh
	tools/licence-lint.sh

# NOTE: This target is only for local development.
container: build
	DOCKER_BUILDKIT=1 docker build --ssh default -f microlith/Dockerfile -t ${DOCKER_USER}/observability-stack:${DOCKER_TAG} microlith/

container-oss: build
	tools/build-oss-container.sh

container-lint:
	docker run --rm -i hadolint/hadolint < microlith/Dockerfile

container-scan: container
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy \
		--severity "HIGH,CRITICAL" --ignore-unfixed --exit-code 1 --no-progress ${DOCKER_USER}/observability-stack:${DOCKER_TAG}
	-docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -e CI=true wagoodman/dive \
		${DOCKER_USER}/observability-stack:${DOCKER_TAG}

# This target pushes the containers to a public repository.
# A typical one liner to deploy to the cloud would be:
# 	make container-public -e DOCKER_USER=couchbase DOCKER_TAG=2.0.0
container-public: container
	docker push ${DOCKER_USER}/observability-stack:${DOCKER_TAG}

# Build and run the examples
example-kubernetes: container
	examples/kubernetes/run.sh

example-native: container
	examples/native/run.sh

examples: clean container example-kubernetes example-native

# Deal with automated testing
test-kubernetes:
	DOCKER_USER=${DOCKER_USER} DOCKER_TAG=${DOCKER_TAG} testing/kubernetes/run.sh

test-native:
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
				  ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG}
	docker image prune --force

clean: container-clean
	rm -rf $(ARTIFACTS) bin/ dist/ test-dist/
	examples/native/stop.sh
	rm -f examples/native/logs/*.log
	examples/kubernetes/stop.sh
