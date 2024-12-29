package util

import "regexp"

func RoutePat(pat string) string {
	return "(?s)^" + pat + "$"
}

func MatchRoutePat(pat regexp.Regexp, path string) (map[string]string, bool) {
	match := pat.FindStringSubmatch(path)
	if match == nil {
		return nil, false
	}

	result := map[string]string{}
	for i, name := range pat.SubexpNames() {
		if name != "" {
			result[name] = match[i]
		}
	}

	return result, true
}
