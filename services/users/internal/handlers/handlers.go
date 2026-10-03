package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/users/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/auth"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/httpjson"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/natsx"
	"golang.org/x/crypto/bcrypt"
)

type Handlers struct {
	Store  *store.Store
	Auth   *auth.Auth
	Bus    *natsx.Client
	Logger *slog.Logger
}

func New(s *store.Store, a *auth.Auth, b *natsx.Client, l *slog.Logger) *Handlers {
	return &Handlers{Store: s, Auth: a, Bus: b, Logger: l}
}

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, 400, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if len(req.Username) < 3 || len(req.Password) < 6 || !strings.Contains(req.Email, "@") {
		httpjson.Error(w, 400, "username >= 3, password >= 6, valid email required")
		return
	}

	if u, _ := h.Store.FindByUsername(r.Context(), req.Username); u != nil {
		httpjson.Error(w, 409, "username taken")
		return
	}
	if u, _ := h.Store.FindByEmail(r.Context(), req.Email); u != nil {
		httpjson.Error(w, 409, "email taken")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpjson.Error(w, 500, "hash failed")
		return
	}

	u, err := h.Store.Create(r.Context(), req.Username, req.Email, string(hash))
	if err != nil {
		h.Logger.Error("create user", "err", err)
		httpjson.Error(w, 500, "create failed")
		return
	}

	// публикуем user.registered — другие сервисы (notifications) отреагируют
	h.publishUserRegistered(r.Context(), u)

	token, err := h.Auth.Issue(u.ID, u.Username)
	if err != nil {
		httpjson.Error(w, 500, "token failed")
		return
	}

	httpjson.Write(w, 201, map[string]any{
		"token": token,
		"user":  u,
	})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, 400, "invalid json")
		return
	}

	u, err := h.Store.FindByUsername(r.Context(), req.Username)
	if err != nil {
		httpjson.Error(w, 500, "lookup failed")
		return
	}
	if u == nil {
		httpjson.Error(w, 401, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		httpjson.Error(w, 401, "invalid credentials")
		return
	}

	token, err := h.Auth.Issue(u.ID, u.Username)
	if err != nil {
		httpjson.Error(w, 500, "token failed")
		return
	}

	httpjson.Write(w, 200, map[string]any{
		"token": token,
		"user":  u,
	})
}

// Me — заголовок Authorization уже проверен gateway, сюда приходит X-User-ID
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	uid := r.Header.Get("X-User-ID")
	if uid == "" {
		httpjson.Error(w, 401, "unauthorized")
		return
	}

	u, err := h.Store.FindByID(r.Context(), mustUUID(uid))
	if err != nil || u == nil {
		httpjson.Error(w, 404, "not found")
		return
	}
	httpjson.Write(w, 200, u)
}

// GetByID — внутренний эндпоинт для других сервисов
func (h *Handlers) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		httpjson.Error(w, 400, "id required")
		return
	}
	u, err := h.Store.FindByID(r.Context(), mustUUID(id))
	if err != nil || u == nil {
		httpjson.Error(w, 404, "not found")
		return
	}
	httpjson.Write(w, 200, u)
}

func (h *Handlers) publishUserRegistered(ctx context.Context, u *store.User) {
	payload := events.UserRegistered{
		UserID:   u.ID.String(),
		Username: u.Username,
		Email:    u.Email,
	}
	env, err := events.NewEnvelope(events.SubjectUserRegistered, payload)
	if err != nil {
		h.Logger.Error("envelope", "err", err)
		return
	}
	if err := h.Bus.Publish(ctx, events.SubjectUserRegistered, env); err != nil {
		// не валим регистрацию из-за брокера — событие критично, но не блокирующее
		h.Logger.Error("publish user.registered", "err", err)
	}
}
