# AWS Bedrock Gateway

An OpenAI-compatible API gateway for AWS Bedrock, allowing you to use AWS Bedrock models with tools and applications designed for OpenAI's API. Written in Go. Works with OpenWebUI.

## Features

- OpenAI-compatible REST API endpoints
- Support for Claude 3 and other AWS Bedrock models
- Streaming support for chat completions
- Model listing endpoint
- Automatic handling of model-specific requirements

## Configuration

Environment variables:

- `AWS_REGION`: AWS region (default: "us-east-1")
- `PORT`: Server port (default: "8000")
- `DEFAULT_MODEL`: Default model ID (default: "anthropic.claude-3-sonnet-20240229-v1:0")
- `API_ROUTE_PREFIX`: API route prefix (default: "/api/v1")
- `DEBUG`: Enable debug mode (default: false)
- `ENABLE_CROSS_REGION_INFERENCE`: Enable cross-region inference (default: false)

## Running

The app uses the AWS SDK for Go, so you need to set up AWS credentials. We use the default [AWS credentials chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

### Docker

```bash
export DOCKER_DEFAULT_PLATFORM=linux/arm64 # This is for Apple Silicon, change to linux/amd64 for Intel
docker run -p 8080:8000 -e AWS_REGION=us-east-1 -e DEFAULT_MODEL=anthropic.claude-3-sonnet-20240229-v1:0 scalecraft/aws-bedrock-gateway:latest
```

### Metal

1. Configure environment variables (optional). The application will load a `.env` file in the root directory if it exists.

```bash
go run .
```

## API Endpoints

### Chat Completions

```bash
POST /api/v1/chat/completions
```

Compatible with OpenAI's chat completions API. Supports both streaming and non-streaming responses.

### List Models

```bash
GET /api/v1/models
```

Lists available Bedrock models in OpenAI-compatible format.

## Example Usage

```bash
# Chat completion
curl -X POST http://localhost:8000/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic.claude-3-sonnet-20240229-v1:0",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# List models
curl http://localhost:8000/api/v1/models
```

## License

[MIT License](LICENSE)
