package passkratoseveryn

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestNewMatchFunc(t *testing.T) {
	config := NewConfig(authkratosroutes.NewInclude("a/b/c", "x/y/z"), 3).
		WithFirstMatch(true)
	matchFunc := NewMatchFunc(config, log.DefaultLogger)
	t.Run("case-1", func(t *testing.T) {
		require.True(t, matchFunc(context.Background(), "a/b/c"))
		for i := 0; i < 10; i++ {
			require.False(t, matchFunc(context.Background(), "a/b/c"))
			require.False(t, matchFunc(context.Background(), "a/b/c"))
			require.True(t, matchFunc(context.Background(), "a/b/c"))
		}
	})
	t.Run("case-2", func(t *testing.T) {
		require.True(t, matchFunc(context.Background(), "x/y/z"))
		for i := 0; i < 10; i++ {
			require.False(t, matchFunc(context.Background(), "x/y/z"))
			require.False(t, matchFunc(context.Background(), "x/y/z"))
			require.True(t, matchFunc(context.Background(), "x/y/z"))
		}
	})
}

func TestNewMatchFunc_NotFirstMatch(t *testing.T) {
	config := NewConfig(authkratosroutes.NewInclude("a/b/c", "x/y/z"), 3).
		WithFirstMatch(false)
	matchFunc := NewMatchFunc(config, log.DefaultLogger)
	t.Run("case-1", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			require.False(t, matchFunc(context.Background(), "a/b/c"))
			require.False(t, matchFunc(context.Background(), "a/b/c"))
			require.True(t, matchFunc(context.Background(), "a/b/c"))
		}
	})
	t.Run("case-2", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			require.False(t, matchFunc(context.Background(), "x/y/z"))
			require.False(t, matchFunc(context.Background(), "x/y/z"))
			require.True(t, matchFunc(context.Background(), "x/y/z"))
		}
	})
}
