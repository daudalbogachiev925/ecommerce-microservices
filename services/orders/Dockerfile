FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/orders /src/services/orders
WORKDIR /src/services/orders
RUN go build -o /out/orders ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/orders /app/orders
COPY services/orders/migrations /app/migrations
EXPOSE 8084
ENTRYPOINT ["/app/orders"]
