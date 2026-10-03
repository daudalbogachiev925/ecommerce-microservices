package middleware

import (
	"net/http"
	"strings"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/gateway/internal/routes"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/auth"
)

// Auth — проверяет JWT на всех маршрутах, кроме Public. Прокидывает X-User-ID внутрь.
//
// ВОПРОС: почему проверка на gateway, а не в каждом сервисе?
// Ответ: единая точка контроля. Сервисы внутри сети доверяют X-User-ID — паттерн
// "trusted proxy". Внутренний трафик не доступен извне. Если проверять в каждом сервисе —
// дублирование, разные версии JWT, риск рассинхрона.
func Auth(a *auth.Auth, table []routes.Route) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ищем маршрут — если он Public, пропускаем без проверки
			route, ok := matchRoute(table, r.URL.Path)
			if ok && route.Public {
				next.ServeHTTP(w, r)
				return
			}

			token := auth.ExtractBearer(r.Header.Get("Authorization"))
			// WebSocket не может ставить заголовок — токен прилетает в query
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			claims, err := a.Verify(token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Внутренние сервисы читают X-User-ID. Заголовок X-User-Name — для логов.
			r.Header.Set("X-User-ID", claims.UserID.String())
			r.Header.Set("X-User-Name", claims.Username)

			// снаружи мы не хотим, чтобы клиент подделал X-User-ID — значит очищаем
			// старые значения, если были (не должно быть, но защита от дурака)
			// Реальная защита — сетевой периметр: gateway единственный, кто видит внешний трафик.

			next.ServeHTTP(w, r)
		})
	}
}

func matchRoute(table []routes.Route, path string) (routes.Route, bool) {
	var best routes.Route
	found := false
	for _, rt := range table {
		if strings.HasPrefix(path, rt.Prefix) {
			if !found || len(rt.Prefix) > len(best.Prefix) {
				best = rt
				found = true
			}
		}
	}
	return best, found
}
