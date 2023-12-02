package random_generator

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)
	rand.Read(randomBytes)
	for i := range randomBytes {
		randomBytes[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(randomBytes)
}

func GenerateRandomName(baseName string, salt string) string {
	tuple := fmt.Sprintf("%s-%s", baseName, salt)
	sum := sha256.Sum256([]byte(tuple))
	subByte := sum[:8]
	return fmt.Sprintf("%s-%s", baseName, hex.EncodeToString(subByte))
}
