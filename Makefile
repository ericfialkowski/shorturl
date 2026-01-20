DOCKER_IMAGE_NAME=short-url
DOCKER_REGISTRY=localhost

build: test
	go build -v -o bin/

run: build
	./bin/shorturl

test:
	go test ./...

coverage:
	go test -coverprofile=cov.out ./... && go tool cover -func=cov.out

lint:
	golangci-lint run

clean:
	go clean
	go clean -testcache
	-rm bin/**

image:
	docker build -t ${DOCKER_IMAGE_NAME} .

publish: image
	docker image tag ${DOCKER_IMAGE_NAME}:latest ${DOCKER_REGISTRY}:5000/${DOCKER_IMAGE_NAME}
