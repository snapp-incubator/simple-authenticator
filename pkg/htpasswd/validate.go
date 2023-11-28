package htpasswd

import "strings"

func ValidateHtpasswdFormat(pass string) bool {
	passParts := strings.Split(pass, ":")
	if len(passParts) != 2 {
		return false
	}
	return true
}
