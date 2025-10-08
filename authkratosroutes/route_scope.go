// Package authkratosroutes: Route scope matching toolkit with middleware selection support
// Provides INCLUDE/EXCLUDE mode route filtering with operation set management
// Enables flexible route-based middleware usage through selection function integration
// Supports APM tracing and debug logging features
//
// authkratosroutes: 路由范围匹配工具包，支持中间件选择器
// 提供 INCLUDE/EXCLUDE 模式的路由过滤和操作集管理
// 通过 selector.MatchFunc 集成实现灵活的基于路由的中间件应用
// 支持 APM 追踪和调试日志功能
package authkratosroutes

import (
	"github.com/orzkratos/authkratos/internal/utils"
	"golang.org/x/exp/maps"
)

// RouteScope defines the scope of routes to match
// RouteScope 定义需要匹配的路由范围
type RouteScope struct {
	Side         SelectSide         // INCLUDE or EXCLUDE mode // 包含或排除模式
	OperationSet map[Operation]bool // Set of operations to match // 需要匹配的操作集合
}

// NewInclude creates a RouteScope that matches only specified operations
// NewInclude 创建仅匹配指定操作的 RouteScope
func NewInclude(operations ...Operation) *RouteScope {
	return &RouteScope{
		Side:         INCLUDE,
		OperationSet: utils.NewSet(operations),
	}
}

// NewExclude creates a RouteScope that matches all except specified operations
// NewExclude 创建排除指定操作后匹配所有其他操作的 RouteScope
func NewExclude(operations ...Operation) *RouteScope {
	return &RouteScope{
		Side:         EXCLUDE,
		OperationSet: utils.NewSet(operations),
	}
}

// Match checks if operation is within the scope
// Match 检查操作是否在范围内
func (c *RouteScope) Match(operation Operation) bool {
	switch c.Side {
	case INCLUDE:
		return c.OperationSet[operation]
	case EXCLUDE:
		return !c.OperationSet[operation]
	default:
		panic("unknown select-side: " + string(c.Side))
	}
}

// Opposite returns a RouteScope with inverted side
// Opposite 返回反转 side 的 RouteScope
func (c *RouteScope) Opposite() *RouteScope {
	switch c.Side {
	case INCLUDE:
		return NewExclude(maps.Keys(c.OperationSet)...)
	case EXCLUDE:
		return NewInclude(maps.Keys(c.OperationSet)...)
	default:
		panic("unknown select-side: " + string(c.Side))
	}
}
