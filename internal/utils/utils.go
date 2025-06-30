package utils

import (
	"encoding/base64"
	"math/rand/v2"
)

func BasicEncode(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func BasicAuth(username, password string) string {
	return "Basic " + BasicEncode(username, password)
}

func NewKeysMap[K comparable](a []K) (mp map[K]bool) {
	mp = make(map[K]bool, len(a))
	for _, v := range a {
		mp[v] = true
	}
	return mp
}

func Sample[T any](a []T) (res T) {
	if len(a) > 0 {
		res = a[rand.IntN(len(a))]
	}
	return res
}

func BooleanToNum(b bool) int {
	if b {
		return 1
	}
	return 0
}
