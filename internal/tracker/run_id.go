package tracker

import (
	"crypto/rand"
	"encoding/hex"
)

func NewRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
