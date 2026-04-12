FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o sub2balance cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/sub2balance .
COPY config.yaml.example config.yaml

RUN mkdir -p data

EXPOSE 8080

CMD ["./sub2balance"]
