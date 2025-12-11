package authkratosroutes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSelectSide_Opposite tests Opposite returns opposite side
// TestSelectSide_Opposite 测试 Opposite 返回相反的一侧
func TestSelectSide_Opposite(t *testing.T) {
	require.Equal(t, EXCLUDE, INCLUDE.Opposite())
	require.Equal(t, INCLUDE, EXCLUDE.Opposite())
}

// TestSelectSide_Opposite_Twice tests double Opposite returns original side
// TestSelectSide_Opposite_Twice 测试两次 Opposite 返回原始一侧
func TestSelectSide_Opposite_Twice(t *testing.T) {
	require.Equal(t, INCLUDE, INCLUDE.Opposite().Opposite())
	require.Equal(t, EXCLUDE, EXCLUDE.Opposite().Opposite())
}

// TestSelectSide_Opposite_Panic tests invalid side causes panic
// TestSelectSide_Opposite_Panic 测试无效的 side 导致 panic
func TestSelectSide_Opposite_Panic(t *testing.T) {
	invalidSide := SelectSide("INVALID")
	require.Panics(t, func() {
		_ = invalidSide.Opposite()
	})
}
