FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/cart /src/services/cart
WORKDIR /src/services/cart
RUN go build -o /out/cart ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/cart /app/cart
COPY services/cart/migrations /app/migrations
EXPOSE 8083
ENTRYPOINT ["/app/cart"]
