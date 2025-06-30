package authkratosroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectPath_Match(t *testing.T) {
	t.Run("match-include", func(t *testing.T) {
		sp := NewInclude("a/b/c", "x/y/z")
		require.True(t, sp.Match("a/b/c"))
		require.True(t, sp.Match("x/y/z"))
		require.False(t, sp.Match("a/b/d"))
	})
	t.Run("match-exclude", func(t *testing.T) {
		sp := NewExclude("a/b/c", "x/y/z")
		require.False(t, sp.Match("a/b/c"))
		require.False(t, sp.Match("x/y/z"))
		require.True(t, sp.Match("a/b/d"))
	})
}

func TestSelectPath_Opposite(t *testing.T) {
	t.Run("match-include", func(t *testing.T) {
		sp := NewInclude("a/b/c", "x/y/z").Opposite()
		require.False(t, sp.Match("a/b/c"))
		require.False(t, sp.Match("x/y/z"))
		require.True(t, sp.Match("a/b/d"))
	})
	t.Run("match-exclude", func(t *testing.T) {
		sp := NewExclude("a/b/c", "x/y/z").Opposite()
		require.True(t, sp.Match("a/b/c"))
		require.True(t, sp.Match("x/y/z"))
		require.False(t, sp.Match("a/b/d"))
	})
}
