FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o aws-bedrock-gateway .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=builder /app/aws-bedrock-gateway .
EXPOSE 8000

CMD ["./aws-bedrock-gateway"]
