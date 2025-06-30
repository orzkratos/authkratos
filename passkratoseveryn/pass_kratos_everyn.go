package passkratoseveryn

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
	"github.com/yyle88/syncmap"
)

type Config struct {
	selectPath *authkratosroutes.SelectPath
	n          uint32
	matchFirst bool
	debugMode  bool
}

func NewConfig(selectPath *authkratosroutes.SelectPath, n uint32) *Config {
	return &Config{
		selectPath: selectPath,
		n:          n,
		matchFirst: true,
		debugMode:  authkratos.GetDebugMode(),
	}
}

func (c *Config) WithMatchFirst(matchFirst bool) *Config {
	c.matchFirst = matchFirst
	return c
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)
	LOG.Infof("pass-kratos-everyn: new middleware include=%s operations=%d match-first=%v everyn=%v", cfg.selectPath.SelectSide, len(cfg.selectPath.Operations), cfg.matchFirst, cfg.n)
	if cfg.debugMode {
		LOG.Debugf("pass-kratos-everyn: new middleware select-path: %s", neatjsons.S(cfg.selectPath))
	}

	type countBox struct {
		mutex *sync.Mutex
		count uint64
	}
	mp := syncmap.New[authkratosroutes.Path, *countBox]()
	return func(ctx context.Context, operation string) bool {
		if match := cfg.selectPath.Match(operation); !match {
			if cfg.debugMode {
				LOG.Debugf("pass-kratos-everyn: operation=%s include=%v match=%d next -> skip everyn", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
			return false
		}
		value, loaded := mp.LoadOrStore(operation, &countBox{&sync.Mutex{}, 0})
		if !loaded && cfg.matchFirst {
			if cfg.debugMode {
				LOG.Debugf("pass-kratos-everyn: operation=%s include=%v match=%d next -> match first (count=0)", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(true))
			}
			return true
		}
		value.mutex.Lock()
		value.count = (value.count + 1) % uint64(max(cfg.n, 1))
		count := value.count
		value.mutex.Unlock()
		match := count == 0
		if cfg.debugMode {
			if match {
				LOG.Debugf("pass-kratos-everyn: operation=%s include=%v match=%d next -> everyn pass (count=%d)", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match), count)
			} else {
				LOG.Debugf("pass-kratos-everyn: operation=%s include=%v match=%d skip -- everyn skip (count=%d)", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match), count)
			}
		}
		return match
	}
}
