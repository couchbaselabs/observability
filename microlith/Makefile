all: build start logs

clean: stop
	-docker rmi couchbase-observability:latest

container-image:
	docker build -t couchbase-observability:latest -f Dockerfile .

build: clean
	docker compose build

start:
	docker compose up -d

stop:
	docker compose down

logs:
	docker compose logs
