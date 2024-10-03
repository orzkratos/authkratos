package utils

import (
	"encoding/base64"
	"encoding/hex"
	"math/rand/v2"

	"github.com/google/uuid"
)

func BasicEncode(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func BasicAuth(username, password string) string {
	return "Basic " + BasicEncode(username, password)
}

func NewUUID() string {
	uux := uuid.New()
	return hex.EncodeToString(uux[:])
}

func MpKB[K comparable](a []K) (mp map[K]bool) {
	mp = make(map[K]bool, len(a))
	for _, v := range a {
		mp[v] = true
	}
	return mp
}

func Keys[K comparable, V any](a map[K]V) (keys []K) {
	keys = make([]K, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	return keys
}

func Sample[T any](a []T) (res T) {
	if len(a) > 0 {
		res = a[rand.IntN(len(a))]
	}
	return res
}