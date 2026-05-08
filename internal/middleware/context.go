package middleware

import (
	"context"

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type contextKey string

const claimsKey contextKey = "claims"

func ContextWithClaims(ctx context.Context, claims *service.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*service.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*service.Claims)
	return claims, ok
}
