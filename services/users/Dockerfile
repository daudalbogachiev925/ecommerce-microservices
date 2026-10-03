FROM golang:1.22-alpine AS builder
WORKDIR /src

# сначала зависимости shared
COPY shared /src/shared

# затем сервис
COPY services/users /src/services/users
WORKDIR /src/services/users
RUN go build -o /out/users ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/users /app/users
COPY services/users/migrations /app/migrations
EXPOSE 8081
ENTRYPOINT ["/app/users"]
