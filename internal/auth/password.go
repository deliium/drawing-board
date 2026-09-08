package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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

// VerifyPassword checks a stored hash against the candidate password.
// ok is true on match. needsUpgrade is true when the stored value is a legacy
// SHA-256 hex hash that should be rewritten to bcrypt after a successful login.
func VerifyPassword(stored, pw string) (ok bool, needsUpgrade bool, err error) {
	authLog("DEBUG", "[auth.hash] VerifyPassword start")
	if stored == "" {
		authLog("DEBUG", "[auth.hash] hash_kind=unknown")
		return false, false, nil
	}
	if strings.HasPrefix(stored, "$2") {
		authLog("DEBUG", "[auth.hash] hash_kind=bcrypt")
		cmpErr := bcrypt.CompareHashAndPassword([]byte(stored), []byte(pw))
		if cmpErr == nil {
			authLog("DEBUG", "[auth.hash] VerifyPassword match hash_kind=bcrypt")
			return true, false, nil
		}
		if cmpErr == bcrypt.ErrMismatchedHashAndPassword {
			authLog("DEBUG", "[auth.hash] VerifyPassword mismatch hash_kind=bcrypt")
			return false, false, nil
		}
		authLog("ERROR", "[auth.hash] bcrypt compare unexpected: "+cmpErr.Error())
		return false, false, cmpErr
	}

	// Legacy unsalted SHA-256 hex (64 hex chars expected).
	authLog("DEBUG", "[auth.hash] hash_kind=legacy")
	sum := sha256.Sum256([]byte(pw))
	want := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(stored), []byte(want)) == 1 {
		authLog("DEBUG", "[auth.hash] VerifyPassword match hash_kind=legacy needsUpgrade=true")
		return true, true, nil
	}
	authLog("DEBUG", "[auth.hash] VerifyPassword mismatch hash_kind=legacy")
	return false, false, nil
}

// legacySHA256Hash is used only by tests to seed pre-migration users.
func legacySHA256Hash(pw string) string {
	sum := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(sum[:])
}
