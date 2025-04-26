FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /load-balancer ./cmd/main.go

FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /load-balancer .
COPY config.yaml .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./load-balancer"]