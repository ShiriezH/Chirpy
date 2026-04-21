package auth

import (
	"time"
	"net/http"
	"strings"
	"fmt"
	"crypto/rand"
	"encoding/hex"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Create JWT
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

// Validate JWT
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(tokenSecret), nil
		},
	)

	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// Extract Bearer token from Authorization header
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", fmt.Errorf("invalid authorization header")
	}

	token := strings.TrimPrefix(authHeader, prefix)
	token = strings.TrimSpace(token)

	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	return token, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	const prefix = "ApiKey "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", fmt.Errorf("invalid format")
	}

	apiKey := strings.TrimPrefix(authHeader, prefix)
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return "", fmt.Errorf("empty api key")
	}

	return apiKey, nil
}