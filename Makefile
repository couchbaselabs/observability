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

.PHONY: all build lint container container-oss container-public container-lint container-scan dist test-dist container-clean clean examples example-containers test test-kubernetes test-native test-containers docs docs-generate-markdown docs-lint

# TODO: add 'test examples'
all: clean build lint container container-oss container-lint container-scan dist test-dist

# We need to copy docs in for packaging: https://github.com/moby/moby/issues/1676
# The other option is to tar things up and pass as the build context: tar -czh . | docker build -
build: docs
	cp -R docs/ microlith/docs/
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

lint: container-lint docs-lint
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

example-containers: container
	examples/containers/run.sh

examples: clean container example-kubernetes example-containers

# Deal with automated testing
test-kubernetes:
	testing/run-k8s.sh

test-containers:
	testing/run-containers.sh

test-native:
	testing/run-native.sh

test: clean container-oss test-native test-containers test-kubernetes

# Special target to verify the internal release pipeline will work as well
# Take the archive we would make and extract it to a local directory to then run the docker builds on
test-dist: dist
	rm -rf test-dist/
	mkdir -p test-dist/
	tar -xzvf dist/couchbase-observability-stack-image_$(productVersion).tgz -C test-dist/
	docker build -f test-dist/Dockerfile test-dist/ -t ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG}

test-dist-oss: dist
	rm -rf test-dist/
	mkdir -p test-dist/
	tar -xzvf dist/couchbase-observability-stack-image_$(productVersion).tgz -C test-dist/
	sed '/^# Couchbase proprietary start/,/^# Couchbase proprietary end/d' "test-dist/Dockerfile" > "test-dist/Dockerfile.oss"
	docker build -f test-dist/Dockerfile.oss test-dist/ -t ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG}

# Remove our images then remove dangling ones to prevent any caching
container-clean:
	docker rmi -f ${DOCKER_USER}/observability-stack:${DOCKER_TAG} \
				  ${DOCKER_USER}/observability-stack-test-dist:${DOCKER_TAG} \
				  ${DOCKER_USER}/observability-stack-docs-generator:${DOCKER_TAG}
	docker image prune --force

clean: container-clean
	rm -rf $(ARTIFACTS) bin/ dist/ test-dist/ build/ .cache/ microlith/html/cmos/ microlith/docs/
	-examples/containers/stop.sh
	rm -f examples/containers/logs/*.log
	-examples/kubernetes/stop.sh

docs-lint:
	docker run --rm -i hadolint/hadolint < Dockerfile.docs
	tools/asciidoc-lint.sh

docs: docs-generate-markdown

# Automatically convert Markdown docs to Asciidoc ones.
# This command needs bind mount support so will not run in Couchbase build infrastructure (Docker Swarm):
# docker run -u $(shell id -u) -v $$PWD:/documents asciidoctor/docker-asciidoctor kramdoc README.md -o docs/modules/ROOT/pages/index.adoc
# We therefore create a custom container for it all. Unfortunately this has a knock on in that forwarding can mess up line endings.
docs-generate-markdown:
	DOCKER_BUILDKIT=1 docker build -t ${DOCKER_USER}/observability-stack-docs-generator:${DOCKER_TAG} -f Dockerfile.docs .
	docker run --rm -t ${DOCKER_USER}/observability-stack-docs-generator:${DOCKER_TAG} > docs/modules/ROOT/pages/index.adoc
	tr -d "\r" < docs/modules/ROOT/pages/index.adoc > /tmp/observability-stack-docs-output.adoc
	mv /tmp/observability-stack-docs-output.adoc docs/modules/ROOT/pages/index.adoc
	rm -f observability-stack-docs-output.adoc
