package authkratosroutes

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
)

type Config struct {
	actionDesc string
	selectPath *SelectPath
	debugMode  bool
}

func NewConfig(actionDesc string, selectPath *SelectPath) *Config {
	return &Config{
		selectPath: selectPath,
		actionDesc: actionDesc,
		debugMode:  authkratos.GetDebugMode(),
	}
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"auth-kratos-routes: new middleware action-desc=%v select-side=%v operations=%v debug-mode=%v",
		cfg.actionDesc,
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
		cfg.debugMode,
	)
	if cfg.debugMode {
		LOG.Debugf("auth-kratos-routes: new middleware action-desc=%v select-path: %s", cfg.actionDesc, neatjsons.S(cfg.selectPath))
	}
	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if cfg.debugMode {
			if match {
				LOG.Debugf("auth-kratos-routes: operation=%s include=%v match=%d next -> %s", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match), cfg.actionDesc)
			} else {
				LOG.Debugf("auth-kratos-routes: operation=%s include=%v match=%d skip -- %s", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match), cfg.actionDesc)
			}
		}
		return match
	}
}
