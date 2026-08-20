FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/outbox-worker ./cmd/outbox-worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/matching-worker ./cmd/matching-worker


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/outbox-worker /bin/outbox-worker
COPY --from=builder /bin/matching-worker /bin/matching-worker