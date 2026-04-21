package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashing(t *testing.T) {
	password := "supersecret"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("error hashing password: %v", err)
	}

	ok, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("error checking password: %v", err)
	}

	if !ok {
		t.Fatalf("expected password to match hash")
	}
}

func TestJWT(t *testing.T) {
	userID := uuid.New()
	secret := "mysecret"
	expires := time.Hour

	token, err := MakeJWT(userID, secret, expires)
	if err != nil {
		t.Fatalf("error creating JWT: %v", err)
	}

	returnedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("error validating JWT: %v", err)
	}

	if returnedID != userID {
		t.Fatalf("expected %v, got %v", userID, returnedID)
	}
}

func TestJWTExpired(t *testing.T) {
	userID := uuid.New()
	secret := "mysecret"

	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("error creating JWT: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatalf("expected error for expired token")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "correct"
	wrongSecret := "wrong"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("error creating JWT: %v", err)
	}

	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatalf("expected error for wrong secret")
	}
}