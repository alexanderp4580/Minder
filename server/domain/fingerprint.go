package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

// Fingerprint names a secret value in logs without revealing it: the first four bytes of its SHA-256 hash.
func Fingerprint(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:4])
}
