package authkratostokens

import (
	"context"
	"testing"

	"github.com/orzkratos/authkratos"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)
	m.Run()
}

func TestGetUsername(t *testing.T) {
	ctx := context.Background()
	ctx = setContextWithUsername(ctx, "kratos-username-abc")
	username, ok := GetUsername(ctx)
	require.True(t, ok)
	require.Equal(t, "kratos-username-abc", username)
}
