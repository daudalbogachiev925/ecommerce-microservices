package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   uuid.UUID
	Username string
}

type Auth struct {
	secret []byte
}

func New(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

func (a *Auth) Issue(userID uuid.UUID, username string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"usr": username,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	return t.SignedString(a.secret)
}

func (a *Auth) Verify(token string) (*Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("bad alg")
		}
		return a.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	c, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("bad claims")
	}
	sub, _ := c["sub"].(string)
	usr, _ := c["usr"].(string)
	id, err := uuid.Parse(sub)
	if err != nil {
		return nil, err
	}
	return &Claims{UserID: id, Username: usr}, nil
}

func ExtractBearer(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
