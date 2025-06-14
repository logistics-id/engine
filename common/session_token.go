package common

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type SessionClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Permissions []string `json:"permission"`

	jwt.RegisteredClaims
}

func (c *SessionClaims) Encode(expiry time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expiry))
	c.IssuedAt = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

func TokenEncode(claim *SessionClaims) (*TokenPair, error) {
	// todo access token should be shorten and use the refresh token
	accessToken, err := claim.Encode(time.Hour * 24 * 365)
	if err != nil {
		return nil, err
	}

	// todo create refresh token

	return &TokenPair{
		AccessToken: accessToken,
	}, nil
}

func TokenDecode(tokenStr string) (*SessionClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&SessionClaims{},
		func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		},
	)
	if err != nil || !token.Valid {
		return nil, err
	}
	return token.Claims.(*SessionClaims), nil
}

func GetSession(ctx context.Context) (*SessionClaims, error) {
	sess := ctx.Value(ContextUserKey)
	if sess == nil {
		return nil, errors.New("context doesn't have any authorization")
	}

	return sess.(*SessionClaims), nil
}

func ValidTokenPermission(ctx context.Context, perm string) bool {
	claim, err := GetSession(ctx)
	if err != nil {
		return false
	}

	for _, p := range claim.Permissions {
		if p == "*" {
			return true
		}

		if p == perm {
			return true
		}

		// Handle wildcard matching
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(perm, prefix+".") {
				return true
			}
		}
	}

	return false
}
