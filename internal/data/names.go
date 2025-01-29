package data

import "unicode"

func SanitizeName(s string) string {
	cleaned := ""
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			cleaned += string(c)
		} else if c == '(' || c == '/' || c == ',' {
			break
		}
	}
	return cleaned
}
