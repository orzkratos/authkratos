package utils

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yyle88/neatjson/neatjsons"
	"github.com/yyle88/rese"
)

func TestBasicEncode(t *testing.T) {
	encoded := BasicEncode("alice", "secret-token-123")
	t.Log("encoded:", encoded)
	require.Equal(t, "YWxpY2U6c2VjcmV0LXRva2VuLTEyMw==", encoded)
}

func TestBasicAuth(t *testing.T) {
	auth := BasicAuth("alice", "secret-token-123")
	t.Log("auth:", auth)
	require.Equal(t, "Basic YWxpY2U6c2VjcmV0LXRva2VuLTEyMw==", auth)
	require.Contains(t, auth, "Basic ")
}

func TestNewSet(t *testing.T) {
	set := NewSet([]string{"a", "b", "c"})
	t.Log(neatjsons.S(set))
	require.Len(t, set, 3)
	require.True(t, set["a"])
	require.True(t, set["b"])
	require.True(t, set["c"])
	require.False(t, set["d"])
}

func TestNewSet_Empty(t *testing.T) {
	set := NewSet([]string{})
	require.Len(t, set, 0)
}

func TestSample(t *testing.T) {
	slice := []string{"a", "b", "c"}
	result := Sample(slice)
	t.Log("sampled:", result)
	require.Contains(t, slice, result)
}

func TestSample_Empty(t *testing.T) {
	result := Sample([]string{})
	require.Equal(t, "", result)
}

func TestBooleanToNum(t *testing.T) {
	require.Equal(t, 1, BooleanToNum(true))
	require.Equal(t, 0, BooleanToNum(false))
}

func TestExtractPort(t *testing.T) {
	endpoint := rese.P1(url.Parse("http://localhost:8080/path"))

	port := ExtractPort(endpoint)
	t.Log("port:", port)
	require.Equal(t, "8080", port)
}

func TestExtractPort_WithIPv6(t *testing.T) {
	endpoint := rese.P1(url.Parse("http://[::1]:9090/path"))

	port := ExtractPort(endpoint)
	t.Log("port:", port)
	require.Equal(t, "9090", port)
}
