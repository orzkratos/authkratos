package authkratosroutes

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
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

func TestSelectPath_NewMatchFunc(t *testing.T) {
	sp := NewInclude("a/b/c", "x/y/z")
	matchFunc := sp.NewMatchFunc("do-something", log.DefaultLogger)
	require.True(t, matchFunc(context.Background(), "a/b/c"))
	require.True(t, matchFunc(context.Background(), "x/y/z"))
	require.False(t, matchFunc(context.Background(), "u/v/w"))
	require.False(t, matchFunc(context.Background(), "r/s/t"))
}
