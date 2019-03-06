package utils

import "regexp"

func StringPtr(str string) *string {
	return &str
}

var ValidTaskNameRegexp = regexp.MustCompile(`(?i)^[A-Za-z0-9\-]+$`)
