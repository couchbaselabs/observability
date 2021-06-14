all: build

clean:
	-docker rmi couchbase-observability:latest

build: clean
	DOCKER_BUILDKIT=1 docker build --ssh default -t couchbase-observability:latest -f ../../microlith/Dockerfile ../../microlith/
