package util

import (
	"net/url"
	"regexp"
)

func MatchRoutePat(re regexp.Regexp, path string) (map[string]string, bool) {
	match := re.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}

	result := map[string]string{}
	for i, name := range re.SubexpNames() {
		if name != "" {
			value, err := url.PathUnescape(match[i])
			if err != nil {
				// todo: should this be responding with a 400?
				result[name] = match[i]
			} else {
				result[name] = value
			}
		}
	}

	return result, true
}
