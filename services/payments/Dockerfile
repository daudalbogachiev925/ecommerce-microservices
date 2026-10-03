FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/payments /src/services/payments
WORKDIR /src/services/payments
RUN go build -o /out/payments ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/payments /app/payments
COPY services/payments/migrations /app/migrations
EXPOSE 8085
ENTRYPOINT ["/app/payments"]
