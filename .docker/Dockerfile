FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bedrock-gateway .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bedrock-gateway .
EXPOSE 8000

CMD ["./bedrock-gateway"]
