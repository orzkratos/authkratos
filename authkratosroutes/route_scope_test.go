package authkratosroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
