package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/gateway/internal/routes"
)

// Proxy — reverse proxy поверх таблицы маршрутов.
type Proxy struct {
	routes []routes.Route
	log    *slog.Logger
}

func New(rs []routes.Route, log *slog.Logger) *Proxy {
	return &Proxy{routes: rs, log: log}
}

// ServeHTTP — выбирает маршрут по самому длинному совпадающему префиксу
// и проксирует запрос. Если совпадений нет — 404.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, ok := p.match(r.URL.Path)
	if !ok {
		http.Error(w, "no route", http.StatusNotFound)
		return
	}

	target, err := url.Parse(route.Target)
	if err != nil {
		p.log.Error("bad target", "target", route.Target, "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	// StripPrefix — убираем /api/... перед отправкой в сервис.
	// Сервисы внутри не знают про префикс /api — он нужен только снаружи.
	if route.StripPrefix != "" {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, route.StripPrefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
	r.URL.Host = target.Host
	r.Host = target.Host

	// X-Forwarded-* — стандартные заголовки для проксей. Внутренние сервисы
	// могут логировать реальный IP клиента.
	r.Header.Set("X-Forwarded-Host", r.Host)

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.log.Error("proxy error", "target", route.Target, "err", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	rp.ServeHTTP(w, r)
}

// match возвращает самый длинный совпадающий префикс.
//
// ВОПРОС: почему самый длинный, а не первый?
// Ответ: /api/cart и /api/catalog начинаются одинаково. Без сортировки по длине
// можно случайно направить /api/catalog в cart. Сортировка по длине убыв. — правильный приоритет.
func (p *Proxy) match(path string) (routes.Route, bool) {
	var best routes.Route
	found := false
	for _, rt := range p.routes {
		if strings.HasPrefix(path, rt.Prefix) {
			if !found || len(rt.Prefix) > len(best.Prefix) {
				best = rt
				found = true
			}
		}
	}
	return best, found
}
