
.PHONY: docker-build docker-publish
docker-build:
	docker build . --tag emyrk/screeps-watcher:latest

docker-publish: docker-build
	docker push emyrk/screeps-watcher:latest