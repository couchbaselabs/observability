all: build start logs

clean: stop
	-docker compose rm --force --stop -v
	-docker rmi couchbase-observability:latest

container-image:
	docker build -t couchbase-observability:latest -f Dockerfile .

build: clean
	docker compose build

start:
	docker compose up -d --force-recreate

stop:
	docker compose down -v --remove-orphans

logs:
	docker compose logs
