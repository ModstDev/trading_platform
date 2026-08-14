package auth

import "testing"

func TestHashPassword(t *testing.T) {
	password := "correct testing password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned an empty hash")
	}

	if hash == password {
		t.Fatal("password was stored as plaintext")
	}
}

func TestHashPasswordGeneratesDifferentHashes(t *testing.T) {
	password := "correct testing password"

	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if first == second {
		t.Fatal("same password produced identical hashes")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "correct testing password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := CheckPassword(password, hash)
	if err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}

	if !ok {
		t.Fatal("CheckPassword() returned false for the correct password")
	}
}

func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct testing password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := CheckPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}

	if ok {
		t.Fatal("CheckPassword() returned true for the wrong password")
	}
}
