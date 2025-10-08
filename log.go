// Package authkratos: Kratos authentication middleware collection
// Provides ready-to-use auth middlewares with route scope management and APM tracing support
// Includes token-based auth, random pass-through, periodic throttling, and rate limiting
//
// authkratos: 简单的 Kratos 认证中间件集合
// 提供开箱即用的认证中间件，支持路由范围控制和 APM 追踪
// 包含基于令牌的认证、随机放行、周期性限流和速率限制功能
package authkratos

var debugModeOpen = false

// GetDebugMode gets the current debug mode state
// When on, middlewares output extensive debug logging
//
// GetDebugMode 返回当前调试模式状态
// 启用后，中间件会输出详细的调试日志
func GetDebugMode() bool {
	return debugModeOpen
}

// SetDebugMode sets the global debug mode switch
// Turn on to enable extensive logging in each middleware
//
// SetDebugMode 配置全局调试模式
// 设置成 true 可以在所有中间件中启用详细日志记录
func SetDebugMode(enable bool) {
	debugModeOpen = enable
}
