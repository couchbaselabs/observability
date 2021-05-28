all: build start logs

clean: stop
	-docker compose rm --force --stop -v
	-docker rmi couchbase-observability:latest

container-image:
	DOCKER_BUILDKIT=1 docker build --ssh default -t couchbase-observability:latest -f Dockerfile .

build: clean container-image

start:
	docker compose up -d --force-recreate

stop:
	docker compose down -v --remove-orphans

logs:
	docker compose logs
