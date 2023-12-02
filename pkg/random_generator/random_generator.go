package random_generator

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	for i := range randomBytes {
		randomBytes[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(randomBytes), nil
}

func GenerateRandomName(baseName string, salt string) string {
	tuple := fmt.Sprintf("%s-%s", baseName, salt)
	sum := sha256.Sum256([]byte(tuple))
	subByte := sum[:8]
	return fmt.Sprintf("%s-%s", baseName, hex.EncodeToString(subByte))
}
