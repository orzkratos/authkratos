package authkratosroutes

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/orzkratos/authkratos"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)
	m.Run()
}

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
