module github.com/daudalobogachiev925/ecommerce-microservices/services/cart

go 1.22

require (
	github.com/daudalobogachiev925/ecommerce-microservices/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.5
)

replace github.com/daudalobogachiev925/ecommerce-microservices/shared => ../../shared
