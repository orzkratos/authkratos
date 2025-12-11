package authkratosroutes

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/orzkratos/authkratos"
	"github.com/stretchr/testify/require"
)

// TestMain sets up test environment with debug mode enabled
// TestMain 设置测试环境并启用调试模式
func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)
	m.Run()
}

// TestNewMatchFunc tests NewMatchFunc with INCLUDE mode
// TestNewMatchFunc 测试 NewMatchFunc 在 INCLUDE 模式下的行为
func TestNewMatchFunc(t *testing.T) {
	config := NewConfig("do-something", NewInclude(
		"a/b/c",
		"x/y/z",
	))
	matchFunc := NewMatchFunc(config, log.DefaultLogger)
	require.True(t, matchFunc(context.Background(), "a/b/c"))
	require.True(t, matchFunc(context.Background(), "x/y/z"))
	require.False(t, matchFunc(context.Background(), "u/v/w"))
	require.False(t, matchFunc(context.Background(), "r/s/t"))
}

// TestNewMatchFunc_Exclude tests NewMatchFunc with EXCLUDE mode
// TestNewMatchFunc_Exclude 测试 NewMatchFunc 在 EXCLUDE 模式下的行为
func TestNewMatchFunc_Exclude(t *testing.T) {
	config := NewConfig("do-something", NewExclude(
		"a/b/c",
		"x/y/z",
	))
	matchFunc := NewMatchFunc(config, log.DefaultLogger)
	require.False(t, matchFunc(context.Background(), "a/b/c"))
	require.False(t, matchFunc(context.Background(), "x/y/z"))
	require.True(t, matchFunc(context.Background(), "u/v/w"))
	require.True(t, matchFunc(context.Background(), "r/s/t"))
}
