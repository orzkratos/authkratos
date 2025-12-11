package authkratosroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRouteScope_Match tests RouteScope Match with INCLUDE and EXCLUDE modes
// TestRouteScope_Match 测试 RouteScope Match 在 INCLUDE 和 EXCLUDE 模式下的行为
func TestRouteScope_Match(t *testing.T) {
	t.Run("match-include", func(t *testing.T) {
		scope := NewInclude("a/b/c", "x/y/z")
		require.True(t, scope.Match("a/b/c"))
		require.True(t, scope.Match("x/y/z"))
		require.False(t, scope.Match("a/b/d"))
	})
	t.Run("match-exclude", func(t *testing.T) {
		scope := NewExclude("a/b/c", "x/y/z")
		require.False(t, scope.Match("a/b/c"))
		require.False(t, scope.Match("x/y/z"))
		require.True(t, scope.Match("a/b/d"))
	})
}

// TestRouteScope_Opposite tests RouteScope Opposite inverts matching mode
// TestRouteScope_Opposite 测试 RouteScope Opposite 反转匹配模式
func TestRouteScope_Opposite(t *testing.T) {
	t.Run("match-include", func(t *testing.T) {
		scope := NewInclude("a/b/c", "x/y/z").Opposite()
		require.False(t, scope.Match("a/b/c"))
		require.False(t, scope.Match("x/y/z"))
		require.True(t, scope.Match("a/b/d"))
	})
	t.Run("match-exclude", func(t *testing.T) {
		scope := NewExclude("a/b/c", "x/y/z").Opposite()
		require.True(t, scope.Match("a/b/c"))
		require.True(t, scope.Match("x/y/z"))
		require.False(t, scope.Match("a/b/d"))
	})
}
