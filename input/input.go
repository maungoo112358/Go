package input

import (
	"errors"
	"regexp"
)

var youtubeRegex = regexp.MustCompile(`^(https?://)?(www\.)?(youtube\.com|youtu\.be)/.+$`)

func ValidateURL(url string) (string, error) {
	if youtubeRegex.MatchString(url) {
		return url, nil
	}

	return "", errors.New("Invalid Youtube URL :(")
}
