module github.com/daudalobogachiev925/ecommerce-microservices/services/users

go 1.22

require (
	github.com/daudalobogachiev925/ecommerce-microservices/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/nats-io/nats.go v1.34.1
	golang.org/x/crypto v0.22.0
)

replace github.com/daudalobogachiev925/ecommerce-microservices/shared => ../../shared
