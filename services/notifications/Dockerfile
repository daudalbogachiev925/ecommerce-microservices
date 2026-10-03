FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/notifications /src/services/notifications
WORKDIR /src/services/notifications
RUN go build -o /out/notifications ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/notifications /app/notifications
COPY services/notifications/migrations /app/migrations
EXPOSE 8086
ENTRYPOINT ["/app/notifications"]
