package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/christopherdang/vibecloud/api/internal/response"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const EmailKey contextKey = "email"
const TierKey contextKey = "tier"

func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

func GetEmail(ctx context.Context) string {
	v, _ := ctx.Value(EmailKey).(string)
	return v
}

func GetTier(ctx context.Context) string {
	v, _ := ctx.Value(TierKey).(string)
	return v
}

type Middleware struct {
	signingSecret []byte
}

func NewMiddleware(secret string) *Middleware {
	return &Middleware{signingSecret: []byte(secret)}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(w, "Invalid authorization header format")
			return
		}

		userID, email, tier, err := m.verifyJWT(parts[1])
		if err != nil {
			response.Unauthorized(w, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, EmailKey, email)
		ctx = context.WithValue(ctx, TierKey, tier)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) verifyJWT(tokenString string) (userID, email, tier string, err error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingSecret, nil
	}, jwt.WithExpirationRequired())

	if err != nil || !token.Valid {
		return "", "", "", fmt.Errorf("invalid token: %w", err)
	}

	sub, _ := claims["sub"].(string)
	emailStr, _ := claims["email"].(string)
	tierStr, _ := claims["tier"].(string)
	return sub, emailStr, tierStr, nil
}
