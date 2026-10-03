FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/gateway /src/services/gateway
WORKDIR /src/services/gateway
RUN go build -o /out/gateway ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/gateway /app/gateway
EXPOSE 8080
ENTRYPOINT ["/app/gateway"]
