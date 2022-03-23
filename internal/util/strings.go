package util

import "strings"

func IsBlank(s *string) bool {
	return s == nil || strings.TrimSpace(*s) == ""
}
