package htpasswd

import "github.com/johnaoss/htpasswd/apr1"

func ApacheHash(pass, salt string) (string, error) {
	hashedPassword, err := apr1.Hash(pass, salt)
	if err != nil {
		return "", err
	}
	return hashedPassword, nil
}
