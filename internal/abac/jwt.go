package abac

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload carrying the subject's ABAC identity.
type Claims struct {
	UserID     int               `json:"uid"`
	Username   string            `json:"username"`
	Role       string            `json:"role"`
	Attributes map[string]string `json:"attrs,omitempty"`
	jwt.RegisteredClaims
}

func IssueToken(secret string, ttl time.Duration, subj Subject) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	claims := Claims{
		UserID:     subj.UserID,
		Username:   subj.Username,
		Role:       subj.Role,
		Attributes: subj.Attributes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "authsvc",
			Subject:   subj.Username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	return signed, expiresAt, err
}

func ParseToken(secret, tokenStr string) (Subject, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Subject{}, err
	}
	if !token.Valid {
		return Subject{}, fmt.Errorf("invalid token")
	}
	return Subject{
		UserID:     claims.UserID,
		Username:   claims.Username,
		Role:       claims.Role,
		Attributes: claims.Attributes,
	}, nil
}
