.PHONY: build
build:
	docker build -t aws-bedrock-gateway -f .docker/Dockerfile .

.PHONY: run
run:
	go run main.go
