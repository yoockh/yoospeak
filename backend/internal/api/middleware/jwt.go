package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yoockh/yoospeak/internal/utils"
)

type apiError struct {
	Code    utils.Code `json:"code"`
	Message string     `json:"message"`
}

type supabaseClaims struct {
	jwt.RegisteredClaims
	Role         string         `json:"role"`         // usually "authenticated" / "anon"
	AppMetadata  map[string]any `json:"app_metadata"` // put {"role":"admin"} here
	UserMetadata map[string]any `json:"user_metadata"`
}

func JWTAuth() gin.HandlerFunc {
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	issuer := os.Getenv("SUPABASE_JWT_ISSUER")     // optional
	audience := os.Getenv("SUPABASE_JWT_AUDIENCE") // optional

	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, apiError{
				Code:    utils.CodeInternal,
				Message: "SUPABASE_JWT_SECRET is not set",
			})
			return
		}

		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:    utils.CodeUnauthorized,
				Message: "missing bearer token",
			})
			return
		}

		raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:    utils.CodeUnauthorized,
				Message: "missing bearer token",
			})
			return
		}

		claims := &supabaseClaims{}
		tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil || tok == nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:    utils.CodeUnauthorized,
				Message: "invalid token",
			})
			return
		}

		if issuer != "" && claims.Issuer != issuer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:    utils.CodeUnauthorized,
				Message: "invalid token issuer",
			})
			return
		}

		if audience != "" {
			valid := false
			for _, aud := range claims.Audience {
				if aud == audience {
					valid = true
					break
				}
			}
			if !valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
					Code:    utils.CodeUnauthorized,
					Message: "invalid token audience",
				})
				return
			}
		}

		userID := claims.Subject // Supabase user UUID ada di "sub"
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:    utils.CodeUnauthorized,
				Message: "missing subject",
			})
			return
		}

		// Default role: "user" (app-level role)
		appRole := "user"
		if claims.AppMetadata != nil {
			if v, ok := claims.AppMetadata["role"]; ok {
				if s, ok := v.(string); ok && s != "" {
					appRole = s
				}
			}
		}

		c.Set("user_id", userID)
		c.Set("role", appRole)
		c.Next()
	}
}
