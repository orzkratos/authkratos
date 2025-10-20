package authkratosroutes

// SelectSide represents the matching mode for RouteScope
// SelectSide 表示 RouteScope 的匹配模式
type SelectSide string

const (
	INCLUDE SelectSide = "INCLUDE" // Match specified operations // 仅匹配指定操作
	EXCLUDE SelectSide = "EXCLUDE" // Match except specified operations // 匹配除指定操作外的所有操作
)

// Opposite returns the opposite side
// 返回相反的一侧
func (s SelectSide) Opposite() SelectSide {
	switch s {
	case INCLUDE:
		return EXCLUDE
	case EXCLUDE:
		return INCLUDE
	default:
		panic("unknown select-side: " + string(s))
	}
}
