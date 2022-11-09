BINARY_NAME='gobank'

docker:
	@docker-compose up --detach

build:
	@go build -o bin/${BINARY_NAME}

run: docker build 
	@./bin/${BINARY_NAME}

test: 
	@go test -v ./...
