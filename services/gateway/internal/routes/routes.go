package routes

import "os"

// Route — правило маршрутизации: какой префикс пути идёт в какой backend
type Route struct {
	Prefix  string  // /api/auth, /api/catalog, ...
	Target  string  // http://users:8081, ...
	Public  bool    // если true — JWT не требуется (register/login)
	StripPrefix string // что срезать перед проксированием
}

// Config читает адреса сервисов из env и строит таблицу маршрутов.
//
// ВОПРОС: почему не Consul/etcd service discovery?
// Ответ: для одного docker-compose достаточно DNS Docker'а — имя сервиса резолвится
// в IP контейнера. Consul нужен, когда сервисы живут на разных хостах/кластерах
// и динамически мигрируют. У нас — фиксированная топология, оверкилл.
func Config() []Route {
	users := env("USERS_URL", "http://localhost:8081")
	catalog := env("CATALOG_URL", "http://localhost:8082")
	cart := env("CART_URL", "http://localhost:8083")
	orders := env("ORDERS_URL", "http://localhost:8084")
	payments := env("PAYMENTS_URL", "http://localhost:8085")

	return []Route{
		{Prefix: "/api/auth/", Target: users, Public: true, StripPrefix: "/api/auth"},
		{Prefix: "/api/me", Target: users, Public: false, StripPrefix: ""},
		{Prefix: "/api/catalog/", Target: catalog, Public: false, StripPrefix: "/api/catalog"},
		{Prefix: "/api/cart", Target: cart, Public: false, StripPrefix: "/api"},
		{Prefix: "/api/orders", Target: orders, Public: false, StripPrefix: "/api"},
		{Prefix: "/api/payments/", Target: payments, Public: false, StripPrefix: "/api"},
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
