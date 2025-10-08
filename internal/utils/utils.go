// Package utils: Shared functions across authkratos packages
// Provides common helpers including Basic Auth encoding, set operations, and port extraction
//
// utils: authkratos 包共享的工具函数
// 提供常用辅助函数，包括 Basic Auth 编码、集合操作和端口提取
package utils

import (
	"encoding/base64"
	"math/rand/v2"
	"net"
	"net/url"

	"github.com/yyle88/must"
)

// BasicEncode encodes username and password to Base64 format
// Returns Base64 string in format "username:password"
//
// BasicEncode 将用户名和密码编码成 Base64 格式
// 返回格式："username:password" 的 Base64 字符串
func BasicEncode(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// BasicAuth creates HTTP Basic Authentication value
// Returns "Basic {base64-encoded-credentials}" string
//
// BasicAuth 创建 HTTP Basic 认证头的值
// 返回 "Basic {base64编码的凭证}" 字符串
func BasicAuth(username, password string) string {
	return "Basic " + BasicEncode(username, password)
}

// NewKeysMap creates a boolean map from slice entries fast lookup
// Returns map where each slice element becomes an entry with true value
//
// NewKeysMap 从切片键创建布尔映射用于快速查找
// 返回所有切片元素作为键且值都是 true 的 map
func NewKeysMap[K comparable](a []K) (mp map[K]bool) {
	mp = make(map[K]bool, len(a))
	for _, v := range a {
		mp[v] = true
	}
	return mp
}

// NewSet creates a new set from slice fast lookup
// Returns map where each slice element becomes an entry with true value
//
// NewSet 从切片创建集合用于快速查找
// 返回所有切片元素作为键且值都是 true 的 map
func NewSet[T comparable](slice []T) map[T]bool {
	set := make(map[T]bool, len(slice))
	for _, v := range slice {
		set[v] = true
	}
	return set
}

// Sample selects one element at random from slice
// Returns zero value when slice is blank
//
// Sample 从切片中随机选择一个元素
// 如果切片为空则返回零值
func Sample[T any](a []T) (res T) {
	if len(a) > 0 {
		res = a[rand.IntN(len(a))]
	}
	return res
}

// BooleanToNum converts boolean to int
// Returns 1 when true, 0 when false
//
// BooleanToNum 将布尔值转换成整数
// true 时返回 1，false 时返回 0
func BooleanToNum(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ExtractPort extracts port from URL endpoint
// Returns port string from endpoint.Host field
//
// ExtractPort 从 URL 端点提取端口号
// 从 endpoint.Host 字段返回端口字符串
func ExtractPort(endpoint *url.URL) string {
	must.Full(endpoint)
	_, port, _ := net.SplitHostPort(must.Nice(endpoint.Host))
	return must.Nice(port)
}
