package authkratosroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectSide_Opposite(t *testing.T) {
	require.Equal(t, EXCLUDE, INCLUDE.Opposite())
	require.Equal(t, INCLUDE, EXCLUDE.Opposite())
}

func TestSelectSide_Opposite_Twice(t *testing.T) {
	require.Equal(t, INCLUDE, INCLUDE.Opposite().Opposite())
	require.Equal(t, EXCLUDE, EXCLUDE.Opposite().Opposite())
}

func TestSelectSide_Opposite_Panic(t *testing.T) {
	invalidSide := SelectSide("INVALID")
	require.Panics(t, func() {
		_ = invalidSide.Opposite()
	})
}
