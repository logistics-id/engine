package common

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultExpiry = time.Hour * 24 * 365

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type SessionClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Permissions []string `json:"permission"`
	Type        string   `json:"type"`
	jwt.RegisteredClaims
}

// Global factory agar setiap service bisa menentukan tipe claim-nya
var claimFactory func() jwt.Claims

// Optional setter, hanya dipakai oleh service yang butuh klaim custom
func SetClaimFactory(factory func() jwt.Claims) {
	claimFactory = factory
}

// setExpiry — otomatis mengisi expiry ke semua claim termasuk yang embed SessionClaims.
func setExpiry(c jwt.Claims) {
	now := time.Now()
	exp := now.Add(defaultExpiry)

	switch v := c.(type) {
	case *SessionClaims:
		v.ExpiresAt = jwt.NewNumericDate(exp)
		v.IssuedAt = jwt.NewNumericDate(now)
	default:
		type hasBase interface{ GetBase() *SessionClaims }
		if h, ok := c.(hasBase); ok {
			base := h.GetBase()
			base.ExpiresAt = jwt.NewNumericDate(exp)
			base.IssuedAt = jwt.NewNumericDate(now)
		}
	}
}

// TokenEncode — otomatis set expiry (1 tahun) untuk semua struct turunan SessionClaims.
func TokenEncode(c jwt.Claims) (*TokenPair, error) {
	setExpiry(c)

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	accessToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken}, nil
}

func TokenDecode(tokenStr string) (jwt.Claims, error) {
	secret := os.Getenv("JWT_SECRET")

	// gunakan claimFactory kalau sudah diset di service
	var claims jwt.Claims
	if claimFactory != nil {
		claims = claimFactory()
	} else {
		claims = &SessionClaims{}
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	return token.Claims, nil
}

func getSession(ctx context.Context) (*SessionClaims, error) {
	sess := ctx.Value(ContextUserKey)
	if sess == nil {
		return nil, errors.New("context doesn't have any authorization")
	}

	// ✅ langsung cocok dengan *SessionClaims
	if v, ok := sess.(*SessionClaims); ok {
		return v, nil
	}

	// ✅ jika pakai custom claim (warehouse/order) yg embed SessionClaims
	rv := reflect.ValueOf(sess)
	if rv.Kind() == reflect.Ptr {
		elem := rv.Elem()
		if elem.Kind() == reflect.Struct {
			for i := 0; i < elem.NumField(); i++ {
				field := elem.Field(i)
				if field.CanInterface() && field.Type() == reflect.TypeOf(SessionClaims{}) {
					base := field.Interface().(SessionClaims)
					return &base, nil
				}
				// kalau field berupa pointer ke SessionClaims
				if field.CanInterface() && field.Type() == reflect.TypeOf(&SessionClaims{}) {
					if sc, ok := field.Interface().(*SessionClaims); ok {
						return sc, nil
					}
				}
			}
		}
	}

	return nil, errors.New("invalid session type")
}

func ValidTokenPermission(ctx context.Context, perm string) bool {
	claim, err := getSession(ctx)
	if err != nil {
		return false
	}

	if len(claim.Permissions) == 0 {
		return true
	}

	for _, p := range claim.Permissions {
		if p == "*" {
			return true
		}

		if p == perm {
			return true
		}

		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(perm, prefix+".") {
				return true
			}
		}
	}

	return false
}
