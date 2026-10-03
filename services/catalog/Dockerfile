  FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY shared /src/shared
COPY services/catalog /src/services/catalog
WORKDIR /src/services/catalog
RUN go build -o /out/catalog ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/catalog /app/catalog
COPY services/catalog/migrations /app/migrations
EXPOSE 8082
ENTRYPOINT ["/app/catalog"]
