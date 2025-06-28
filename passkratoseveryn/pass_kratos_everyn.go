package passkratoseveryn

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/yyle88/syncmap"
)

type Config struct {
	selectPath *authkratosroutes.SelectPath
	n          uint32
	firstMatch bool
}

func NewConfig(selectPath *authkratosroutes.SelectPath, n uint32) *Config {
	return &Config{
		selectPath: selectPath,
		n:          n,
		firstMatch: true,
	}
}

func (c *Config) WithFirstMatch(firstMatch bool) *Config {
	c.firstMatch = firstMatch
	return c
}

func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)
	LOG.Infof("new everyn_pass middleware include=%s operations=%v everyn=%v", cfg.selectPath.SelectSide, len(cfg.selectPath.Operations), cfg.n)

	type countBox struct {
		mutex *sync.Mutex
		count uint64
	}
	mp := syncmap.New[authkratosroutes.Path, *countBox]()
	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if !match {
			return false
		}
		value, loaded := mp.LoadOrStore(operation, &countBox{&sync.Mutex{}, 0})
		if !loaded && cfg.firstMatch {
			return true
		}
		value.mutex.Lock()
		value.count = (value.count + 1) % uint64(max(cfg.n, 1))
		count := value.count
		value.mutex.Unlock()
		return count == 0
	}
}
