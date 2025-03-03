.PHONY: build-image
build-image:
	docker build -t scalecraft/aws-bedrock-gateway:$(VERSION)-linux-arm64 -t scalecraft/aws-bedrock-gateway:latest \
		--platform linux/arm64 \
		-f .docker/Dockerfile .

	docker build -t scalecraft/aws-bedrock-gateway:$(VERSION)-linux-amd64 -t scalecraft/aws-bedrock-gateway:latest \
		--platform linux/amd64 \
		-f .docker/Dockerfile .

.PHONY: push-image
push-image:
	docker push scalecraft/aws-bedrock-gateway:$(VERSION)-linux-arm64
	docker push scalecraft/aws-bedrock-gateway:$(VERSION)-linux-amd64

	docker manifest create scalecraft/aws-bedrock-gateway:$(VERSION) \
		--amend scalecraft/aws-bedrock-gateway:$(VERSION)-linux-arm64 --amend scalecraft/aws-bedrock-gateway:$(VERSION)-linux-amd64

	docker manifest push scalecraft/aws-bedrock-gateway:$(VERSION)

	docker manifest create scalecraft/aws-bedrock-gateway:latest \
		--amend scalecraft/aws-bedrock-gateway:$(VERSION)-linux-arm64 --amend scalecraft/aws-bedrock-gateway:$(VERSION)-linux-amd64

	docker manifest push scalecraft/aws-bedrock-gateway:latest

.PHONY: run
run:
	go run main.go
