package util

import (
	"net/url"
	"regexp"
)

var PatCache = map[string]regexp.Regexp{}

func RoutePat(pat string) string {
	return "(?s)^" + pat + "$"
}

func MatchRoutePat(pat string, path string) (map[string]string, bool) {
	re, isCacheHit := PatCache[pat]
	if !isCacheHit {
		re = *regexp.MustCompile(pat)
		PatCache[pat] = re
	}

	match := re.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}

	result := map[string]string{}
	for i, name := range re.SubexpNames() {
		if name != "" {
			value, err := url.QueryUnescape(match[i])
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
