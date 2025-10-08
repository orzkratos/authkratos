package authkratos

import (
	"github.com/yyle88/zaplog"
	"go.elastic.co/apm/v2"
	"go.uber.org/zap"
)

// CheckApmAgentVersion checks APM package version alignment across modules
// Gives true when versions match, false with warning log on mismatch
// APM as distinct module needs version alignment across dependencies
// Suggest each package using APM implements this check
//
// CheckApmAgentVersion 检查 apm 包版本是否相同
// 版本匹配时返回 true，否则返回 false 并记录警告日志
// apm 作为单独的模块，要求各依赖间版本对齐
// 建议所有使用 apm 的包都实现此检查
func CheckApmAgentVersion(version string) bool {
	if agentVersion := apm.AgentVersion; version != agentVersion {
		zaplog.LOGGER.LOG.Warn("check apm agent versions not match", zap.String("arg_version", version), zap.String("pkg_version", agentVersion))
		return false
	}
	return true
}

// GetApmAgentVersion gets the APM agent version text
// Using this package gives the version stated in go.mod
// When using APM in the project, check version alignment
// Mismatched versions cause runtime logic to fail
//
// GetApmAgentVersion 返回 APM agent 版本号字符串
// 使用本包时，返回 go.mod 中引用的版本号
// 如果项目中直接使用 APM 包，需检查版本是否对齐
// 版本不一致会导致运行时逻辑失败
func GetApmAgentVersion() string {
	return apm.AgentVersion
}
