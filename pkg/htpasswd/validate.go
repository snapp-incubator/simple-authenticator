package htpasswd

import "strings"

func ValidateHtpasswdFormat(pass string) bool {
	passParts := strings.Split(pass, ":")
	if len(passParts) != 2 || passParts[0] == "" || passParts[1] == "" {
		return false
	}
	return true
}
