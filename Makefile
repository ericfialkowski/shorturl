DOCKER_IMAGE_NAME=short-url
DOCKER_REGISTRY=localhost

build:
	go build -v -o bin/

run: build
	./bin/shorturl

clean:
	go clean
	del bin\**

image:
	docker build -t ${DOCKER_IMAGE_NAME} .

publish: image
	docker image tag ${DOCKER_IMAGE_NAME}:latest ${DOCKER_REGISTRY}:5000/${DOCKER_IMAGE_NAME}
