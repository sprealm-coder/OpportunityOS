package platform

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("secure random source unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
