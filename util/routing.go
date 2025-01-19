package util

import (
	"net/url"
	"regexp"
	"sync"
)

var patCache = map[string]regexp.Regexp{}
var cacheLock = sync.RWMutex{}

func MatchRoutePat(pat string, path string) (map[string]string, bool) {
	re := getPattern("^" + pat + "$")

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

func getPattern(pat string) regexp.Regexp {
	// this implementation is fine since we don't ever delete items from cache.
	cacheLock.RLock()
	re, isCacheHit := patCache[pat]
	cacheLock.RUnlock()

	if !isCacheHit {
		cacheLock.Lock()
		defer cacheLock.Unlock()
		re = *regexp.MustCompile(pat)
		patCache[pat] = re
	}

	return re
}
