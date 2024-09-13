package main

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtConfig struct {
	Issuer    string
	ExpiresAt int
	Subject   string
}

func (cfg *JwtConfig) generateToken() (string, error) {
	// expiredTime := time.Duration(cfg.ExpiresAt) * time.Second
	// if cfg.ExpiresAt > 86400 {
	// 	expiredTime = 86400 * time.Second
	// }
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   cfg.Subject,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	)

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return claims, err
	}

	if !token.Valid {
		return claims, err
	}

	return claims, nil
}
