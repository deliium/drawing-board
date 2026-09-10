package auth

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes a password with bcrypt (cost 12). New writes must use this only.
func HashPassword(pw string) (string, error) {
	authLog("DEBUG", "[auth.hash] HashPassword start")
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		authLog("ERROR", "[auth.hash] bcrypt generate failed: "+err.Error())
		return "", err
	}
	authLog("DEBUG", "[auth.hash] HashPassword ok hash_kind=bcrypt")
	return string(hash), nil
}

// VerifyPassword checks a stored bcrypt hash against the candidate password.
// Non-bcrypt stored values (including pre-migration SHA-256 hex) never match.
func VerifyPassword(stored, pw string) (ok bool, err error) {
	authLog("DEBUG", "[auth.hash] VerifyPassword start")
	if stored == "" || !strings.HasPrefix(stored, "$2") {
		authLog("DEBUG", "[auth.hash] hash_kind=unknown")
		return false, nil
	}
	authLog("DEBUG", "[auth.hash] hash_kind=bcrypt")
	cmpErr := bcrypt.CompareHashAndPassword([]byte(stored), []byte(pw))
	if cmpErr == nil {
		authLog("DEBUG", "[auth.hash] VerifyPassword match hash_kind=bcrypt")
		return true, nil
	}
	if cmpErr == bcrypt.ErrMismatchedHashAndPassword {
		authLog("DEBUG", "[auth.hash] VerifyPassword mismatch hash_kind=bcrypt")
		return false, nil
	}
	authLog("ERROR", "[auth.hash] bcrypt compare unexpected: "+cmpErr.Error())
	return false, cmpErr
}
